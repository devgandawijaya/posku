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
