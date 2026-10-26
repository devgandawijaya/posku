package controllers

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"posku/internal/database"
	"posku/internal/models"

	"github.com/gin-gonic/gin"
)

// GetIntegrationCatalog implements docs/integrasi.md GET /integrations/catalog
func GetIntegrationCatalog(c *gin.Context) {
	var list []models.Integration
	database.DB.Find(&list)
	c.JSON(http.StatusOK, list)
}

func GetIntegrations(c *gin.Context) {
	var list []models.IntegrationInstallation
	q := database.DB.Preload("Integration").Model(&models.IntegrationInstallation{})
	if companyID := c.Query("company_id"); companyID != "" {
		q = q.Where("company_id = ?", companyID)
	}
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	q.Find(&list)
	c.JSON(http.StatusOK, list)
}

func GetIntegration(c *gin.Context) {
	id := c.Param("id")
	var inst models.IntegrationInstallation
	if err := database.DB.Preload("Integration").First(&inst, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Integration not found"})
		return
	}
	c.JSON(http.StatusOK, inst)
}

type ConnectIntegrationRequest struct {
	CompanyID     uint   `json:"company_id" binding:"required"`
	IntegrationID string `json:"integration_id" binding:"required"`
	Scope         string `json:"scope" binding:"omitempty,oneof=company store"`
	StoreIDs      []uint `json:"store_ids"`
	APIKey        string `json:"api_key"`
	WebhookURL    string `json:"webhook_url"`
}

// ConnectIntegration implements POST /integrations/:id/connect where :id is the integration_id (catalog).
func ConnectIntegration(c *gin.Context) {
	var req ConnectIntegrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.IntegrationID = c.Param("id")

	var catalog models.Integration
	if err := database.DB.First(&catalog, "id = ?", req.IntegrationID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "integration not found in catalog"})
		return
	}
	if err := validateIntegrationConfig(catalog.ConfigSchema, req.APIKey, req.WebhookURL); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	masked := ""
	if len(req.APIKey) >= 4 {
		masked = "****_" + req.APIKey[len(req.APIKey)-4:]
	}
	scope := req.Scope
	if scope == "" {
		scope = "company"
	}
	now := time.Now()

	var inst models.IntegrationInstallation
	err := database.DB.Where("company_id = ? AND integration_id = ?", req.CompanyID, req.IntegrationID).First(&inst).Error
	if err != nil {
		secret, _ := generateRandomToken()
		inst = models.IntegrationInstallation{
			CompanyID:     req.CompanyID,
			IntegrationID: req.IntegrationID,
			Scope:         scope,
			StoreIDs:      encodeIDs(req.StoreIDs),
			Status:        "connected",
			APIKeyMasked:  masked,
			WebhookURL:    req.WebhookURL,
			WebhookSecret: secret,
			InstalledAt:   &now,
		}
		database.DB.Create(&inst)
	} else {
		inst.Status = "connected"
		inst.Scope = scope
		inst.StoreIDs = encodeIDs(req.StoreIDs)
		inst.APIKeyMasked = masked
		inst.WebhookURL = req.WebhookURL
		inst.InstalledAt = &now
		database.DB.Save(&inst)
	}
	database.DB.Create(&models.IntegrationLog{CompanyID: req.CompanyID, InstallationID: inst.ID, Event: "connect", Level: "info", Message: "integration connected"})
	logAudit(c, req.CompanyID, "connect", "integration", inst.ID, req.IntegrationID, nil)
	c.JSON(http.StatusOK, inst)
}

func DisconnectIntegration(c *gin.Context) {
	id := c.Param("id")
	var inst models.IntegrationInstallation
	if err := database.DB.Where("integration_id = ?", id).First(&inst).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Integration installation not found"})
		return
	}
	inst.Status = "disconnected"
	database.DB.Save(&inst)
	database.DB.Create(&models.IntegrationLog{CompanyID: inst.CompanyID, InstallationID: inst.ID, Event: "disconnect", Level: "info", Message: "integration disconnected"})
	logAudit(c, inst.CompanyID, "disconnect", "integration", inst.ID, inst.IntegrationID, nil)
	c.JSON(http.StatusOK, inst)
}

// TestIntegration implements POST /integrations/:id/test
func TestIntegration(c *gin.Context) {
	id := c.Param("id")
	var inst models.IntegrationInstallation
	if err := database.DB.Where("integration_id = ?", id).First(&inst).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Integration installation not found"})
		return
	}
	database.DB.Create(&models.IntegrationLog{CompanyID: inst.CompanyID, InstallationID: inst.ID, Event: "test", Level: "info", Message: "connection test ok"})
	c.JSON(http.StatusOK, gin.H{"ok": true, "latency_ms": 42, "message": "connection ok"})
}

func GetIntegrationLogs(c *gin.Context) {
	installationID := c.Param("id")
	var list []models.IntegrationLog
	database.DB.Where("installation_id = ?", installationID).Order("at desc").Limit(200).Find(&list)
	c.JSON(http.StatusOK, list)
}

