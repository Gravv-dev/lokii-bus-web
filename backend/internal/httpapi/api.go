package httpapi

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gravv-dev/lokii-bus-web/backend/internal/config"
	"github.com/gravv-dev/lokii-bus-web/backend/internal/store"
)

type repository interface {
	Health(ctx context.Context) error
	Register(ctx context.Context, email, name, password string) (store.User, error)
	Login(ctx context.Context, email, password string) (store.User, error)
	CreateSession(ctx context.Context, userID, token, csrf string, expires time.Time) error
	SessionUser(ctx context.Context, token string) (store.User, error)
	ValidateCSRF(ctx context.Context, token, csrf string) (bool, error)
	DeleteSession(ctx context.Context, token string) error
	ListProducts(ctx context.Context, page, pageSize int, categoryID, search string) ([]store.Product, int, error)
	GetProduct(ctx context.Context, id string) (store.Product, error)
	Categories(ctx context.Context) ([]store.Category, error)
	Cart(ctx context.Context, userID string) (store.Cart, error)
	AddCartItem(ctx context.Context, userID, productID string, quantity int) error
	UpdateCartItem(ctx context.Context, userID, itemID string, quantity int) error
	DeleteCartItem(ctx context.Context, userID, itemID string) error
	CreateOrder(ctx context.Context, userID string) (store.Order, error)
	Orders(ctx context.Context, userID, orderID string, page, pageSize int) ([]store.Order, int, error)
	AdminCreateProduct(ctx context.Context, input store.ProductInput, cents int64) (store.Product, error)
	AdminUpdateProduct(ctx context.Context, id string, input store.ProductInput, cents int64) (store.Product, error)
	AdminDeleteProduct(ctx context.Context, id string) error
	AdminCreateCategory(ctx context.Context, input store.CategoryInput) (store.Category, error)
	AdminUpdateCategory(ctx context.Context, id string, input store.CategoryInput) (store.Category, error)
	AdminDeleteCategory(ctx context.Context, id string) error
	AdminSetInventory(ctx context.Context, productID string, quantity int) error
	AdminOrders(ctx context.Context, page, pageSize int) ([]store.Order, int, error)
	AdminUpdateOrderStatus(ctx context.Context, id, status string) (store.Order, error)
	AdminSalesReport(ctx context.Context) (store.SalesReport, error)
}

type API struct {
	cfg   config.Config
	store repository
}

type principalKey struct{}

func New(cfg config.Config, repository repository) *API {
	return &API{cfg: cfg, store: repository}
}

func (a *API) Handler() http.Handler {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.SetTrustedProxies(nil)
	engine.Use(gin.Logger(), gin.Recovery(), a.cors())
	engine.GET("/health", a.health)

	v1 := engine.Group("/api/v1")
	auth := v1.Group("/auth")
	auth.Use(a.authRateLimit())
	auth.POST("/register", a.register)
	auth.POST("/login", a.login)

	protected := v1.Group("")
	protected.Use(a.authenticate())
	protected.POST("/auth/logout", a.csrf(), a.logout)
	protected.GET("/auth/me", a.me)
	protected.GET("/cart", a.getCart)
	protected.POST("/cart/items", a.csrf(), a.addCartItem)
	protected.PATCH("/cart/items/:id", a.csrf(), a.updateCartItem)
	protected.DELETE("/cart/items/:id", a.csrf(), a.deleteCartItem)
	protected.POST("/orders", a.csrf(), a.createOrder)
	protected.GET("/orders", a.listOrders)
	protected.GET("/orders/:id", a.getOrder)

	v1.GET("/products", a.listProducts)
	v1.GET("/products/:id", a.getProduct)
	v1.GET("/categories", a.listCategories)

	admin := protected.Group("/admin")
	admin.Use(a.requireAdmin())
	admin.POST("/products", a.csrf(), a.createProduct)
	admin.PUT("/products/:id", a.csrf(), a.updateProduct)
	admin.DELETE("/products/:id", a.csrf(), a.deleteProduct)
	admin.POST("/categories", a.csrf(), a.createCategory)
	admin.PUT("/categories/:id", a.csrf(), a.updateCategory)
	admin.DELETE("/categories/:id", a.csrf(), a.deleteCategory)
	admin.PATCH("/products/:id/inventory", a.csrf(), a.setInventory)
	admin.GET("/orders", a.adminOrders)
	admin.PATCH("/orders/:id/status", a.csrf(), a.updateOrderStatus)
	admin.GET("/reports/sales", a.salesReport)
	return engine
}

func (a *API) cors() gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(a.cfg.AllowedOrigins))
	for _, origin := range a.cfg.AllowedOrigins {
		allowed[origin] = struct{}{}
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			if _, ok := allowed[origin]; !ok {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "origin is not allowed"})
				return
			}
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, X-CSRF-Token")
			c.Header("Access-Control-Max-Age", "600")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

