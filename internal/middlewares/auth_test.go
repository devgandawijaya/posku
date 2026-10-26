package middlewares

import (
	"net/http"
	"net/http/httptest"
	"posku/internal/config"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func createTestToken(claims jwt.MapClaims) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(config.JWTSecret))
	return tokenString
}

func TestAuthRequired(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config.JWTSecret = "test-secret-key-123"

	r := gin.New()
	r.Use(AuthRequired())
	r.GET("/protected", func(c *gin.Context) {
		companyID, _ := c.Get("company_id")
		role, _ := c.Get("employee_role")
		c.JSON(http.StatusOK, gin.H{
			"company_id": companyID,
			"role":       role,
		})
	})

	t.Run("missing header returns 401", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/protected", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("invalid token format returns 401", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "InvalidFormat")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("valid token sets context claims", func(t *testing.T) {
		tokenStr := createTestToken(jwt.MapClaims{
			"employee_id":   1,
			"employee_name": "Budi",
			"role":          "kasir",
			"company_id":    float64(42),
			"permissions":   []interface{}{"produk.read", "transaksi.*"},
			"exp":           time.Now().Add(time.Hour).Unix(),
		})

		req, _ := http.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
		}
	})
}

func TestRequirePermission(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		role           string
		permissions    []string
		required       string
		expectedStatus int
	}{
		{
			name:           "admin bypasses all permissions",
			role:           "admin",
			permissions:    nil,
			required:       "any.permission",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "user with exact permission succeeds",
			role:           "staff",
			permissions:    []string{"produk.read", "produk.create"},
			required:       "produk.create",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "user with wildcard module permission succeeds",
			role:           "staff",
			permissions:    []string{"produk.*"},
			required:       "produk.delete",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "user with global wildcard succeeds",
			role:           "manager",
			permissions:    []string{"*"},
			required:       "gudang.create",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "user without permission gets 403",
			role:           "kasir",
			permissions:    []string{"produk.read"},
			required:       "produk.delete",
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(func(c *gin.Context) {
				c.Set("employee_role", tt.role)
				c.Set("permissions", tt.permissions)
				c.Next()
			})
			r.GET("/test", RequirePermission(tt.required), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req, _ := http.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestRequireAnyPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("employee_role", "kasir")
		c.Set("permissions", []string{"transaksi.read"})
		c.Next()
	})
	r.GET("/test", RequireAnyPermission("laporan.read", "transaksi.read"), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

