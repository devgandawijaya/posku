package controllers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSwaggerEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/swagger", ServeSwaggerUI)
	r.GET("/swagger/index.html", ServeSwaggerUI)
	r.GET("/swagger/doc.json", ServeSwaggerJSON)
	r.GET("/docs/swagger.json", ServeSwaggerJSON)

	endpoints := []string{
		"/swagger",
		"/swagger/index.html",
		"/swagger/doc.json",
		"/docs/swagger.json",
	}

	for _, ep := range endpoints {
		req, _ := http.NewRequest("GET", ep, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d", ep, w.Code)
		}
		if len(w.Body.Bytes()) == 0 {
			t.Fatalf("expected non-empty body for %s", ep)
		}
	}
}
