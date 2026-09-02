package controllers

import (
	"net/http"
	"time"

	"posku/internal/database"
	"posku/internal/models"

	"github.com/gin-gonic/gin"
)

type ReturnItemInput struct {
	SalesItemID uint    `json:"sales_item_id" binding:"required"`
	ProductID   uint    `json:"product_id" binding:"required"`
	Qty         float64 `json:"qty" binding:"required,gt=0"`
	Price       float64 `json:"price" binding:"required,gt=0"`
	Fate        string  `json:"fate" binding:"required,oneof=restock qc damaged supplier"`
}

type CreateReturnRequest struct {
	CompanyID          uint              `json:"company_id" binding:"required"`
	StoreID            uint              `json:"store_id" binding:"required"`
	EmployeeID         uint              `json:"employee_id" binding:"required"`
	CustomerID         *uint             `json:"customer_id"`
	SalesTransactionID uint              `json:"sales_transaction_id" binding:"required"`
	Reason             string            `json:"reason" binding:"required,oneof=damaged wrong-item not-as-described expired changed-mind other"`
	ReasonNote         string            `json:"reason_note"`
	Items              []ReturnItemInput `json:"items" binding:"required,dive,required"`
}

func CreateReturn(c *gin.Context) {
	var req CreateReturnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var sale models.SalesTransaction
	if err := database.DB.First(&sale, req.SalesTransactionID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sales transaction not found"})
		return
	}

	var total float64
	items := make([]models.ReturnItem, 0, len(req.Items))
	for _, it := range req.Items {
		var product models.Product
		if err := database.DB.First(&product, it.ProductID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "product not found"})
			return
		}
		amount := it.Qty * it.Price
		total += amount
		items = append(items, models.ReturnItem{
			SalesItemID:  it.SalesItemID,
			ProductID:    it.ProductID,
			SKUSnapshot:  product.SKU,
			NameSnapshot: product.Name,
			Qty:          it.Qty,
			Price:        it.Price,
			Amount:       amount,
			Fate:         it.Fate,
		})
	}

	ret := models.Return{
		CompanyID:          req.CompanyID,
		StoreID:            req.StoreID,
		EmployeeID:         req.EmployeeID,
		CustomerID:         req.CustomerID,
		SalesTransactionID: req.SalesTransactionID,
		TotalRefund:        total,
		Reason:             req.Reason,
		ReasonNote:         req.ReasonNote,
		Status:             models.ReturnStatusPending,
		Items:              items,
	}

	if err := database.DB.Create(&ret).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create return"})
		return
	}

	steps := []struct {
		Role string
	}{{"Kasir"}, {"Supervisor"}, {"Manager"}}
	for i, s := range steps {
		status := "pending"
		if i == 0 {
			status = "current"
		}
		database.DB.Create(&models.ReturnApproval{
			ReturnID: ret.ID,
			Step:     i + 1,
			Role:     s.Role,
			Status:   status,
		})
	}

	logAudit(c, ret.CompanyID, "return_create", "return", ret.ID, "", nil)
	c.JSON(http.StatusCreated, ret)
}

func GetReturns(c *gin.Context) {
	var list []models.Return
	q := database.DB.Preload("Items").Model(&models.Return{})
	if companyID := c.Query("company_id"); companyID != "" {
		q = q.Where("company_id = ?", companyID)
	}
	if storeID := c.Query("store_id"); storeID != "" {
		q = q.Where("store_id = ?", storeID)
	}
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	if reason := c.Query("reason"); reason != "" {
		q = q.Where("reason = ?", reason)
	}
	if dateFrom := c.Query("date_from"); dateFrom != "" {
		q = q.Where("date >= ?", dateFrom)
	}
	if dateTo := c.Query("date_to"); dateTo != "" {
		q = q.Where("date <= ?", dateTo)
	}
	if err := q.Order("date desc").Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch returns"})
		return
	}
	c.JSON(http.StatusOK, list)
}

func GetReturn(c *gin.Context) {
	id := c.Param("id")
	var ret models.Return
	if err := database.DB.Preload("Items").Preload("Approvals").Preload("Store").Preload("Employee").Preload("Customer").First(&ret, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Return not found"})
		return
	}
	c.JSON(http.StatusOK, ret)
}

func loadReturn(c *gin.Context) (*models.Return, bool) {
	id := c.Param("id")
	var ret models.Return
	if err := database.DB.First(&ret, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Return not found"})
		return nil, false
	}
	return &ret, true
}

