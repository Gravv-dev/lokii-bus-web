package httpapi

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gravv-dev/lokii-bus-web/backend/internal/store"
)

type registerRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (a *API) register(c *gin.Context) {
	var input registerRequest
	if !decodeJSON(c, &input) {
		return
	}
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.Name = strings.TrimSpace(input.Name)
	address, err := mail.ParseAddress(input.Email)
	if err != nil || address.Address != input.Email || len(input.Email) > 254 {
		writeError(c, http.StatusBadRequest, "a valid email is required")
		return
	}
	if input.Name == "" || len([]rune(input.Name)) > 100 {
		writeError(c, http.StatusBadRequest, "name must be between 1 and 100 characters")
		return
	}
	if len(input.Password) < 12 || len(input.Password) > 72 {
		writeError(c, http.StatusBadRequest, "password must be between 12 and 72 bytes")
		return
	}
	user, err := a.store.Register(requestContext(c), input.Email, input.Name, input.Password)
	if err != nil {
		if errors.Is(err, store.ErrConflict) {
			writeError(c, http.StatusConflict, "an account with that email already exists")
			return
		}
		writeInternalError(c, err, "could not create account")
		return
	}
	a.createSessionResponse(c, user, http.StatusCreated)
}

func (a *API) login(c *gin.Context) {
	var input loginRequest
	if !decodeJSON(c, &input) {
		return
	}
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	address, addressErr := mail.ParseAddress(input.Email)
	if addressErr != nil || address.Address != input.Email || len(input.Password) == 0 || len(input.Password) > 72 {
		writeError(c, http.StatusUnauthorized, "email or password is incorrect")
		return
	}
	user, err := a.store.Login(requestContext(c), input.Email, input.Password)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(c, http.StatusUnauthorized, "email or password is incorrect")
			return
		}
		writeInternalError(c, err, "could not authenticate")
		return
	}
	a.createSessionResponse(c, user, http.StatusOK)
}

func (a *API) createSessionResponse(c *gin.Context, user store.User, status int) {
	token, err := randomToken()
	if err != nil {
		writeInternalError(c, err, "could not create session")
		return
	}
	csrf, err := randomToken()
	if err != nil {
		writeInternalError(c, err, "could not create session")
		return
	}
	expires := time.Now().Add(14 * 24 * time.Hour)
	if err := a.store.CreateSession(requestContext(c), user.ID, token, csrf, expires); err != nil {
		writeInternalError(c, err, "could not create session")
		return
	}
	a.setSessionCookies(c, token, csrf)
	c.JSON(status, gin.H{"data": user, "csrf_token": csrf})
}

func randomToken() (string, error) {
	data := make([]byte, 32)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func (a *API) logout(c *gin.Context) {
	token := c.GetString("session_token")
	if err := a.store.DeleteSession(requestContext(c), token); err != nil {
		writeInternalError(c, err, "could not end session")
		return
	}
	a.clearSessionCookies(c)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"logged_out": true}})
}

func (a *API) me(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": principal(c)})
}

func (a *API) listProducts(c *gin.Context) {
	page, size, ok := pageParams(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "page must be positive and page_size must be between 1 and 100")
		return
	}
	categoryID := strings.TrimSpace(c.Query("category_id"))
	if categoryID != "" && !uuidPattern.MatchString(categoryID) {
		writeError(c, http.StatusBadRequest, "category_id must be a UUID")
		return
	}
	search := normalizeSearch(c.Query("search"))
	if len(search) > 100 {
		writeError(c, http.StatusBadRequest, "search must be at most 100 characters")
		return
	}
	products, total, err := a.store.ListProducts(requestContext(c), page, size, categoryID, search)
	if err != nil {
		writeInternalError(c, err, "could not list products")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": products,
		"pagination": gin.H{
			"page": page, "page_size": size, "total": total,
			"total_pages": int(math.Ceil(float64(total) / float64(size))),
		},
	})
}

func (a *API) getProduct(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "product id must be a UUID")
		return
	}
	product, err := a.store.GetProduct(requestContext(c), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(c, http.StatusNotFound, "product not found")
		return
	}
	if err != nil {
		writeInternalError(c, err, "could not load product")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": product})
}

