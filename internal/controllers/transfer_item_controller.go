package controllers

import (
	"net/http"
	"posku/internal/database"
	"posku/internal/models"

	"github.com/gin-gonic/gin"
)

type CreateTransferItemRequest struct {
	StockTransferID uint `json:"stock_transfer_id" binding:"required"`
	ProductID       uint `json:"product_id" binding:"required"`
	Quantity        int  `json:"quantity" binding:"required,gt=0"`
}

type UpdateTransferItemRequest struct {
	Quantity int `json:"quantity" binding:"required,gt=0"`
}

func CreateTransferItem(c *gin.Context) {
	var req CreateTransferItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ti := models.TransferItem{
		StockTransferID: req.StockTransferID,
		ProductID:       req.ProductID,
		Quantity:        req.Quantity,
	}

	if err := database.DB.Create(&ti).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create transfer item"})
		return
	}

	c.JSON(http.StatusCreated, ti)
}

func GetTransferItems(c *gin.Context) {
	var list []models.TransferItem
	if err := database.DB.Preload("Product").Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transfer items"})
		return
	}
	c.JSON(http.StatusOK, list)
}

func GetTransferItem(c *gin.Context) {
	id := c.Param("id")
	var ti models.TransferItem
	if err := database.DB.Preload("Product").First(&ti, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transfer item not found"})
		return
	}
	c.JSON(http.StatusOK, ti)
}

func UpdateTransferItem(c *gin.Context) {
	id := c.Param("id")
	var ti models.TransferItem
	if err := database.DB.First(&ti, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transfer item not found"})
		return
	}
	var input UpdateTransferItemRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ti.Quantity = input.Quantity

	if err := database.DB.Save(&ti).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update transfer item"})
		return
	}

	c.JSON(http.StatusOK, ti)
}

func DeleteTransferItem(c *gin.Context) {
	id := c.Param("id")
	var ti models.TransferItem
	if err := database.DB.First(&ti, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transfer item not found"})
		return
	}

	if err := database.DB.Delete(&ti).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete transfer item"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Transfer item deleted"})
}
