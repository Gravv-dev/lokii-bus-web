package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gravv-dev/lokii-bus-web/backend/internal/store"
	"github.com/jackc/pgx/v5/pgconn"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-8][0-9a-fA-F]{3}-[89aAbB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)
var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func decodeJSON(c *gin.Context, destination any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		writeError(c, http.StatusBadRequest, "request body must be valid JSON")
		return false
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		writeError(c, http.StatusBadRequest, "request body must contain a single JSON value")
		return false
	}
	return true
}

func writeError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}

func writeInternalError(c *gin.Context, err error, message string) {
	log.Printf("%s: %v", message, err)
	writeError(c, http.StatusInternalServerError, message)
}

func writeStoreError(c *gin.Context, err error, fallback string) {
	var pgError *pgconn.PgError
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(c, http.StatusNotFound, "resource not found")
	case errors.Is(err, store.ErrInsufficientStock):
		writeError(c, http.StatusConflict, "requested quantity is not available")
	case errors.Is(err, store.ErrQuantityLimit):
		writeError(c, http.StatusConflict, "cart item quantity cannot exceed 99")
	case errors.Is(err, store.ErrEmptyCart):
		writeError(c, http.StatusConflict, "cart is empty")
	case errors.Is(err, store.ErrConflict) || strings.Contains(err.Error(), "duplicate key"):
		writeError(c, http.StatusConflict, "resource conflicts with an existing record")
	case errors.Is(err, store.ErrInvalidTransition):
		writeError(c, http.StatusConflict, "order status transition is not allowed")
	case errors.As(err, &pgError) && pgError.Code == "23505":
		writeError(c, http.StatusConflict, "resource conflicts with an existing record")
	case errors.As(err, &pgError) && pgError.Code == "23503":
		writeError(c, http.StatusNotFound, "related resource not found")
	default:
		writeInternalError(c, err, fallback)
	}
}