func (a *API) listCategories(c *gin.Context) {
	categories, err := a.store.Categories(requestContext(c))
	if err != nil {
		writeInternalError(c, err, "could not list categories")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": categories})
}

func (a *API) getCart(c *gin.Context) {
	cart, err := a.store.Cart(requestContext(c), principal(c).ID)
	if err != nil {
		writeInternalError(c, err, "could not load cart")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": cart})
}

type cartItemRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type quantityRequest struct {
	Quantity int `json:"quantity"`
}

func (a *API) addCartItem(c *gin.Context) {
	var input cartItemRequest
	if !decodeJSON(c, &input) {
		return
	}
	if !uuidPattern.MatchString(input.ProductID) || input.Quantity < 1 || input.Quantity > 99 {
		writeError(c, http.StatusBadRequest, "product_id must be a UUID and quantity must be between 1 and 99")
		return
	}
	if err := a.store.AddCartItem(requestContext(c), principal(c).ID, input.ProductID, input.Quantity); err != nil {
		writeStoreError(c, err, "could not add item to cart")
		return
	}
	a.getCart(c)
}

func (a *API) updateCartItem(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "cart item id must be a UUID")
		return
	}
	var input quantityRequest
	if !decodeJSON(c, &input) {
		return
	}
	if input.Quantity < 1 || input.Quantity > 99 {
		writeError(c, http.StatusBadRequest, "quantity must be between 1 and 99")
		return
	}
	if err := a.store.UpdateCartItem(requestContext(c), principal(c).ID, id, input.Quantity); err != nil {
		writeStoreError(c, err, "could not update cart item")
		return
	}
	a.getCart(c)
}

func (a *API) deleteCartItem(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "cart item id must be a UUID")
		return
	}
	if err := a.store.DeleteCartItem(requestContext(c), principal(c).ID, id); err != nil {
		writeStoreError(c, err, "could not remove cart item")
		return
	}
	a.getCart(c)
}

func (a *API) createOrder(c *gin.Context) {
	order, err := a.store.CreateOrder(requestContext(c), principal(c).ID)
	if err != nil {
		writeStoreError(c, err, "could not create order")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": order})
}

func (a *API) listOrders(c *gin.Context) {
	page, size, ok := pageParams(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "page must be positive and page_size must be between 1 and 100")
		return
	}
	orders, total, err := a.store.Orders(requestContext(c), principal(c).ID, "", page, size)
	if err != nil {
		writeInternalError(c, err, "could not list orders")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": orders, "pagination": gin.H{"page": page, "page_size": size, "total": total}})
}

func (a *API) getOrder(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "order id must be a UUID")
		return
	}
	orders, _, err := a.store.Orders(requestContext(c), principal(c).ID, id, 1, 1)
	if err != nil {
		writeInternalError(c, err, "could not load order")
		return
	}
	if len(orders) == 0 {
		writeError(c, http.StatusNotFound, "order not found")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": orders[0]})
}

func (a *API) createProduct(c *gin.Context) {
	var input store.ProductInput
	if !decodeJSON(c, &input) || !validateProduct(c, input) {
		return
	}
	product, err := a.store.AdminCreateProduct(requestContext(c), input, int64(math.Round(input.Price*100)))
	if err != nil {
		writeStoreError(c, err, "could not create product")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": product})
}

func (a *API) updateProduct(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "product id must be a UUID")
		return
	}
	var input store.ProductInput
	if !decodeJSON(c, &input) || !validateProduct(c, input) {
		return
	}
	product, err := a.store.AdminUpdateProduct(requestContext(c), id, input, int64(math.Round(input.Price*100)))
	if err != nil {
		writeStoreError(c, err, "could not update product")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": product})
}

func validateProduct(c *gin.Context, input store.ProductInput) bool {
	if strings.TrimSpace(input.Name) == "" || len([]rune(input.Name)) > 160 ||
		!slugPattern.MatchString(input.Slug) || len(input.Description) > 4000 ||
		!validImageURL(input.ImageURL) || input.Price <= 0 || input.Price > 100000 ||
		math.IsNaN(input.Price) || math.IsInf(input.Price, 0) ||
		input.Inventory < 0 || input.Inventory > 1000000 {
		writeError(c, http.StatusBadRequest, "product fields are invalid")
		return false
	}
	if input.CategoryID != nil && !uuidPattern.MatchString(*input.CategoryID) {
		writeError(c, http.StatusBadRequest, "category_id must be a UUID")
		return false
	}
	return true
}