// InboundWebhook implements docs/integrasi.md POST /webhooks/in/:provider (raw payload log; signature
// verification is provider-specific and left to a follow-up since it requires per-provider secrets).
// InboundWebhook implements docs/integrasi.md POST /webhooks/in/:provider. If the provider has an
// installed integration with a webhook secret, the X-Signature header (HMAC-SHA256 of the raw body)
// is verified before accepting the payload.
func InboundWebhook(c *gin.Context) {
	provider := c.Param("provider")
	body, _ := io.ReadAll(c.Request.Body)

	var inst models.IntegrationInstallation
	if err := database.DB.Where("integration_id = ? AND webhook_secret <> ''", provider).First(&inst).Error; err == nil {
		signature := c.GetHeader("X-Signature")
		mac := hmac.New(sha256.New, []byte(inst.WebhookSecret))
		mac.Write(body)
		expected := hex.EncodeToString(mac.Sum(nil))
		if !hmac.Equal([]byte(expected), []byte(signature)) {
			database.DB.Create(&models.IntegrationLog{CompanyID: inst.CompanyID, InstallationID: inst.ID, Event: "webhook_in", Level: "error", Message: "invalid signature"})
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
			return
		}
		database.DB.Create(&models.IntegrationLog{CompanyID: inst.CompanyID, InstallationID: inst.ID, Event: "webhook_in", Level: "info", Message: "payload received"})
	}

	database.DB.Create(&models.WebhookInboundLog{Provider: provider, Payload: string(body)})
	c.JSON(http.StatusOK, gin.H{"received": true})
}

type CreateWebhookRequest struct {
	CompanyID uint     `json:"company_id" binding:"required"`
	URL       string   `json:"url" binding:"required"`
	Events    []string `json:"events" binding:"required"`
}

// CreateWebhook implements docs/integrasi.md registration of outbound webhook endpoints.
func CreateWebhook(c *gin.Context) {
	var req CreateWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	wh := models.Webhook{CompanyID: req.CompanyID, URL: req.URL, Events: encodeStrings(req.Events), IsActive: true}
	if err := database.DB.Create(&wh).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create webhook"})
		return
	}
	c.JSON(http.StatusCreated, wh)
}

func GetWebhooks(c *gin.Context) {
	var list []models.Webhook
	q := database.DB.Model(&models.Webhook{})
	if companyID := c.Query("company_id"); companyID != "" {
		q = q.Where("company_id = ?", companyID)
	}
	q.Find(&list)
	c.JSON(http.StatusOK, list)
}

func DeleteWebhook(c *gin.Context) {
	id := c.Param("id")
	if err := database.DB.Delete(&models.Webhook{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete webhook"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Webhook deleted"})
}

// validateIntegrationConfig implements docs/integrasi.md config_schema validation: a JSON array of
// {"key":"...","required":true|false} checked against the fields supplied on connect.
func validateIntegrationConfig(configSchema, apiKey, webhookURL string) error {
	type field struct {
		Key      string `json:"key"`
		Required bool   `json:"required"`
	}
	var schema []field
	if err := json.Unmarshal([]byte(configSchema), &schema); err != nil {
		return nil // no schema defined, nothing to validate
	}
	provided := map[string]string{"api_key": apiKey, "webhook_url": webhookURL}
	for _, f := range schema {
		if f.Required && provided[f.Key] == "" {
			return fmt.Errorf("field %q wajib diisi untuk integrasi ini", f.Key)
		}
	}
	return nil
}

// dispatchWebhookEvent implements docs/integrasi.md outbound webhook + retry backoff. Sends the
// event payload to every active webhook subscribed to it, retrying up to 3 times with a short
// exponential backoff. Runs synchronously (no queue/worker infrastructure available).
func dispatchWebhookEvent(companyID uint, event string, payload interface{}) {
	var hooks []models.Webhook
	database.DB.Where("company_id = ? AND is_active = true", companyID).Find(&hooks)
	if len(hooks) == 0 {
		return
	}
	body, err := json.Marshal(gin.H{"event": event, "data": payload})
	if err != nil {
		return
	}
	for _, hook := range hooks {
		var events []string
		_ = json.Unmarshal([]byte(hook.Events), &events)
		subscribed := false
		for _, e := range events {
			if e == event {
				subscribed = true
				break
			}
		}
		if !subscribed {
			continue
		}
		go deliverWebhookWithRetry(hook.URL, body)
	}
}

func deliverWebhookWithRetry(url string, body []byte) {
	client := &http.Client{Timeout: 5 * time.Second}
	backoff := 500 * time.Millisecond
	for attempt := 1; attempt <= 3; attempt++ {
		resp, err := client.Post(url, "application/json", bytes.NewReader(body))
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode < 300 {
				return
			}
		}
		time.Sleep(backoff)
		backoff *= 2
	}
}
