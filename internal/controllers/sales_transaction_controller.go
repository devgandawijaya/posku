package controllers

import (
	"errors"
	"net/http"
	"posku/internal/database"
	"posku/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SalesItemInput struct {
	ProductID uint    `json:"product_id" binding:"required"`
	Quantity  int     `json:"quantity" binding:"required,gt=0"`
	Price     float64 `json:"price" binding:"required,gt=0"`
}

type PaymentInput struct {
	Method          string  `json:"method" binding:"required,oneof=cash transfer wallet"`
	Amount          float64 `json:"amount" binding:"required,gt=0"`
	ReferenceNumber string  `json:"reference_number"`
}

type CreateSalesTransactionRequest struct {
	StoreID     uint             `json:"store_id" binding:"required"`
	EmployeeID  uint             `json:"employee_id" binding:"required"`
	CustomerID  *uint            `json:"customer_id"`
	TotalAmount float64          `json:"total_amount" binding:"required,gt=0"`
	Items       []SalesItemInput `json:"items" binding:"required,dive,required"`
	Payment     *PaymentInput    `json:"payment"`
}

func CreateSalesTransaction(c *gin.Context) {
	var req CreateSalesTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx := database.DB.Begin()

	// find warehouse for store
	var wh models.Warehouse
	if err := tx.Where("store_id = ?", req.StoreID).First(&wh).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "warehouse not found for store"})
		return
	}

	st := models.SalesTransaction{
		StoreID:     req.StoreID,
		EmployeeID:  req.EmployeeID,
		CustomerID:  req.CustomerID,
		TotalAmount: req.TotalAmount,
	}

	if err := tx.Create(&st).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create sales transaction"})
		return
	}

	// process items and decrement stock with row lock
	for _, it := range req.Items {
		var stock models.Stock
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("warehouse_id = ? AND product_id = ?", wh.ID, it.ProductID).First(&stock).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				tx.Rollback()
				c.JSON(http.StatusBadRequest, gin.H{"error": "stock not found for product in warehouse"})
				return
			}
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch stock"})
			return
		}

		if stock.Quantity < it.Quantity {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient stock"})
			return
		}

		stock.Quantity -= it.Quantity
		if err := tx.Save(&stock).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update stock"})
			return
		}

		si := models.SalesItem{
			SalesTransactionID: st.ID,
			ProductID:          it.ProductID,
			Quantity:           it.Quantity,
			Price:              it.Price,
			Subtotal:           float64(it.Quantity) * it.Price,
		}
		if err := tx.Create(&si).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create sales item"})
			return
		}
	}

	// payment if present
	if req.Payment != nil {
		p := models.Payment{
			SalesTransactionID: st.ID,
			Method:             req.Payment.Method,
			Amount:             req.Payment.Amount,
			ReferenceNumber:    req.Payment.ReferenceNumber,
		}
		if err := tx.Create(&p).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create payment"})
			return
		}
	}

	tx.Commit()

	var created models.SalesTransaction
	if err := database.DB.Preload("SalesItems").Preload("Payment").First(&created, st.ID).Error; err != nil {
		c.JSON(http.StatusOK, st)
		return
	}
	c.JSON(http.StatusCreated, created)
}

func GetSalesTransactions(c *gin.Context) {
	var list []models.SalesTransaction
	if err := database.DB.Preload("SalesItems").Preload("Payment").Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sales transactions"})
		return
	}
	c.JSON(http.StatusOK, list)
}

func GetSalesTransaction(c *gin.Context) {
	id := c.Param("id")
	var st models.SalesTransaction
	if err := database.DB.Preload("SalesItems").Preload("Payment").First(&st, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sales transaction not found"})
		return
	}
	c.JSON(http.StatusOK, st)
}

func DeleteSalesTransaction(c *gin.Context) {
	id := c.Param("id")
	var st models.SalesTransaction
	if err := database.DB.First(&st, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sales transaction not found"})
		return
	}

	if err := database.DB.Delete(&st).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete sales transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Sales transaction deleted"})
}
