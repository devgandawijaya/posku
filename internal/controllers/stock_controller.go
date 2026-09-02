package controllers

import (
	"net/http"
	"strconv"

	"posku/internal/database"
	"posku/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm/clause"
)

func parseUintParam(v string) uint {
	n, _ := strconv.ParseUint(v, 10, 64)
	return uint(n)
}

// GetStockMovements implements docs/stok.md GET /stocks/:productId/movements
func GetStockMovements(c *gin.Context) {
	var list []models.StockMovement
	q := database.DB.Model(&models.StockMovement{})
	if productID := c.Param("productId"); productID != "" && productID != "movements" {
		q = q.Where("product_id = ?", productID)
	}
	if storeID := c.Query("store_id"); storeID != "" {
		q = q.Where("store_id = ?", storeID)
	}
	if companyID := c.Query("company_id"); companyID != "" {
		q = q.Where("company_id = ?", companyID)
	}
	if dateFrom := c.Query("date_from"); dateFrom != "" {
		q = q.Where("created_at >= ?", dateFrom)
	}
	if dateTo := c.Query("date_to"); dateTo != "" {
		q = q.Where("created_at <= ?", dateTo)
	}
	if err := q.Order("created_at desc").Limit(200).Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch stock movements"})
		return
	}
	c.JSON(http.StatusOK, list)
}

// GetStockByStore implements docs/stok.md GET /stocks/:productId/by-store
func GetStockByStore(c *gin.Context) {
	productID := c.Param("productId")
	var list []models.Stock
	if err := database.DB.Preload("Warehouse.Store").Where("product_id = ?", productID).Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch stock"})
		return
	}
	c.JSON(http.StatusOK, list)
}

type StockAdjustItemInput struct {
	ProductID uint `json:"product_id" binding:"required"`
	StoreID   uint `json:"store_id" binding:"required"`
	ActualQty int  `json:"actual_qty"`
}

type StockAdjustRequest struct {
	CompanyID uint                   `json:"company_id" binding:"required"`
	CreatedBy uint                   `json:"created_by"`
	Note      string                 `json:"note"`
	Items     []StockAdjustItemInput `json:"items" binding:"required,dive,required"`
}

// AdjustStock implements docs/stok.md POST /stocks/adjust (stock opname, applied immediately).
func AdjustStock(c *gin.Context) {
	var req StockAdjustRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx := database.DB.Begin()
	for _, it := range req.Items {
		var wh models.Warehouse
		if err := tx.Where("store_id = ?", it.StoreID).First(&wh).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "warehouse not found for store"})
			return
		}
		var stock models.Stock
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("warehouse_id = ? AND product_id = ?", wh.ID, it.ProductID).First(&stock).Error
		if err != nil {
			stock = models.Stock{WarehouseID: wh.ID, ProductID: it.ProductID, Quantity: 0}
			tx.Create(&stock)
		}
		delta := it.ActualQty - stock.Quantity
		stock.Quantity = it.ActualQty
		tx.Save(&stock)
		tx.Create(&models.StockMovement{
			CompanyID: req.CompanyID,
			ProductID: it.ProductID,
			StoreID:   it.StoreID,
			Delta:     float64(delta),
			Reason:    "adjust",
			RefType:   "adjustment",
			Note:      req.Note,
			CreatedBy: req.CreatedBy,
		})
	}
	tx.Commit()
	logAudit(c, req.CompanyID, "stock_adjust", "stock", 0, req.Note, req.Items)
	c.JSON(http.StatusOK, gin.H{"message": "stock adjusted"})
}

type RestockRequest struct {
	CompanyID uint   `json:"company_id" binding:"required"`
	StoreID   uint   `json:"store_id" binding:"required"`
	Qty       int    `json:"qty" binding:"required,gt=0"`
	CreatedBy uint   `json:"created_by"`
	Note      string `json:"note"`
}

// RestockProduct implements docs/stok.md POST /stocks/:productId/restock
func RestockProduct(c *gin.Context) {
	productID := c.Param("productId")
	var req RestockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx := database.DB.Begin()
	var wh models.Warehouse
	if err := tx.Where("store_id = ?", req.StoreID).First(&wh).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "warehouse not found for store"})
		return
	}
	var stock models.Stock
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("warehouse_id = ? AND product_id = ?", wh.ID, productID).First(&stock).Error
	if err != nil {
		stock = models.Stock{WarehouseID: wh.ID, ProductID: parseUintParam(productID), Quantity: 0}
		tx.Create(&stock)
	}
	stock.Quantity += req.Qty
	tx.Save(&stock)
	tx.Create(&models.StockMovement{
		CompanyID: req.CompanyID,
		ProductID: parseUintParam(productID),
		StoreID:   req.StoreID,
		Delta:     float64(req.Qty),
		Reason:    "masuk",
		RefType:   "purchase",
		Note:      req.Note,
		CreatedBy: req.CreatedBy,
	})
	tx.Commit()
	c.JSON(http.StatusOK, stock)
}

