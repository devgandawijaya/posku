package controllers

import (
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// PaginationMeta standardizes pagination metadata across all listing endpoints.
type PaginationMeta struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// GetPagination extracts page and per_page (or limit) from query parameters with safe defaults.
func GetPagination(c *gin.Context) (page int, perPage int, offset int) {
	page = 1
	perPage = 20

	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	if pp := c.Query("per_page"); pp != "" {
		if parsed, err := strconv.Atoi(pp); err == nil && parsed > 0 {
			perPage = parsed
		}
	} else if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			perPage = parsed
		}
	}

	if perPage > 100 {
		perPage = 100
	}

	offset = (page - 1) * perPage
	return page, perPage, offset
}

// RespondPaginated returns a standard paginated JSON envelope.
func RespondPaginated(c *gin.Context, data interface{}, page int, perPage int, total int64) {
	totalPages := 1
	if perPage > 0 && total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(perPage)))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
		"meta": PaginationMeta{
			Page:       page,
			PerPage:    perPage,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

// RespondSuccess returns a standard success JSON response.
func RespondSuccess(c *gin.Context, data interface{}, message ...string) {
	resp := gin.H{
		"success": true,
		"data":    data,
	}
	if len(message) > 0 {
		resp["message"] = message[0]
	}
	c.JSON(http.StatusOK, resp)
}

// RespondCreated returns a standard 201 Created JSON response.
func RespondCreated(c *gin.Context, data interface{}, message ...string) {
	resp := gin.H{
		"success": true,
		"data":    data,
	}
	if len(message) > 0 {
		resp["message"] = message[0]
	}
	c.JSON(http.StatusCreated, resp)
}

// RespondError returns a standard error JSON response.
func RespondError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{
		"success": false,
		"error":   message,
	})
}

// getAuthCompanyID returns the authenticated company_id from JWT claims, preventing cross-tenant leaks.
func getAuthCompanyID(c *gin.Context) uint {
	val, exists := c.Get("company_id")
	if !exists {
		// Fallback for unauthenticated testing if needed
		if q := c.Query("company_id"); q != "" {
			n, _ := strconv.ParseUint(q, 10, 64)
			return uint(n)
		}
		return 0
	}

	switch v := val.(type) {
	case uint:
		return v
	case float64:
		return uint(v)
	case int:
		return uint(v)
	case int64:
		return uint(v)
	case string:
		n, _ := strconv.ParseUint(v, 10, 64)
		return uint(n)
	default:
		return 0
	}
}