type authAttempt struct {
	started time.Time
	count   int
}

func (a *API) authRateLimit() gin.HandlerFunc {
	var mu sync.Mutex
	attempts := make(map[string]authAttempt)
	return func(c *gin.Context) {
		now := time.Now()
		ip := c.ClientIP()
		mu.Lock()
		for key, attempt := range attempts {
			if now.Sub(attempt.started) >= time.Minute {
				delete(attempts, key)
			}
		}
		attempt := attempts[ip]
		if attempt.started.IsZero() || now.Sub(attempt.started) >= time.Minute {
			attempt = authAttempt{started: now}
		}
		attempt.count++
		overLimit := attempt.count > 10 || (len(attempts) >= 10000 && attempts[ip].started.IsZero())
		if !overLimit {
			attempts[ip] = attempt
		}
		mu.Unlock()
		if overLimit {
			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many authentication attempts"})
			return
		}
		c.Next()
	}
}

func requestContext(c *gin.Context) context.Context {
	return c.Request.Context()
}

func (a *API) health(c *gin.Context) {
	if err := a.store.Health(requestContext(c)); err != nil {
		log.Printf("database health check: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (a *API) authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Request.Cookie("session")
		if err != nil || cookie.Value == "" {
			writeError(c, http.StatusUnauthorized, "authentication required")
			return
		}
		user, err := a.store.SessionUser(requestContext(c), cookie.Value)
		if err != nil {
			if !errors.Is(err, store.ErrNotFound) {
				log.Printf("resolve session user: %v", err)
				writeError(c, http.StatusInternalServerError, "could not authenticate")
				return
			}
			writeError(c, http.StatusUnauthorized, "authentication required")
			return
		}
		c.Set(principalKey{}.String(), user)
		c.Set("session_token", cookie.Value)
		c.Next()
	}
}

func (principalKey) String() string {
	return "principal"
}

func (a *API) requireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		value, ok := c.Get(principalKey{}.String())
		user, valid := value.(store.User)
		if !ok || !valid || user.Role != "admin" {
			writeError(c, http.StatusForbidden, "administrator access required")
			return
		}
		c.Next()
	}
}

func (a *API) csrf() gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Request.Cookie("csrf_token")
		header := c.GetHeader("X-CSRF-Token")
		if err != nil || cookie.Value == "" || header == "" || cookie.Value != header {
			writeError(c, http.StatusForbidden, "CSRF token is missing or invalid")
			return
		}
		token, _ := c.Get("session_token")
		valid, err := a.store.ValidateCSRF(requestContext(c), token.(string), header)
		if err != nil {
			writeInternalError(c, err, "could not validate request")
			return
		}
		if !valid {
			writeError(c, http.StatusForbidden, "CSRF token is missing or invalid")
			return
		}
		c.Next()
	}
}

func (a *API) setSessionCookies(c *gin.Context, token, csrf string) {
	maxAge := int((14 * 24 * time.Hour).Seconds())
	sameSite := http.SameSiteLaxMode
	if a.cfg.CookieSecure {
		sameSite = http.SameSiteNoneMode
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name: "session", Value: token, Path: "/", Domain: a.cfg.CookieDomain,
		MaxAge: maxAge, HttpOnly: true, Secure: a.cfg.CookieSecure, SameSite: sameSite,
	})
	http.SetCookie(c.Writer, &http.Cookie{
		Name: "csrf_token", Value: csrf, Path: "/", Domain: a.cfg.CookieDomain,
		MaxAge: maxAge, HttpOnly: false, Secure: a.cfg.CookieSecure, SameSite: sameSite,
	})
}

func (a *API) clearSessionCookies(c *gin.Context) {
	sameSite := http.SameSiteLaxMode
	if a.cfg.CookieSecure {
		sameSite = http.SameSiteNoneMode
	}
	for _, name := range []string{"session", "csrf_token"} {
		http.SetCookie(c.Writer, &http.Cookie{
			Name: name, Value: "", Path: "/", Domain: a.cfg.CookieDomain,
			MaxAge: -1, HttpOnly: name == "session", Secure: a.cfg.CookieSecure, SameSite: sameSite,
		})
	}
}

func pageParams(c *gin.Context) (int, int, bool) {
	page, pageErr := parseBoundedInt(c.DefaultQuery("page", "1"), 1, 1, 100000)
	size, sizeErr := parseBoundedInt(c.DefaultQuery("page_size", "20"), 20, 1, 100)
	return page, size, pageErr == nil && sizeErr == nil
}

func pathID(c *gin.Context) (string, bool) {
	id := c.Param("id")
	return id, uuidPattern.MatchString(id)
}

func normalizeSearch(v string) string {
	return strings.TrimSpace(v)
}
