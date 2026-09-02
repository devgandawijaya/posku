package controllers

import (
	"errors"
	"fmt"
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
	ShiftID     *uint            `json:"shift_id"`
	EmployeeID  uint             `json:"employee_id" binding:"required"`
	CustomerID  *uint            `json:"customer_id"`
	TotalAmount float64          `json:"total_amount" binding:"required,gt=0"`
	Items       []SalesItemInput `json:"items" binding:"required,dive,required"`
	Payment     *PaymentInput    `json:"payment"`
}

func CreateSalesTransaction(c *gin.Context) {
	if cached, ok := idempotencyReplay(c, "sales.create"); ok {
		c.JSON(http.StatusCreated, cached)
		return
	}

	var req CreateSalesTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var store models.Store
	if err := database.DB.First(&store, req.StoreID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "store not found"})
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
		CompanyID:   store.CompanyID,
		StoreID:     req.StoreID,
		ShiftID:     req.ShiftID,
		EmployeeID:  req.EmployeeID,
		CustomerID:  req.CustomerID,
		TotalAmount: req.TotalAmount,
		Subtotal:    req.TotalAmount,
		Status:      "lunas",
		SyncStatus:  "synced",
	}

	if err := tx.Create(&st).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create sales transaction"})
		return
	}
	st.InvoiceNo = fmt.Sprintf("INV-%06d", st.ID)
	tx.Model(&st).Update("invoice_no", st.InvoiceNo)

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
		tx.Create(&models.StockMovement{
			CompanyID: store.CompanyID,
			ProductID: it.ProductID,
			StoreID:   req.StoreID,
			Delta:     -float64(it.Quantity),
			Reason:    "sale",
			RefType:   "sale",
			RefID:     &st.ID,
			CreatedBy: req.EmployeeID,
		})

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
	idempotencySave(c, "sales.create", created)
	incrementTransactionQuota(created.CompanyID)
	dispatchWebhookEvent(created.CompanyID, "sale.created", created)
	c.JSON(http.StatusCreated, created)
}

func GetSalesTransactions(c *gin.Context) {
	var list []models.SalesTransaction
	q := database.DB.Preload("SalesItems").Preload("Payment")
	if companyID := c.Query("company_id"); companyID != "" {
		q = q.Where("company_id = ?", companyID)
	}
	if storeID := c.Query("store_id"); storeID != "" {
		q = q.Where("store_id = ?", storeID)
	}
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	if payment := c.Query("payment"); payment != "" {
		q = q.Where("payment_method = ?", payment)
	}
	if dateFrom := c.Query("date_from"); dateFrom != "" {
		q = q.Where("transaction_date >= ?", dateFrom)
	}
	if dateTo := c.Query("date_to"); dateTo != "" {
		q = q.Where("transaction_date <= ?", dateTo)
	}
	if err := q.Order("transaction_date desc").Find(&list).Error; err != nil {
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

// VoidSalesTransaction implements docs/transaksi.md POST /sales/:id/void
func VoidSalesTransaction(c *gin.Context) {
	id := c.Param("id")
	var st models.SalesTransaction
	if err := database.DB.First(&st, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sales transaction not found"})
		return
	}
	if st.Status == "void" {
		c.JSON(http.StatusConflict, gin.H{"error": "transaksi sudah void"})
		return
	}
	st.Status = "void"
	database.DB.Save(&st)
	logAudit(c, st.CompanyID, "void", "sale", st.ID, st.InvoiceNo, nil)
	actorID, _ := currentEmployeeID(c)
	database.DB.Create(&models.SaleAudit{SaleID: st.ID, ActorID: actorID, Action: "void", Kind: "danger"})
	c.JSON(http.StatusOK, st)
}

type RefundSalesRequest struct {
	Amount float64 `json:"amount" binding:"required,gt=0"`
	Reason string  `json:"reason"`
}

// RefundSalesTransaction implements docs/transaksi.md POST /sales/:id/refund
func RefundSalesTransaction(c *gin.Context) {
	id := c.Param("id")
	var st models.SalesTransaction
	if err := database.DB.First(&st, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sales transaction not found"})
		return
	}
	var req RefundSalesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	st.Status = "refund"
	database.DB.Save(&st)
	logAudit(c, st.CompanyID, "refund", "sale", st.ID, st.InvoiceNo, req)
	actorID, _ := currentEmployeeID(c)
	database.DB.Create(&models.SaleRefund{SaleID: st.ID, Amount: req.Amount, Reason: req.Reason, ByUserID: actorID})
	database.DB.Create(&models.SaleAudit{SaleID: st.ID, ActorID: actorID, Action: "refund", Detail: req.Reason, Kind: "warning"})
	c.JSON(http.StatusOK, st)
}

// SyncSalesTransaction implements docs/transaksi.md POST /sales/:id/sync (offline POS sync).
func SyncSalesTransaction(c *gin.Context) {
	id := c.Param("id")
	var st models.SalesTransaction
	if err := database.DB.First(&st, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sales transaction not found"})
		return
	}
	st.SyncStatus = "synced"
	database.DB.Save(&st)
	database.DB.Create(&models.SaleAudit{SaleID: st.ID, Action: "sync", Kind: "info"})
	c.JSON(http.StatusOK, st)
}

// ExportSalesTransactions implements docs/transaksi.md GET /sales/export
func ExportSalesTransactions(c *gin.Context) {
	var list []models.SalesTransaction
	q := applySalesFilters(c, database.DB.Model(&models.SalesTransaction{}))
	q.Order("transaction_date desc").Limit(1000).Find(&list)

	rows := make([][]string, 0, len(list))
	for _, s := range list {
		rows = append(rows, []string{
			s.InvoiceNo, s.TransactionDate.Format("2006-01-02 15:04:05"), fmt.Sprint(s.StoreID),
			fmt.Sprintf("%.2f", s.TotalAmount), s.PaymentMethod, s.Status,
		})
	}
	writeCSV(c, "sales.csv", []string{"invoice_no", "date", "store_id", "total", "payment_method", "status"}, rows)
}

// PrintSalesTransaction implements docs/kasir.md POST /sales/:id/print. Returns a structured receipt
// payload for the client to render/print (no physical printer/PDF integration).
func PrintSalesTransaction(c *gin.Context) {
	id := c.Param("id")
	var st models.SalesTransaction
	if err := database.DB.Preload("SalesItems.Product").Preload("Payment").Preload("Store").First(&st, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sales transaction not found"})
		return
	}
	database.DB.Create(&models.SaleAudit{SaleID: st.ID, Action: "print", Kind: "info"})
	c.JSON(http.StatusOK, gin.H{
		"invoice_no": st.InvoiceNo,
		"store":      st.Store.Name,
		"date":       st.TransactionDate,
		"items":      st.SalesItems,
		"subtotal":   st.Subtotal,
		"discount":   st.Discount,
		"tax":        st.Tax,
		"total":      st.TotalAmount,
		"payment":    st.PaymentMethod,
	})
}

// GetSaleAudits implements docs/transaksi.md sale_audit trail for one invoice.
func GetSaleAudits(c *gin.Context) {
	id := c.Param("id")
	var list []models.SaleAudit
	database.DB.Where("sale_id = ?", id).Order("at desc").Find(&list)
	c.JSON(http.StatusOK, list)
}

// GetSaleRefunds implements docs/transaksi.md sale_refunds ledger for one invoice.
func GetSaleRefunds(c *gin.Context) {
	id := c.Param("id")
	var list []models.SaleRefund
	database.DB.Where("sale_id = ?", id).Order("at desc").Find(&list)
	c.JSON(http.StatusOK, list)
}
