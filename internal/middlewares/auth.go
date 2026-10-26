package middlewares

import (
	"net/http"
	"posku/internal/config"
	"strings"

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

			// Parse company_id safely as uint
			var companyID uint
			if rawCID, exists := claims["company_id"]; exists {
				if f, ok := rawCID.(float64); ok {
					companyID = uint(f)
				}
			}
			c.Set("company_id", companyID)

			// Extract permissions slice
			var perms []string
			if rawPerms, exists := claims["permissions"]; exists {
				if permList, ok := rawPerms.([]interface{}); ok {
					for _, p := range permList {
						if s, ok := p.(string); ok {
							perms = append(perms, s)
						}
					}
				} else if permStrings, ok := rawPerms.([]string); ok {
					perms = permStrings
				}
			}
			c.Set("permissions", perms)
		}

		c.Next()
	}
}

// hasPermission checks whether the permissions list grants the required action.
func hasPermission(perms []string, required string) bool {
	reqModule := ""
	if parts := strings.Split(required, "."); len(parts) == 2 {
		reqModule = parts[0]
	}

	for _, p := range perms {
		if p == "*" || p == required {
			return true
		}
		if reqModule != "" && (p == reqModule+".*" || p == reqModule) {
			return true
		}
	}
	return false
}

// RequirePermission ensures the user is an admin OR has the granular permission code (e.g. "produk.create").
func RequirePermission(required string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get("employee_role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "role not found in token"})
			return
		}

		role, _ := roleVal.(string)
		if role == "admin" {
			c.Next()
			return
		}

		permsVal, _ := c.Get("permissions")
		perms, _ := permsVal.([]string)

		if hasPermission(perms, required) {
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": "insufficient permissions: required " + required,
		})
	}
}

// RequireAnyPermission allows access if user is admin OR possesses at least one of the listed permissions.
func RequireAnyPermission(required ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get("employee_role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "role not found in token"})
			return
		}

		role, _ := roleVal.(string)
		if role == "admin" {
			c.Next()
			return
		}

		permsVal, _ := c.Get("permissions")
		perms, _ := permsVal.([]string)

		for _, req := range required {
			if hasPermission(perms, req) {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": "insufficient permissions",
		})
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
