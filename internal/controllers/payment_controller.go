package controllers

import (
	"net/http"
	"posku/internal/database"
	"posku/internal/models"

	"github.com/gin-gonic/gin"
)

type CreatePaymentRequest struct {
	SalesTransactionID uint    `json:"sales_transaction_id" binding:"required"`
	Method             string  `json:"method" binding:"required,oneof=cash transfer wallet"`
	Amount             float64 `json:"amount" binding:"required,gt=0"`
	ReferenceNumber    string  `json:"reference_number"`
}

type UpdatePaymentRequest struct {
	Method          string  `json:"method" binding:"required,oneof=cash transfer wallet"`
	Amount          float64 `json:"amount" binding:"required,gt=0"`
	ReferenceNumber string  `json:"reference_number"`
}

func CreatePayment(c *gin.Context) {
	var req CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	p := models.Payment{
		SalesTransactionID: req.SalesTransactionID,
		Method:             req.Method,
		Amount:             req.Amount,
		ReferenceNumber:    req.ReferenceNumber,
	}

	if err := database.DB.Create(&p).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create payment"})
		return
	}

	c.JSON(http.StatusCreated, p)
}

func GetPayments(c *gin.Context) {
	var list []models.Payment
	if err := database.DB.Preload("SalesTransaction").Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch payments"})
		return
	}
	c.JSON(http.StatusOK, list)
}

func GetPayment(c *gin.Context) {
	id := c.Param("id")
	var p models.Payment
	if err := database.DB.Preload("SalesTransaction").First(&p, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Payment not found"})
		return
	}
	c.JSON(http.StatusOK, p)
}

func UpdatePayment(c *gin.Context) {
	id := c.Param("id")
	var p models.Payment
	if err := database.DB.First(&p, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Payment not found"})
		return
	}
	var input UpdatePaymentRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	p.Method = input.Method
	p.Amount = input.Amount
	p.ReferenceNumber = input.ReferenceNumber

	if err := database.DB.Save(&p).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update payment"})
		return
	}

	c.JSON(http.StatusOK, p)
}

func DeletePayment(c *gin.Context) {
	id := c.Param("id")
	var p models.Payment
	if err := database.DB.First(&p, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Payment not found"})
		return
	}

	if err := database.DB.Delete(&p).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete payment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Payment deleted"})
}
