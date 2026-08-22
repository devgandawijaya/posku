package middlewares

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestJSONResponseMiddlewareWrapsSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(JSONResponseMiddleware())
	router.GET("/products/1", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"id": 1, "name": "Indomie"})
	})

	response := performRequest(router, http.MethodGet, "/products/1")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	body := decodeEnvelope(t, response)
	if body["success"] != true || body["version"] != "v1" || body["error"] != nil {
		t.Fatalf("unexpected success envelope: %#v", body)
	}
	if _, err := time.Parse(time.RFC3339, body["timestamp"].(string)); err != nil {
		t.Fatalf("timestamp is not RFC3339: %v", err)
	}
	if body["requestId"] == "" || response.Header().Get("X-Request-ID") != body["requestId"] {
		t.Fatalf("request ID is missing or differs from header: %#v", body)
	}
}

func TestJSONResponseMiddlewareWrapsError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(JSONResponseMiddleware())
	router.GET("/products/1", func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
	})

	response := performRequest(router, http.MethodGet, "/products/1")
	body := decodeEnvelope(t, response)
	if body["success"] != false || body["data"] != nil || body["meta"] != nil {
		t.Fatalf("unexpected error envelope: %#v", body)
	}
	errorBody := body["error"].(map[string]interface{})
	if errorBody["code"] != "RESOURCE_NOT_FOUND" || body["message"] != "Product not found" {
		t.Fatalf("unexpected error details: %#v", body)
	}
}

func performRequest(router http.Handler, method, path string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func decodeEnvelope(t *testing.T, response *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	return body
}
