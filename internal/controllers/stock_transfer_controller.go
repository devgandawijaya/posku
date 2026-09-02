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

type TransferItemInput struct {
	ProductID uint `json:"product_id" binding:"required"`
	Quantity  int  `json:"quantity" binding:"required,gt=0"`
}

type CreateStockTransferRequest struct {
	FromStoreID   int                 `json:"from_store_id" binding:"required"`
	ToStoreID     int                 `json:"to_store_id" binding:"required"`
	TransferItems []TransferItemInput `json:"transfer_items" binding:"required,dive,required"`
}

func CreateStockTransfer(c *gin.Context) {
	var req CreateStockTransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx := database.DB.Begin()

	// find warehouses
	var fromWh, toWh models.Warehouse
	if err := tx.Where("store_id = ?", req.FromStoreID).First(&fromWh).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "from warehouse not found"})
		return
	}
	if err := tx.Where("store_id = ?", req.ToStoreID).First(&toWh).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "to warehouse not found"})
		return
	}

	var fromStoreModel, toStoreModel models.Store
	if err := tx.First(&fromStoreModel, req.FromStoreID).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "from store not found"})
		return
	}
	if err := tx.First(&toStoreModel, req.ToStoreID).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "to store not found"})
		return
	}

	st := models.StockTransfer{
		FromStoreID: uint(req.FromStoreID),
		ToStoreID:   uint(req.ToStoreID),
	}

	if err := tx.Create(&st).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create stock transfer"})
		return
	}

	for _, it := range req.TransferItems {
		// lock source stock
		var fromStock models.Stock
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("warehouse_id = ? AND product_id = ?", fromWh.ID, it.ProductID).First(&fromStock).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				tx.Rollback()
				c.JSON(http.StatusBadRequest, gin.H{"error": "source stock not found"})
				return
			}
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch source stock"})
			return
		}

		if fromStock.Quantity < it.Quantity {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient stock in source"})
			return
		}

		fromStock.Quantity -= it.Quantity
		if err := tx.Save(&fromStock).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update source stock"})
			return
		}

		// lock destination stock (may not exist)
		var toStock models.Stock
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("warehouse_id = ? AND product_id = ?", toWh.ID, it.ProductID).First(&toStock).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// create
				toStock = models.Stock{
					WarehouseID: toWh.ID,
					ProductID:   it.ProductID,
					Quantity:    it.Quantity,
				}
				if err := tx.Create(&toStock).Error; err != nil {
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create destination stock"})
					return
				}
			} else {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch destination stock"})
				return
			}
		} else {
			toStock.Quantity += it.Quantity
			if err := tx.Save(&toStock).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update destination stock"})
				return
			}
		}

		// create transfer item
		ti := models.TransferItem{
			StockTransferID: st.ID,
			ProductID:       it.ProductID,
			Quantity:        it.Quantity,
		}
		if err := tx.Create(&ti).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create transfer item"})
			return
		}

		tx.Create(&models.StockMovement{
			CompanyID: fromStoreModel.CompanyID,
			ProductID: it.ProductID,
			StoreID:   fromStoreModel.ID,
			Delta:     -float64(it.Quantity),
			Reason:    "transfer_out",
			RefType:   "transfer",
			RefID:     &st.ID,
		})
		tx.Create(&models.StockMovement{
			CompanyID: toStoreModel.CompanyID,
			ProductID: it.ProductID,
			StoreID:   toStoreModel.ID,
			Delta:     float64(it.Quantity),
			Reason:    "transfer_in",
			RefType:   "transfer",
			RefID:     &st.ID,
		})
	}

	tx.Commit()
	var created models.StockTransfer
	if err := database.DB.Preload("TransferItems").First(&created, st.ID).Error; err != nil {
		c.JSON(http.StatusOK, st)
		return
	}
	c.JSON(http.StatusCreated, created)
}

func GetStockTransfers(c *gin.Context) {
	var list []models.StockTransfer
	if err := database.DB.Preload("TransferItems").Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch stock transfers"})
		return
	}
	c.JSON(http.StatusOK, list)
}

func GetStockTransfer(c *gin.Context) {
	id := c.Param("id")
	var st models.StockTransfer
	if err := database.DB.Preload("TransferItems").First(&st, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Stock transfer not found"})
		return
	}
	c.JSON(http.StatusOK, st)
}

func UpdateStockTransfer(c *gin.Context) {
	id := c.Param("id")
	var st models.StockTransfer
	if err := database.DB.First(&st, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Stock transfer not found"})
		return
	}

	var input models.StockTransfer
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	st.Status = input.Status

	if err := database.DB.Save(&st).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update stock transfer"})
		return
	}

	c.JSON(http.StatusOK, st)
}

func DeleteStockTransfer(c *gin.Context) {
	id := c.Param("id")
	var st models.StockTransfer
	if err := database.DB.First(&st, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Stock transfer not found"})
		return
	}

	if err := database.DB.Delete(&st).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete stock transfer"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Stock transfer deleted"})
}
