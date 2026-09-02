package controllers

import (
	"encoding/json"
	"net/http"

	"posku/internal/database"
	"posku/internal/models"

	"github.com/gin-gonic/gin"
)

type PermissionInput struct {
	Module  string `json:"module" binding:"required"`
	Action  string `json:"action" binding:"required"`
	Granted bool   `json:"granted"`
}

type CreateRoleRequest struct {
	Name        string            `json:"name" binding:"required"`
	Description string            `json:"description"`
	CompanyID   uint              `json:"company_id" binding:"required"`
	Scope       string            `json:"scope" binding:"required,oneof=company store"`
	StoreIDs    []uint            `json:"store_ids"`
	Permissions []PermissionInput `json:"permissions"`
}

type UpdateRoleRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Scope       string `json:"scope" binding:"required,oneof=company store"`
	StoreIDs    []uint `json:"store_ids"`
}

type UpdatePermissionsRequest struct {
	Permissions []PermissionInput `json:"permissions" binding:"required"`
}

func encodePermissions(items []PermissionInput) string {
	matrix := map[string]map[string]bool{}
	for _, it := range items {
		if !it.Granted {
			continue
		}
		if matrix[it.Module] == nil {
			matrix[it.Module] = map[string]bool{}
		}
		matrix[it.Module][it.Action] = true
	}
	b, _ := json.Marshal(matrix)
	return string(b)
}

func encodeIDs(ids []uint) string {
	if ids == nil {
		ids = []uint{}
	}
	b, _ := json.Marshal(ids)
	return string(b)
}

func encodeStrings(items []string) string {
	if items == nil {
		items = []string{}
	}
	b, _ := json.Marshal(items)
	return string(b)
}

func CreateRole(c *gin.Context) {
	var req CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Scope == "store" && len(req.StoreIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "storeIds wajib diisi minimal 1 untuk scope store"})
		return
	}

	var company models.Company
	if err := database.DB.First(&company, req.CompanyID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "company not found"})
		return
	}

	role := models.Role{
		CompanyID:   req.CompanyID,
		Name:        req.Name,
		Description: req.Description,
		Scope:       req.Scope,
		Status:      "aktif",
		Permissions: encodePermissions(req.Permissions),
		StoreIDs:    encodeIDs(req.StoreIDs),
	}

	if err := database.DB.Create(&role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create role"})
		return
	}

	logAudit(c, role.CompanyID, "role_create", "role", role.ID, role.Name, nil)
	c.JSON(http.StatusCreated, role)
}

func GetRoles(c *gin.Context) {
	var list []models.Role
	q := database.DB.Model(&models.Role{})
	if companyID := c.Query("company_id"); companyID != "" {
		q = q.Where("company_id = ?", companyID)
	}
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	if scope := c.Query("scope"); scope != "" {
		q = q.Where("scope = ?", scope)
	}
	if search := c.Query("q"); search != "" {
		q = q.Where("name ILIKE ?", "%"+search+"%")
	}
	if err := q.Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch roles"})
		return
	}
	c.JSON(http.StatusOK, list)
}

func GetRole(c *gin.Context) {
	id := c.Param("id")
	var role models.Role
	if err := database.DB.First(&role, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
		return
	}
	var userCount int64
	database.DB.Model(&models.Employee{}).Where("role_id = ?", role.ID).Count(&userCount)
	c.JSON(http.StatusOK, gin.H{
		"role":       role,
		"user_count": userCount,
	})
}

func UpdateRole(c *gin.Context) {
	id := c.Param("id")
	var role models.Role
	if err := database.DB.First(&role, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
		return
	}
	if role.IsSystem {
		c.JSON(http.StatusForbidden, gin.H{"error": "role sistem tidak bisa diubah"})
		return
	}

	var req UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Scope == "store" && len(req.StoreIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "storeIds wajib diisi minimal 1 untuk scope store"})
		return
	}

	role.Name = req.Name
	role.Description = req.Description
	role.Scope = req.Scope
	role.StoreIDs = encodeIDs(req.StoreIDs)

	if err := database.DB.Save(&role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update role"})
		return
	}
	logAudit(c, role.CompanyID, "role_update", "role", role.ID, role.Name, nil)
	c.JSON(http.StatusOK, role)
}

func UpdateRolePermissions(c *gin.Context) {
	id := c.Param("id")
	var role models.Role
	if err := database.DB.First(&role, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
		return
	}

	var req UpdatePermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	role.Permissions = encodePermissions(req.Permissions)
	if err := database.DB.Save(&role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update permissions"})
		return
	}
	logAudit(c, role.CompanyID, "permission_change", "role", role.ID, role.Name, nil)
	c.JSON(http.StatusOK, role)
}

func ToggleRoleStatus(c *gin.Context) {
	id := c.Param("id")
	var role models.Role
	if err := database.DB.First(&role, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
		return
	}
	if role.IsSystem {
		c.JSON(http.StatusForbidden, gin.H{"error": "role sistem tidak bisa dinonaktifkan"})
		return
	}
	if role.Status == "aktif" {
		role.Status = "nonaktif"
	} else {
		role.Status = "aktif"
	}
	if err := database.DB.Save(&role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to toggle role"})
		return
	}
	logAudit(c, role.CompanyID, "role_toggle", "role", role.ID, role.Name, nil)
	c.JSON(http.StatusOK, role)
}

func DeleteRole(c *gin.Context) {
	id := c.Param("id")
	var role models.Role
	if err := database.DB.First(&role, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
		return
	}
	if role.IsSystem {
		c.JSON(http.StatusForbidden, gin.H{"error": "role sistem tidak bisa dihapus"})
		return
	}
	var userCount int64
	database.DB.Model(&models.Employee{}).Where("role_id = ?", role.ID).Count(&userCount)
	if userCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "tidak bisa hapus role yang masih dipakai user aktif"})
		return
	}
	if err := database.DB.Delete(&role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete role"})
		return
	}
	logAudit(c, role.CompanyID, "role_delete", "role", role.ID, role.Name, nil)
	c.JSON(http.StatusOK, gin.H{"message": "Role deleted"})
}

// GetPermissionsCatalog returns the module x action matrix for the permission editor.
func GetPermissionsCatalog(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"modules": models.PermissionModules,
		"actions": models.PermissionActions,
	})
}