func ApproveReturn(c *gin.Context) {
	ret, ok := loadReturn(c)
	if !ok {
		return
	}
	if ret.Status != models.ReturnStatusPending {
		c.JSON(http.StatusConflict, gin.H{"error": "hanya retur berstatus pending yang bisa disetujui"})
		return
	}

	var current models.ReturnApproval
	if err := database.DB.Where("return_id = ? AND status = 'current'", ret.ID).First(&current).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "tidak ada langkah approval yang menunggu"})
		return
	}

	now := time.Now()
	current.Status = "done"
	current.At = &now
	if v, exists := c.Get("employee_name"); exists {
		if s, ok := v.(string); ok {
			current.ApproverName = s
		}
	}
	database.DB.Save(&current)

	var next models.ReturnApproval
	err := database.DB.Where("return_id = ? AND step = ?", ret.ID, current.Step+1).First(&next).Error
	if err != nil {
		// no more steps -> fully approved
		ret.Status = models.ReturnStatusApproved
		database.DB.Save(ret)
	} else {
		next.Status = "current"
		database.DB.Save(&next)
		ret.CurrentStep = next.Step
		database.DB.Save(ret)
	}

	logAudit(c, ret.CompanyID, "return_approve", "return", ret.ID, "", nil)
	c.JSON(http.StatusOK, ret)
}

type RejectReturnRequest struct {
	Note string `json:"note"`
}

func RejectReturn(c *gin.Context) {
	ret, ok := loadReturn(c)
	if !ok {
		return
	}
	var req RejectReturnRequest
	_ = c.ShouldBindJSON(&req)

	database.DB.Model(&models.ReturnApproval{}).Where("return_id = ? AND status = 'current'", ret.ID).
		Updates(map[string]interface{}{"status": "rejected", "note": req.Note})

	ret.Status = models.ReturnStatusRejected
	ret.QCNote = req.Note
	database.DB.Save(ret)
	logAudit(c, ret.CompanyID, "return_reject", "return", ret.ID, "", req.Note)
	c.JSON(http.StatusOK, ret)
}

func ProcessReturn(c *gin.Context) {
	ret, ok := loadReturn(c)
	if !ok {
		return
	}
	if ret.Status != models.ReturnStatusApproved {
		c.JSON(http.StatusConflict, gin.H{"error": "retur harus disetujui sebelum diproses"})
		return
	}

	var items []models.ReturnItem
	database.DB.Where("return_id = ?", ret.ID).Find(&items)

	tx := database.DB.Begin()
	for _, it := range items {
		if it.Fate != "restock" {
			continue
		}
		var wh models.Warehouse
		if err := tx.Where("store_id = ?", ret.StoreID).First(&wh).Error; err != nil {
			continue
		}
		var stock models.Stock
		if err := tx.Where("warehouse_id = ? AND product_id = ?", wh.ID, it.ProductID).First(&stock).Error; err == nil {
			stock.Quantity += int(it.Qty)
			tx.Save(&stock)
		}
	}
	ret.Status = models.ReturnStatusProcessing
	now := time.Now()
	ret.RestockAt = &now
	tx.Save(ret)
	tx.Commit()

	logAudit(c, ret.CompanyID, "return_process", "return", ret.ID, "", nil)
	c.JSON(http.StatusOK, ret)
}

func CompleteReturn(c *gin.Context) {
	ret, ok := loadReturn(c)
	if !ok {
		return
	}
	if ret.Status != models.ReturnStatusProcessing {
		c.JSON(http.StatusConflict, gin.H{"error": "retur harus diproses sebelum diselesaikan"})
		return
	}
	ret.Status = models.ReturnStatusCompleted
	database.DB.Save(ret)
	logAudit(c, ret.CompanyID, "return_complete", "return", ret.ID, "", nil)
	c.JSON(http.StatusOK, ret)
}

type RefundReturnRequest struct {
	Amount float64 `json:"amount" binding:"required,gt=0"`
	Method string  `json:"method" binding:"required,oneof=cash qris debit credit ewallet voucher"`
	RefNo  string  `json:"ref_no"`
}

// RefundReturn implements docs/retur.md POST /returns/:id/refund (records refund_payments).
func RefundReturn(c *gin.Context) {
	ret, ok := loadReturn(c)
	if !ok {
		return
	}
	var req RefundReturnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	actorID, _ := currentEmployeeID(c)
	payment := models.RefundPayment{
		ReturnID: ret.ID,
		Amount:   req.Amount,
		Method:   req.Method,
		RefNo:    req.RefNo,
		ByUserID: actorID,
	}
	database.DB.Create(&payment)
	logAudit(c, ret.CompanyID, "refund_payment", "return", ret.ID, "", req)
	c.JSON(http.StatusCreated, payment)
}

// GetReturnRefundPayments lists refund_payments for a return.
func GetReturnRefundPayments(c *gin.Context) {
	id := c.Param("id")
	var list []models.RefundPayment
	database.DB.Where("return_id = ?", id).Order("paid_at desc").Find(&list)
	c.JSON(http.StatusOK, list)
}
