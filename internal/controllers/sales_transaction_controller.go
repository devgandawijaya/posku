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
	Method          string  `json:"method" binding:"required"`
	Amount          float64 `json:"amount" binding:"required,gt=0"`
	CashReceived    float64 `json:"cash_received"`
	Change          float64 `json:"change"`
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
	Payments    []PaymentInput   `json:"payments"`
}

func CreateSalesTransaction(c *gin.Context) {
	if cached, ok := idempotencyReplay(c, "sales.create"); ok {
		c.JSON(http.StatusCreated, cached)
		return
	}

	var req CreateSalesTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, err.Error())
		return
	}

	companyID := getAuthCompanyID(c)
	var store models.Store
	if err := database.DB.First(&store, req.StoreID).Error; err != nil {
		RespondError(c, http.StatusBadRequest, "store not found")
		return
	}
	if companyID > 0 && store.CompanyID != companyID {
		RespondError(c, http.StatusForbidden, "store does not belong to authorized company")
		return
	}

	// Consolidate payment list
	var paymentsToProcess []PaymentInput
	if len(req.Payments) > 0 {
		paymentsToProcess = req.Payments
	} else if req.Payment != nil {
		paymentsToProcess = append(paymentsToProcess, *req.Payment)
	}

	primaryMethod := "cash"
	if len(paymentsToProcess) == 1 {
		primaryMethod = paymentsToProcess[0].Method
	} else if len(paymentsToProcess) > 1 {
		primaryMethod = "split"
	}

	tx := database.DB.Begin()

	// find warehouse for store
	var wh models.Warehouse
	if err := tx.Where("store_id = ?", req.StoreID).First(&wh).Error; err != nil {
		if err2 := tx.Where("company_id = ?", store.CompanyID).First(&wh).Error; err2 != nil {
			tx.Rollback()
			RespondError(c, http.StatusBadRequest, "warehouse not found for store")
			return
		}
	}

	st := models.SalesTransaction{
		CompanyID:     store.CompanyID,
		StoreID:       req.StoreID,
		ShiftID:       req.ShiftID,
		EmployeeID:    req.EmployeeID,
		CustomerID:    req.CustomerID,
		TotalAmount:   req.TotalAmount,
		Subtotal:      req.TotalAmount,
		PaymentMethod: primaryMethod,
		Status:        "lunas",
		SyncStatus:    "synced",
	}

	if err := tx.Create(&st).Error; err != nil {
		tx.Rollback()
		RespondError(c, http.StatusInternalServerError, "Failed to create sales transaction")
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
				RespondError(c, http.StatusBadRequest, "stock not found for product in warehouse")
				return
			}
			tx.Rollback()
			RespondError(c, http.StatusInternalServerError, "failed to fetch stock")
			return
		}

		if stock.Quantity < it.Quantity {
			tx.Rollback()
			RespondError(c, http.StatusBadRequest, "insufficient stock")
			return
		}

		stock.Quantity -= it.Quantity
		if err := tx.Save(&stock).Error; err != nil {
			tx.Rollback()
			RespondError(c, http.StatusInternalServerError, "failed to update stock")
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
			RespondError(c, http.StatusInternalServerError, "Failed to create sales item")
			return
		}
	}

	// record payments (supporting split payments!)
	for _, p := range paymentsToProcess {
		change := p.Change
		if p.Method == "cash" && p.CashReceived > p.Amount && change == 0 {
			change = p.CashReceived - p.Amount
		}
		payRecord := models.Payment{
			SalesTransactionID: st.ID,
			Method:             p.Method,
			Amount:             p.Amount,
			CashReceived:       p.CashReceived,
			Change:             change,
			ReferenceNumber:    p.ReferenceNumber,
		}
		if err := tx.Create(&payRecord).Error; err != nil {
			tx.Rollback()
			RespondError(c, http.StatusInternalServerError, "Failed to create payment: "+err.Error())
			return
		}
	}

	tx.Commit()

	var created models.SalesTransaction
	if err := database.DB.Preload("SalesItems").Preload("Payments").Preload("Payment").First(&created, st.ID).Error; err != nil {
		c.JSON(http.StatusOK, st)
		return
	}
	idempotencySave(c, "sales.create", created)
	incrementTransactionQuota(created.CompanyID)
	dispatchWebhookEvent(created.CompanyID, "sale.created", created)
	RespondCreated(c, created, "Sales transaction created successfully")
}

