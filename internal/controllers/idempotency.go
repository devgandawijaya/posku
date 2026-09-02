package controllers

import (
	"encoding/json"

	"posku/internal/database"
	"posku/internal/models"

	"github.com/gin-gonic/gin"
)

// idempotencyReplay returns a previously stored response for the given Idempotency-Key header, if any
// (docs/kasir.md checkout idempotency). Returns ok=false when there's no header or no prior record.
func idempotencyReplay(c *gin.Context, endpoint string) (map[string]interface{}, bool) {
	key := c.GetHeader("Idempotency-Key")
	if key == "" {
		return nil, false
	}
	var rec models.IdempotencyKey
	if err := database.DB.Where("key = ? AND endpoint = ?", key, endpoint).First(&rec).Error; err != nil {
		return nil, false
	}
	var out map[string]interface{}
	if err := json.Unmarshal([]byte(rec.Response), &out); err != nil {
		return nil, false
	}
	return out, true
}

// idempotencySave persists the response for later replay when the same Idempotency-Key is reused.
func idempotencySave(c *gin.Context, endpoint string, response interface{}) {
	key := c.GetHeader("Idempotency-Key")
	if key == "" {
		return
	}
	b, err := json.Marshal(response)
	if err != nil {
		return
	}
	database.DB.Create(&models.IdempotencyKey{Key: key, Endpoint: endpoint, Response: string(b)})
}
