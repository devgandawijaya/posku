package middlewares

import (
	"net/http"
	"posku/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization required"})
			return
		}

		// expect Bearer <token>
		var tokenString string
		if len(auth) > 7 && auth[:7] == "Bearer " {
			tokenString = auth[7:]
		} else {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header"})
			return
		}

		tok, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			return []byte(config.JWTSecret), nil
		})
		if err != nil || !tok.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		if claims, ok := tok.Claims.(jwt.MapClaims); ok {
			c.Set("employee_id", claims["employee_id"])
			c.Set("employee_name", claims["employee_name"])
			c.Set("employee_role", claims["role"])
			c.Set("company_id", claims["company_id"])
			c.Set("permissions", claims["permissions"])
		}

		c.Next()
	}
}

// RequireRoles returns a middleware that ensures the authenticated user has one of the allowed roles.
func RequireRoles(allowed ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get("employee_role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "role not found"})
			return
		}

		role, ok := roleVal.(string)
		if !ok {
			// some tokens may have string values stored differently
			if m, ok := roleVal.(map[string]interface{}); ok {
				if v, ok := m["role"].(string); ok {
					role = v
				}
			}
		}

		for _, a := range allowed {
			if role == a {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
	}
}
