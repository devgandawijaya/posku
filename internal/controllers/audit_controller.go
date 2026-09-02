package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"posku/internal/database"
	"posku/internal/models"

	"github.com/gin-gonic/gin"
)

// logAudit writes an entry to audit_logs; failures are ignored (best-effort logging).
func logAudit(c *gin.Context, companyID uint, action, entityType string, entityID uint, entityName string, diff interface{}) {
	actorID := uint(0)
	actorName := ""
	if v, ok := c.Get("employee_id"); ok {
		if f, ok := v.(float64); ok {
			actorID = uint(f)
		}
	}
	if v, ok := c.Get("employee_name"); ok {
		if s, ok := v.(string); ok {
			actorName = s
		}
	}

	diffJSON := ""
	if diff != nil {
		b, _ := json.Marshal(diff)
		diffJSON = redactSensitiveJSON(b)
	}

	entry := models.AuditLog{
		CompanyID:  companyID,
		ActorID:    actorID,
		ActorName:  actorName,
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		EntityName: entityName,
		Diff:       diffJSON,
		IP:         c.ClientIP(),
	}
	database.DB.Create(&entry)
}

// sensitiveAuditFields lists keys that must never be persisted in audit_logs.diff (docs/audit-logs.md
// PII/secret redaction requirement).
var sensitiveAuditFields = map[string]bool{
	"password": true, "password_hash": true, "old_password": true, "new_password": true,
	"api_key": true, "api_key_encrypted": true, "webhook_secret": true, "webhook_secret_encrypted": true,
	"token": true, "refresh_token": true, "secret": true,
}

// redactSensitiveJSON removes sensitive fields (recursively) before storing a diff payload.
func redactSensitiveJSON(raw []byte) string {
	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	redactValue(v)
	out, err := json.Marshal(v)
	if err != nil {
		return string(raw)
	}
	return string(out)
}

func redactValue(v interface{}) {
	switch val := v.(type) {
	case map[string]interface{}:
		for k, vv := range val {
			if sensitiveAuditFields[k] {
				val[k] = "***redacted***"
				continue
			}
			redactValue(vv)
		}
	case []interface{}:
		for _, item := range val {
			redactValue(item)
		}
	}
}

func GetAuditLogs(c *gin.Context) {
	var list []models.AuditLog
	q := database.DB.Model(&models.AuditLog{})
	if companyID := c.Query("company_id"); companyID != "" {
		q = q.Where("company_id = ?", companyID)
	}
	if entity := c.Query("entity"); entity != "" {
		q = q.Where("entity_type = ?", entity)
	}
	if entityID := c.Query("entity_id"); entityID != "" {
		q = q.Where("entity_id = ?", entityID)
	}
	if actor := c.Query("actor"); actor != "" {
		q = q.Where("actor_id = ?", actor)
	}
	if action := c.Query("action"); action != "" {
		q = q.Where("action = ?", action)
	}
	if err := q.Order("at desc").Limit(200).Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch audit logs"})
		return
	}
	c.JSON(http.StatusOK, list)
}

// GetAuditLog implements docs/audit-logs.md GET /audit/:id
func GetAuditLog(c *gin.Context) {
	id := c.Param("id")
	var entry models.AuditLog
	if err := database.DB.First(&entry, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Audit log not found"})
		return
	}
	c.JSON(http.StatusOK, entry)
}

// GetAuditByEntity implements docs/audit-logs.md GET /audit/entity/:entityType/:entityId
func GetAuditByEntity(c *gin.Context) {
	entityType := c.Param("entityType")
	entityID := c.Param("entityId")
	var list []models.AuditLog
	database.DB.Where("entity_type = ? AND entity_id = ?", entityType, entityID).Order("at desc").Find(&list)
	c.JSON(http.StatusOK, list)
}

// ExportAuditLogs implements docs/audit-logs.md GET /audit/export
func ExportAuditLogs(c *gin.Context) {
	var list []models.AuditLog
	q := database.DB.Model(&models.AuditLog{})
	if companyID := c.Query("company_id"); companyID != "" {
		q = q.Where("company_id = ?", companyID)
	}
	q.Order("at desc").Limit(1000).Find(&list)

	rows := make([][]string, 0, len(list))
	for _, e := range list {
		rows = append(rows, []string{
			fmt.Sprint(e.ID), e.ActorName, e.Action, e.EntityType, fmt.Sprint(e.EntityID), e.EntityName, e.At.Format("2006-01-02 15:04:05"),
		})
	}
	writeCSV(c, "audit_logs.csv", []string{"id", "actor_name", "action", "entity_type", "entity_id", "entity_name", "at"}, rows)
}
