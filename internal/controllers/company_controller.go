package controllers

import (
	"net/http"
	"posku/internal/database"
	"posku/internal/models"

	"github.com/gin-gonic/gin"
)

type CreateCompanyRequest struct {
	Name    string `json:"name" binding:"required"`
	Address string `json:"address"`
}

type UpdateCompanyRequest struct {
	Name    string `json:"name" binding:"required"`
	Address string `json:"address"`
}

// allPermissionsJSON builds a permission matrix JSON with every module/action granted.
func allPermissionsJSON() string {
	items := make([]PermissionInput, 0)
	for _, m := range models.PermissionModules {
		for _, a := range models.PermissionActions {
			items = append(items, PermissionInput{Module: m, Action: a, Granted: true})
		}
	}
	return encodePermissions(items)
}

// seedDefaultRoles creates the system "admin" (scope company, full access) and
// a base "user" (kasir, scope store) role for a newly created tenant/company.
func seedDefaultRoles(companyID uint) {
	database.DB.Create(&models.Role{
		CompanyID:   companyID,
		Name:        "admin",
		Description: "Pemilik/administrator tenant - akses penuh",
		Scope:       "company",
		IsSystem:    true,
		Status:      "aktif",
		Permissions: allPermissionsJSON(),
		StoreIDs:    "[]",
	})
	database.DB.Create(&models.Role{
		CompanyID:   companyID,
		Name:        "user",
		Description: "Kasir - akses operasional dasar per outlet",
		Scope:       "store",
		IsSystem:    false,
		Status:      "aktif",
		Permissions: encodePermissions([]PermissionInput{
			{Module: "kasir", Action: "view", Granted: true},
			{Module: "kasir", Action: "create", Granted: true},
			{Module: "produk", Action: "view", Granted: true},
			{Module: "stok", Action: "view", Granted: true},
		}),
		StoreIDs: "[]",
	})
}

func CreateCompany(c *gin.Context) {
	var req CreateCompanyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	company := models.Company{
		Name:    req.Name,
		Address: req.Address,
	}

	if err := database.DB.Create(&company).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create company"})
		return
	}

	seedDefaultRoles(company.ID)

	c.JSON(http.StatusCreated, company)
}

func GetCompanies(c *gin.Context) {
	var companies []models.Company
	if err := database.DB.Find(&companies).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch companies"})
		return
	}

	c.JSON(http.StatusOK, companies)
}

func GetCompany(c *gin.Context) {
	id := c.Param("id")
	var company models.Company
	if err := database.DB.First(&company, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Company not found"})
		return
	}
	c.JSON(http.StatusOK, company)
}

func UpdateCompany(c *gin.Context) {
	id := c.Param("id")
	var company models.Company
	if err := database.DB.First(&company, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Company not found"})
		return
	}
	var input UpdateCompanyRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	company.Name = input.Name
	company.Address = input.Address

	if err := database.DB.Save(&company).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update company"})
		return
	}

	c.JSON(http.StatusOK, company)
}

func DeleteCompany(c *gin.Context) {
	id := c.Param("id")
	var company models.Company
	if err := database.DB.First(&company, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Company not found"})
		return
	}

	if err := database.DB.Delete(&company).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete company"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Company deleted"})
}
