package controllers

import (
	"net/http"
	"posku/internal/database"
	"posku/internal/models"

	"github.com/gin-gonic/gin"
)

type CreateStoreRequest struct {
	CompanyID uint   `json:"company_id" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Address   string `json:"address"`
}

type UpdateStoreRequest struct {
	CompanyID uint   `json:"company_id" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Address   string `json:"address"`
}

func CreateStore(c *gin.Context) {
	var req CreateStoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	store := models.Store{
		CompanyID: req.CompanyID,
		Name:      req.Name,
		Address:   req.Address,
	}

	if err := database.DB.Create(&store).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create store"})
		return
	}

	c.JSON(http.StatusCreated, store)
}

func GetStores(c *gin.Context) {
	var stores []models.Store
	if err := database.DB.Preload("Company").Find(&stores).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch stores"})
		return
	}
	c.JSON(http.StatusOK, stores)
}

func GetStore(c *gin.Context) {
	id := c.Param("id")
	var store models.Store
	if err := database.DB.Preload("Company").First(&store, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Store not found"})
		return
	}
	c.JSON(http.StatusOK, store)
}

func UpdateStore(c *gin.Context) {
	id := c.Param("id")
	var store models.Store
	if err := database.DB.First(&store, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Store not found"})
		return
	}

	var input UpdateStoreRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	store.Name = input.Name
	store.Address = input.Address
	store.CompanyID = input.CompanyID

	if err := database.DB.Save(&store).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update store"})
		return
	}

	c.JSON(http.StatusOK, store)
}

func DeleteStore(c *gin.Context) {
	id := c.Param("id")
	var store models.Store
	if err := database.DB.First(&store, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Store not found"})
		return
	}

	if err := database.DB.Delete(&store).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete store"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Store deleted"})
}

func GetStoresByCompany(c *gin.Context) {
	companyID := c.Param("company_id")
	var stores []models.Store
	if err := database.DB.Where("company_id = ?", companyID).Find(&stores).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch stores"})
		return
	}
	c.JSON(http.StatusOK, stores)
}
