package controllers

import (
	"net/http"
	"posku/internal/database"
	"posku/internal/models"

	"github.com/gin-gonic/gin"
)

type CreateSalesItemRequest struct {
	SalesTransactionID uint    `json:"sales_transaction_id" binding:"required"`
	ProductID          uint    `json:"product_id" binding:"required"`
	Quantity           int     `json:"quantity" binding:"required,gt=0"`
	Price              float64 `json:"price" binding:"required,gt=0"`
}

type UpdateSalesItemRequest struct {
	Quantity int     `json:"quantity" binding:"required,gt=0"`
	Price    float64 `json:"price" binding:"required,gt=0"`
}

func CreateSalesItem(c *gin.Context) {
	var req CreateSalesItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	si := models.SalesItem{
		SalesTransactionID: req.SalesTransactionID,
		ProductID:          req.ProductID,
		Quantity:           req.Quantity,
		Price:              req.Price,
		Subtotal:           float64(req.Quantity) * req.Price,
	}

	if err := database.DB.Create(&si).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create sales item"})
		return
	}

	c.JSON(http.StatusCreated, si)
}

func GetSalesItems(c *gin.Context) {
	var list []models.SalesItem
	if err := database.DB.Preload("Product").Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sales items"})
		return
	}
	c.JSON(http.StatusOK, list)
}

func GetSalesItem(c *gin.Context) {
	id := c.Param("id")
	var si models.SalesItem
	if err := database.DB.Preload("Product").First(&si, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sales item not found"})
		return
	}
	c.JSON(http.StatusOK, si)
}

func UpdateSalesItem(c *gin.Context) {
	id := c.Param("id")
	var si models.SalesItem
	if err := database.DB.First(&si, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sales item not found"})
		return
	}
	var input UpdateSalesItemRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	si.Quantity = input.Quantity
	si.Price = input.Price
	si.Subtotal = float64(input.Quantity) * input.Price

	if err := database.DB.Save(&si).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sales item"})
		return
	}

	c.JSON(http.StatusOK, si)
}

func DeleteSalesItem(c *gin.Context) {
	id := c.Param("id")
	var si models.SalesItem
	if err := database.DB.First(&si, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sales item not found"})
		return
	}

	if err := database.DB.Delete(&si).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete sales item"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Sales item deleted"})
}