type StockAdjustmentItemDraftInput struct {
	ProductID uint    `json:"product_id" binding:"required"`
	ActualQty float64 `json:"actual_qty"`
	Note      string  `json:"note"`
}

type CreateStockAdjustmentRequest struct {
	CompanyID uint                            `json:"company_id" binding:"required"`
	StoreID   uint                            `json:"store_id" binding:"required"`
	CreatedBy uint                            `json:"created_by"`
	Note      string                          `json:"note"`
	Items     []StockAdjustmentItemDraftInput `json:"items" binding:"required,dive,required"`
}

// CreateStockAdjustment implements docs/stok.md POST /stocks/adjustments (draft form, stok opname).
func CreateStockAdjustment(c *gin.Context) {
	var req CreateStockAdjustmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var wh models.Warehouse
	if err := database.DB.Where("store_id = ?", req.StoreID).First(&wh).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "warehouse not found for store"})
		return
	}

	adj := models.StockAdjustment{
		CompanyID: req.CompanyID,
		StoreID:   req.StoreID,
		Status:    "draft",
		Note:      req.Note,
		CreatedBy: req.CreatedBy,
	}
	for _, it := range req.Items {
		var stock models.Stock
		systemQty := 0.0
		if err := database.DB.Where("warehouse_id = ? AND product_id = ?", wh.ID, it.ProductID).First(&stock).Error; err == nil {
			systemQty = float64(stock.Quantity)
		}
		adj.Items = append(adj.Items, models.StockAdjustmentItem{
			ProductID: it.ProductID,
			SystemQty: systemQty,
			ActualQty: it.ActualQty,
			Note:      it.Note,
		})
	}
	if err := database.DB.Create(&adj).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create stock adjustment"})
		return
	}
	c.JSON(http.StatusCreated, adj)
}

func GetStockAdjustments(c *gin.Context) {
	var list []models.StockAdjustment
	q := database.DB.Preload("Items").Model(&models.StockAdjustment{})
	if companyID := c.Query("company_id"); companyID != "" {
		q = q.Where("company_id = ?", companyID)
	}
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	q.Order("created_at desc").Find(&list)
	c.JSON(http.StatusOK, list)
}

// SubmitStockAdjustment moves a draft adjustment to 'submitted' (awaiting approval).
func SubmitStockAdjustment(c *gin.Context) {
	id := c.Param("id")
	var adj models.StockAdjustment
	if err := database.DB.First(&adj, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Stock adjustment not found"})
		return
	}
	if adj.Status != "draft" {
		c.JSON(http.StatusConflict, gin.H{"error": "hanya draft yang bisa disubmit"})
		return
	}
	adj.Status = "submitted"
	database.DB.Save(&adj)
	c.JSON(http.StatusOK, adj)
}

// ApproveStockAdjustment applies the adjustment deltas as stock_movements + updates stocks (docs/stok.md).
func ApproveStockAdjustment(c *gin.Context) {
	id := c.Param("id")
	var adj models.StockAdjustment
	if err := database.DB.Preload("Items").First(&adj, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Stock adjustment not found"})
		return
	}
	if adj.Status != "submitted" {
		c.JSON(http.StatusConflict, gin.H{"error": "hanya adjustment submitted yang bisa disetujui"})
		return
	}

	tx := database.DB.Begin()
	var wh models.Warehouse
	if err := tx.Where("store_id = ?", adj.StoreID).First(&wh).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "warehouse not found for store"})
		return
	}
	for _, it := range adj.Items {
		var stock models.Stock
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("warehouse_id = ? AND product_id = ?", wh.ID, it.ProductID).First(&stock).Error
		if err != nil {
			stock = models.Stock{WarehouseID: wh.ID, ProductID: it.ProductID, Quantity: 0}
			tx.Create(&stock)
		}
		delta := it.ActualQty - float64(stock.Quantity)
		stock.Quantity = int(it.ActualQty)
		tx.Save(&stock)
		tx.Create(&models.StockMovement{
			CompanyID: adj.CompanyID, ProductID: it.ProductID, StoreID: adj.StoreID,
			Delta: delta, Reason: "adjust", RefType: "adjustment", RefID: &adj.ID, CreatedBy: adj.CreatedBy,
		})
	}
	adj.Status = "approved"
	tx.Save(&adj)
	tx.Commit()

	logAudit(c, adj.CompanyID, "stock_adjust", "stock_adjustment", adj.ID, adj.Note, nil)
	c.JSON(http.StatusOK, adj)
}

// RejectStockAdjustment implements docs/stok.md stock_adjustments status='rejected'.
func RejectStockAdjustment(c *gin.Context) {
	id := c.Param("id")
	var adj models.StockAdjustment
	if err := database.DB.First(&adj, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Stock adjustment not found"})
		return
	}
	adj.Status = "rejected"
	database.DB.Save(&adj)
	c.JSON(http.StatusOK, adj)
}