func GetSalesTransactions(c *gin.Context) {
	companyID := getAuthCompanyID(c)
	q := database.DB.Model(&models.SalesTransaction{}).
		Preload("SalesItems").
		Preload("Payments").
		Preload("Payment").
		Preload("Store").
		Preload("Employee")

	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	} else if qCID := c.Query("company_id"); qCID != "" {
		q = q.Where("company_id = ?", qCID)
	}

	if storeID := c.Query("store_id"); storeID != "" {
		q = q.Where("store_id = ?", storeID)
	}
	if employeeID := c.Query("employee_id"); employeeID != "" {
		q = q.Where("employee_id = ?", employeeID)
	}
	if shiftID := c.Query("shift_id"); shiftID != "" {
		q = q.Where("shift_id = ?", shiftID)
	}
	if customerID := c.Query("customer_id"); customerID != "" {
		q = q.Where("customer_id = ?", customerID)
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

	// Pagination
	if c.Query("page") != "" || c.Query("per_page") != "" {
		var total int64
		q.Count(&total)
		page, perPage, offset := GetPagination(c)
		var list []models.SalesTransaction
		if err := q.Order("transaction_date desc").Limit(perPage).Offset(offset).Find(&list).Error; err != nil {
			RespondError(c, http.StatusInternalServerError, "Failed to fetch sales transactions")
			return
		}
		RespondPaginated(c, list, page, perPage, total)
		return
	}

	var list []models.SalesTransaction
	if err := q.Order("transaction_date desc").Find(&list).Error; err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to fetch sales transactions")
		return
	}
	c.JSON(http.StatusOK, list)
}

func GetSalesTransaction(c *gin.Context) {
	id := c.Param("id")
	companyID := getAuthCompanyID(c)

	var st models.SalesTransaction
	q := database.DB.Preload("SalesItems.Product").
		Preload("Payments").
		Preload("Payment").
		Preload("Store").
		Preload("Employee").
		Preload("Customer")

	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	}

	if err := q.First(&st, id).Error; err != nil {
		RespondError(c, http.StatusNotFound, "Sales transaction not found")
		return
	}
	RespondSuccess(c, st)
}

func DeleteSalesTransaction(c *gin.Context) {
	id := c.Param("id")
	companyID := getAuthCompanyID(c)

	var st models.SalesTransaction
	q := database.DB
	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	}
	if err := q.First(&st, id).Error; err != nil {
		RespondError(c, http.StatusNotFound, "Sales transaction not found")
		return
	}

	if err := database.DB.Delete(&st).Error; err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to delete sales transaction")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Sales transaction deleted"})
}

// VoidSalesTransaction implements docs/transaksi.md POST /sales/:id/void
func VoidSalesTransaction(c *gin.Context) {
	id := c.Param("id")
	var st models.SalesTransaction
	if err := database.DB.First(&st, id).Error; err != nil {
		RespondError(c, http.StatusNotFound, "Sales transaction not found")
		return
	}
	if st.Status == "void" {
		RespondError(c, http.StatusConflict, "transaksi sudah void")
		return
	}
	st.Status = "void"
	database.DB.Save(&st)
	logAudit(c, st.CompanyID, "void", "sale", st.ID, st.InvoiceNo, nil)
	actorID, _ := currentEmployeeID(c)
	database.DB.Create(&models.SaleAudit{SaleID: st.ID, ActorID: actorID, Action: "void", Kind: "danger"})
	RespondSuccess(c, st, "Transaction voided successfully")
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
		RespondError(c, http.StatusNotFound, "Sales transaction not found")
		return
	}
	var req RefundSalesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	st.Status = "refund"
	database.DB.Save(&st)
	logAudit(c, st.CompanyID, "refund", "sale", st.ID, st.InvoiceNo, req)
	actorID, _ := currentEmployeeID(c)
	database.DB.Create(&models.SaleRefund{SaleID: st.ID, Amount: req.Amount, Reason: req.Reason, ByUserID: actorID})
	database.DB.Create(&models.SaleAudit{SaleID: st.ID, ActorID: actorID, Action: "refund", Detail: req.Reason, Kind: "warning"})
	RespondSuccess(c, st, "Transaction refunded successfully")
}

// SyncSalesTransaction implements docs/transaksi.md POST /sales/:id/sync (offline POS sync).
func SyncSalesTransaction(c *gin.Context) {
	id := c.Param("id")
	var st models.SalesTransaction
	if err := database.DB.First(&st, id).Error; err != nil {
		RespondError(c, http.StatusNotFound, "Sales transaction not found")
		return
	}
	st.SyncStatus = "synced"
	database.DB.Save(&st)
	database.DB.Create(&models.SaleAudit{SaleID: st.ID, Action: "sync", Kind: "info"})
	RespondSuccess(c, st)
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

// PrintSalesTransaction implements docs/kasir.md POST /sales/:id/print. Returns structured receipt.
func PrintSalesTransaction(c *gin.Context) {
	id := c.Param("id")
	var st models.SalesTransaction
	if err := database.DB.Preload("SalesItems.Product").Preload("Payments").Preload("Payment").Preload("Store").First(&st, id).Error; err != nil {
		RespondError(c, http.StatusNotFound, "Sales transaction not found")
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
		"payments":   st.Payments,
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