func validImageURL(imageURL string) bool {
	if imageURL == "" {
		return true
	}
	if len(imageURL) > 2048 {
		return false
	}
	if strings.HasPrefix(imageURL, "/images/") {
		return !strings.Contains(imageURL, "..") && !strings.ContainsAny(imageURL, `\?#`)
	}
	parsed, err := url.ParseRequestURI(imageURL)
	return err == nil && parsed.Scheme == "https" && parsed.Host != "" && parsed.User == nil
}

func (a *API) deleteProduct(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "product id must be a UUID")
		return
	}
	if err := a.store.AdminDeleteProduct(requestContext(c), id); err != nil {
		writeStoreError(c, err, "could not delete product")
		return
	}
	c.Status(http.StatusNoContent)
}

func (a *API) createCategory(c *gin.Context) {
	var input store.CategoryInput
	if !decodeJSON(c, &input) || !validateCategory(c, input) {
		return
	}
	category, err := a.store.AdminCreateCategory(requestContext(c), input)
	if err != nil {
		writeStoreError(c, err, "could not create category")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": category})
}

func (a *API) updateCategory(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "category id must be a UUID")
		return
	}
	var input store.CategoryInput
	if !decodeJSON(c, &input) || !validateCategory(c, input) {
		return
	}
	category, err := a.store.AdminUpdateCategory(requestContext(c), id, input)
	if err != nil {
		writeStoreError(c, err, "could not update category")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": category})
}

func validateCategory(c *gin.Context, input store.CategoryInput) bool {
	if strings.TrimSpace(input.Name) == "" || len([]rune(input.Name)) > 100 ||
		!slugPattern.MatchString(input.Slug) || len(input.Description) > 2000 {
		writeError(c, http.StatusBadRequest, "category fields are invalid")
		return false
	}
	return true
}

func (a *API) deleteCategory(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "category id must be a UUID")
		return
	}
	if err := a.store.AdminDeleteCategory(requestContext(c), id); err != nil {
		writeStoreError(c, err, "could not delete category")
		return
	}
	c.Status(http.StatusNoContent)
}

func (a *API) setInventory(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "product id must be a UUID")
		return
	}
	var input quantityRequest
	if !decodeJSON(c, &input) {
		return
	}
	if input.Quantity < 0 || input.Quantity > 1000000 {
		writeError(c, http.StatusBadRequest, "quantity must be between 0 and 1000000")
		return
	}
	if err := a.store.AdminSetInventory(requestContext(c), id, input.Quantity); err != nil {
		writeStoreError(c, err, "could not update inventory")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"product_id": id, "quantity": input.Quantity}})
}

func (a *API) adminOrders(c *gin.Context) {
	page, size, ok := pageParams(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "page must be positive and page_size must be between 1 and 100")
		return
	}
	orders, total, err := a.store.AdminOrders(requestContext(c), page, size)
	if err != nil {
		writeInternalError(c, err, "could not list orders")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": orders, "pagination": gin.H{"page": page, "page_size": size, "total": total}})
}

type statusRequest struct {
	Status string `json:"status"`
}

func (a *API) updateOrderStatus(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "order id must be a UUID")
		return
	}
	var input statusRequest
	if !decodeJSON(c, &input) {
		return
	}
	switch input.Status {
	case "pending", "processing", "shipped", "completed", "cancelled":
	default:
		writeError(c, http.StatusBadRequest, "order status is invalid")
		return
	}
	order, err := a.store.AdminUpdateOrderStatus(requestContext(c), id, input.Status)
	if err != nil {
		writeStoreError(c, err, "could not update order status")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": order})
}

func (a *API) salesReport(c *gin.Context) {
	report, err := a.store.AdminSalesReport(requestContext(c))
	if err != nil {
		writeInternalError(c, err, "could not load sales report")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": report})
}

func principal(c *gin.Context) store.User {
	value, _ := c.Get(principalKey{}.String())
	return value.(store.User)
}

func parseBoundedInt(raw string, fallback, min, max int) (int, error) {
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < min || value > max {
		return 0, fmt.Errorf("integer must be between %d and %d", min, max)
	}
	return value, nil
}
