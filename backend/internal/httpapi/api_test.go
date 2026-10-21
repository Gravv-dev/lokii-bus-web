package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gravv-dev/lokii-bus-web/backend/internal/config"
	"github.com/gravv-dev/lokii-bus-web/backend/internal/store"
)

type testRepository struct {
	repository
	user store.User
}

func (r testRepository) Health(context.Context) error { return nil }

func (r testRepository) SessionUser(context.Context, string) (store.User, error) {
	return r.user, nil
}

func (r testRepository) Login(context.Context, string, string) (store.User, error) {
	return r.user, nil
}

func (testRepository) CreateSession(context.Context, string, string, string, time.Time) error {
	return nil
}

func (r testRepository) ValidateCSRF(context.Context, string, string) (bool, error) {
	return true, nil
}

func (r testRepository) ListProducts(context.Context, int, int, string, string) ([]store.Product, int, error) {
	return []store.Product{}, 0, nil
}

func (r testRepository) AdminOrders(context.Context, int, int) ([]store.Order, int, error) {
	return []store.Order{}, 0, nil
}

func testHandler(role string) http.Handler {
	gin.SetMode(gin.TestMode)
	return New(config.Config{AllowedOrigins: []string{"http://localhost:3000"}}, testRepository{
		user: store.User{ID: "d9428888-122b-11e1-b85c-61cd3cbb3210", Role: role},
	}).Handler()
}

func TestHealthAndProductList(t *testing.T) {
	for _, path := range []string{"/health", "/api/v1/products?page=1&page_size=20"} {
		response := httptest.NewRecorder()
		testHandler("customer").ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want %d: %s", path, response.Code, http.StatusOK, response.Body.String())
		}
	}
}

func TestCORSAllowsOnlyConfiguredOrigin(t *testing.T) {
	handler := testHandler("customer")
	for _, test := range []struct {
		origin string
		status int
	}{
		{"http://localhost:3000", http.StatusOK},
		{"https://untrusted.example", http.StatusForbidden},
	} {
		request := httptest.NewRequest(http.MethodGet, "/health", nil)
		request.Header.Set("Origin", test.origin)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != test.status {
			t.Errorf("origin %q status = %d, want %d", test.origin, response.Code, test.status)
		}
		if test.status == http.StatusOK && response.Header().Get("Access-Control-Allow-Origin") != test.origin {
			t.Errorf("allowed origin response header = %q", response.Header().Get("Access-Control-Allow-Origin"))
		}
	}
}

func TestAdminRoutesRejectCustomer(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/products", strings.NewReader("{}"))
	request.AddCookie(&http.Cookie{Name: "session", Value: "opaque-session"})
	response := httptest.NewRecorder()
	testHandler("customer").ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusForbidden, response.Body.String())
	}
}

func TestAdminRouteRequiresAuthentication(t *testing.T) {
	response := httptest.NewRecorder()
	testHandler("admin").ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/admin/orders", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusUnauthorized, response.Body.String())
	}
}

func TestAdminReadRouteDoesNotRequireCSRF(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/orders", nil)
	request.AddCookie(&http.Cookie{Name: "session", Value: "opaque-session"})
	response := httptest.NewRecorder()
	testHandler("admin").ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
}

func TestAuthRateLimit(t *testing.T) {
	handler := testHandler("customer")
	for i := 0; i < 11; i++ {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader("{}"))
		request.RemoteAddr = "192.0.2.8:1234"
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if i < 10 && response.Code == http.StatusTooManyRequests {
			t.Fatalf("request %d unexpectedly rate limited", i+1)
		}
		if i == 10 && response.Code != http.StatusTooManyRequests {
			t.Fatalf("request %d status = %d, want %d", i+1, response.Code, http.StatusTooManyRequests)
		}
	}
}

func TestCSRFRejectsMissingToken(t *testing.T) {
	handler := testHandler("customer")
	request := httptest.NewRequest(http.MethodPost, "/api/v1/cart/items", strings.NewReader(`{"product_id":"d9428888-122b-11e1-b85c-61cd3cbb3210","quantity":1}`))
	request.AddCookie(&http.Cookie{Name: "session", Value: "opaque-session"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusForbidden, response.Body.String())
	}
}

func TestLoginCreatesHttpOnlySessionCookie(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		strings.NewReader(`{"email":"baker@example.com","password":"this-is-a-long-password"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	testHandler("customer").ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	var sessionCookie, csrfCookie *http.Cookie
	for _, cookie := range response.Result().Cookies() {
		switch cookie.Name {
		case "session":
			sessionCookie = cookie
		case "csrf_token":
			csrfCookie = cookie
		}
	}
	if sessionCookie == nil || !sessionCookie.HttpOnly || sessionCookie.Value == "" {
		t.Fatal("expected a non-empty HttpOnly session cookie")
	}
	if csrfCookie == nil || csrfCookie.HttpOnly || csrfCookie.Value == "" {
		t.Fatal("expected a readable CSRF cookie")
	}
}

func TestPublicRegistrationCannotSetRole(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register",
		strings.NewReader(`{"email":"baker@example.com","name":"Baker","password":"long-password-123","role":"admin"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	testHandler("customer").ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusBadRequest, response.Body.String())
	}
}
