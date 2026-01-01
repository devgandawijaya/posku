package controllers

import (
	"net/http"
	"posku/internal/database"
	"posku/internal/models"

	"github.com/gin-gonic/gin"
)

type CreateWarehouseRequest struct {
	StoreID  uint   `json:"store_id" binding:"required"`
	Name     string `json:"name" binding:"required"`
	Location string `json:"location"`
}

type UpdateWarehouseRequest struct {
	StoreID  uint   `json:"store_id" binding:"required"`
	Name     string `json:"name" binding:"required"`
	Location string `json:"location"`
}

func CreateWarehouse(c *gin.Context) {
	var req CreateWarehouseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	wh := models.Warehouse{
		StoreID:  req.StoreID,
		Name:     req.Name,
		Location: req.Location,
	}

	if err := database.DB.Create(&wh).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create warehouse"})
		return
	}

	c.JSON(http.StatusCreated, wh)
}

func GetWarehouses(c *gin.Context) {
	var list []models.Warehouse
	if err := database.DB.Preload("Store").Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch warehouses"})
		return
	}
	c.JSON(http.StatusOK, list)
}

func GetWarehouse(c *gin.Context) {
	id := c.Param("id")
	var wh models.Warehouse
	if err := database.DB.Preload("Store").First(&wh, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Warehouse not found"})
		return
	}
	c.JSON(http.StatusOK, wh)
}

func UpdateWarehouse(c *gin.Context) {
	id := c.Param("id")
	var wh models.Warehouse
	if err := database.DB.First(&wh, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Warehouse not found"})
		return
	}

	var input UpdateWarehouseRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	wh.Name = input.Name
	wh.Location = input.Location
	wh.StoreID = input.StoreID

	if err := database.DB.Save(&wh).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update warehouse"})
		return
	}

	c.JSON(http.StatusOK, wh)
}

func DeleteWarehouse(c *gin.Context) {
	id := c.Param("id")
	var wh models.Warehouse
	if err := database.DB.First(&wh, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Warehouse not found"})
		return
	}

	if err := database.DB.Delete(&wh).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete warehouse"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Warehouse deleted"})
}

func GetWarehouseByStore(c *gin.Context) {
	storeID := c.Param("store_id")
	var wh models.Warehouse
	if err := database.DB.Where("store_id = ?", storeID).First(&wh).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Warehouse not found for store"})
		return
	}
	c.JSON(http.StatusOK, wh)
}
