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

type CartItemInput struct {
	ProductID   uint    `json:"product_id" binding:"required"`
	Price       float64 `json:"price"`
	Qty         float64 `json:"qty" binding:"required,gt=0"`
	DiscountPct float64 `json:"discount_pct"`
	Note        string  `json:"note"`
}

type CreateCartRequest struct {
	CompanyID  uint            `json:"company_id"`
	ShiftID    uint            `json:"shift_id" binding:"required"`
	EmployeeID uint            `json:"employee_id" binding:"required"`
	StoreID    uint            `json:"store_id" binding:"required"`
	CustomerID *uint           `json:"customer_id"`
	Status     string          `json:"status" binding:"omitempty,oneof=active held"`
	Items      []CartItemInput `json:"items"`
}

func recalcCartTotals(cart *models.Cart) {
	var subtotal float64
	for _, it := range cart.Items {
		lineDiscount := it.Price * it.Qty * (it.DiscountPct / 100)
		subtotal += it.Price*it.Qty - lineDiscount
	}
	cart.Subtotal = subtotal
	cart.Total = subtotal - cart.VoucherDiscount + cart.Tax - cart.Discount
}

// resolveProductPrice checks if there is a store-specific price override; otherwise uses base product price.
func resolveProductPrice(product models.Product, storeID uint) float64 {
	var sp models.ProductStorePrice
	if err := database.DB.Where("product_id = ? AND store_id = ?", product.ID, storeID).First(&sp).Error; err == nil && sp.Price > 0 {
		return sp.Price
	}
	return product.Price
}

// CreateCart implements docs/kasir.md POST /carts (create / hold).
func CreateCart(c *gin.Context) {
	var req CreateCartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, err.Error())
		return
	}

	companyID := getAuthCompanyID(c)
	if companyID == 0 {
		companyID = req.CompanyID
	}
	if companyID == 0 {
		RespondError(c, http.StatusBadRequest, "company_id is required")
		return
	}

	status := req.Status
	if status == "" {
		status = "active"
	}
	cart := models.Cart{
		CompanyID:  companyID,
		ShiftID:    req.ShiftID,
		EmployeeID: req.EmployeeID,
		StoreID:    req.StoreID,
		CustomerID: req.CustomerID,
		Status:     status,
	}
	for _, it := range req.Items {
		var product models.Product
		if err := database.DB.First(&product, it.ProductID).Error; err == nil {
			unitPrice := it.Price
			if unitPrice == 0 {
				unitPrice = resolveProductPrice(product, req.StoreID)
			}
			cart.Items = append(cart.Items, models.CartItem{
				ProductID:    it.ProductID,
				SKUSnapshot:  product.SKU,
				NameSnapshot: product.Name,
				Price:        unitPrice,
				Qty:          it.Qty,
				DiscountPct:  it.DiscountPct,
				Note:         it.Note,
			})
		}
	}
	recalcCartTotals(&cart)
	if err := database.DB.Create(&cart).Error; err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to create cart")
		return
	}
	RespondCreated(c, cart)
}

type UpdateCartItemsRequest struct {
	Items []CartItemInput `json:"items" binding:"required"`
}

// UpdateCartItems implements docs/kasir.md PATCH /carts/:id/items (replace items).
func UpdateCartItems(c *gin.Context) {
	id := c.Param("id")
	var cart models.Cart
	if err := database.DB.Preload("Items").First(&cart, id).Error; err != nil {
		RespondError(c, http.StatusNotFound, "Cart not found")
		return
	}
	var req UpdateCartItemsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	database.DB.Where("cart_id = ?", cart.ID).Delete(&models.CartItem{})
	cart.Items = nil
	for _, it := range req.Items {
		var product models.Product
		if err := database.DB.First(&product, it.ProductID).Error; err != nil {
			continue
		}
		unitPrice := it.Price
		if unitPrice == 0 {
			unitPrice = resolveProductPrice(product, cart.StoreID)
		}
		item := models.CartItem{
			CartID:       cart.ID,
			ProductID:    it.ProductID,
			SKUSnapshot:  product.SKU,
			NameSnapshot: product.Name,
			Price:        unitPrice,
			Qty:          it.Qty,
			DiscountPct:  it.DiscountPct,
			Note:         it.Note,
		}
		database.DB.Create(&item)
		cart.Items = append(cart.Items, item)
	}
	recalcCartTotals(&cart)
	database.DB.Save(&cart)
	RespondSuccess(c, cart)
}

type ApplyCartVoucherRequest struct {
	Code string `json:"code" binding:"required"`
}

// ApplyCartVoucher implements docs/kasir.md POST /carts/:id/apply-voucher
func ApplyCartVoucher(c *gin.Context) {
	id := c.Param("id")
	var cart models.Cart
	if err := database.DB.Preload("Items").First(&cart, id).Error; err != nil {
		RespondError(c, http.StatusNotFound, "Cart not found")
		return
	}
	var req ApplyCartVoucherRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	var voucher models.Voucher
	if err := database.DB.Where("code = ? AND is_active = true", req.Code).First(&voucher).Error; err != nil {
		RespondError(c, http.StatusBadRequest, "voucher tidak ditemukan atau tidak aktif")
		return
	}
	recalcCartTotals(&cart)
	if cart.Subtotal < voucher.MinSpend {
		RespondError(c, http.StatusBadRequest, "belum memenuhi minimum belanja voucher")
		return
	}
	discount := 0.0
	if voucher.Type == "percent" {
		discount = cart.Subtotal * (voucher.Value / 100)
		if voucher.MaxDiscount != nil && discount > *voucher.MaxDiscount {
			discount = *voucher.MaxDiscount
		}
	} else {
		discount = voucher.Value
	}
	cart.VoucherCode = voucher.Code
	cart.VoucherDiscount = discount
	recalcCartTotals(&cart)
	database.DB.Save(&cart)
	RespondSuccess(c, cart)
}

type CartPaymentInput struct {
	Method          string  `json:"method" binding:"required"`
	Amount          float64 `json:"amount" binding:"required,gt=0"`
	CashReceived    float64 `json:"cash_received"`
	ReferenceNumber string  `json:"reference_number"`
}

type CheckoutCartRequest struct {
	PaymentMethod string             `json:"payment_method"`
	CashReceived  float64            `json:"cash_received"`
	Payments      []CartPaymentInput `json:"payments"`
}

// CheckoutCart implements docs/kasir.md POST /carts/:id/checkout (supports single payment & split payments).
func CheckoutCart(c *gin.Context) {
	if cached, ok := idempotencyReplay(c, "cart.checkout"); ok {
		c.JSON(http.StatusCreated, cached)
		return
	}

	id := c.Param("id")
	var cart models.Cart
	if err := database.DB.Preload("Items").First(&cart, id).Error; err != nil {
		RespondError(c, http.StatusNotFound, "Cart not found")
		return
	}
	if cart.Status == "converted" {
		RespondError(c, http.StatusConflict, "cart sudah checkout")
		return
	}
	var req CheckoutCartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, err.Error())
		return
	}

	recalcCartTotals(&cart)

	// Consolidate payment list
	var paymentsToProcess []CartPaymentInput
	if len(req.Payments) > 0 {
		paymentsToProcess = req.Payments
	} else if req.PaymentMethod != "" {
		paymentsToProcess = append(paymentsToProcess, CartPaymentInput{
			Method:       req.PaymentMethod,
			Amount:       cart.Total,
			CashReceived: req.CashReceived,
		})
	} else {
		RespondError(c, http.StatusBadRequest, "payment_method or payments array is required")
		return
	}

	// Calculate total amount paid and change
	var totalPaid float64
	var totalCashChange float64
	methods := make([]string, 0, len(paymentsToProcess))
	for _, p := range paymentsToProcess {
		totalPaid += p.Amount
		methods = append(methods, p.Method)
		if p.Method == "cash" && p.CashReceived > p.Amount {
			totalCashChange += (p.CashReceived - p.Amount)
		}
	}

	if totalPaid < cart.Total {
		RespondError(c, http.StatusBadRequest, fmt.Sprintf("total pembayaran (%.2f) kurang dari tagihan (%.2f)", totalPaid, cart.Total))
		return
	}

	primaryMethod := methods[0]
	if len(methods) > 1 {
		primaryMethod = "split"
	}

	var store models.Store
	if err := database.DB.First(&store, cart.StoreID).Error; err != nil {
		RespondError(c, http.StatusBadRequest, "store not found")
		return
	}

	tx := database.DB.Begin()
	var wh models.Warehouse
	if err := tx.Where("store_id = ?", cart.StoreID).First(&wh).Error; err != nil {
		// fallback to any warehouse owned by company
		if err2 := tx.Where("company_id = ?", store.CompanyID).First(&wh).Error; err2 != nil {
			tx.Rollback()
			RespondError(c, http.StatusBadRequest, "warehouse not found for store")
			return
		}
	}

	st := models.SalesTransaction{
		CompanyID:     store.CompanyID,
		StoreID:       cart.StoreID,
		ShiftID:       &cart.ShiftID,
		EmployeeID:    cart.EmployeeID,
		CustomerID:    cart.CustomerID,
		Subtotal:      cart.Subtotal,
		VoucherAmount: cart.VoucherDiscount,
		TotalAmount:   cart.Total,
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

	for _, it := range cart.Items {
		var stock models.Stock
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("warehouse_id = ? AND product_id = ?", wh.ID, it.ProductID).First(&stock).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				tx.Rollback()
				RespondError(c, http.StatusBadRequest, fmt.Sprintf("stock not found for product %d in warehouse", it.ProductID))
				return
			}
			tx.Rollback()
			RespondError(c, http.StatusInternalServerError, "failed to fetch stock")
			return
		}
		qty := int(it.Qty)
		if stock.Quantity < qty {
			tx.Rollback()
			RespondError(c, http.StatusBadRequest, fmt.Sprintf("insufficient stock for product %s (available %d, requested %d)", it.NameSnapshot, stock.Quantity, qty))
			return
		}
		stock.Quantity -= qty
		tx.Save(&stock)
		tx.Create(&models.StockMovement{
			CompanyID: store.CompanyID, ProductID: it.ProductID, StoreID: cart.StoreID,
			Delta: -it.Qty, Reason: "sale", RefType: "sale", RefID: &st.ID,
		})
		tx.Create(&models.SalesItem{
			SalesTransactionID: st.ID, ProductID: it.ProductID, Quantity: qty,
			Price: it.Price, Subtotal: it.Price * it.Qty,
		})
	}

	// Record each payment into payments table (supporting split payment!)
	for _, p := range paymentsToProcess {
		change := 0.0
		if p.Method == "cash" && p.CashReceived > p.Amount {
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
			RespondError(c, http.StatusInternalServerError, "Failed to record payment: "+err.Error())
			return
		}
	}

	if cart.VoucherCode != "" {
		tx.Model(&models.Voucher{}).Where("code = ?", cart.VoucherCode).UpdateColumn("used_count", gorm.Expr("used_count + 1"))
	}

	cart.Status = "converted"
	tx.Save(&cart)
	tx.Commit()

	response := gin.H{
		"sale_id":        st.ID,
		"invoice_no":     st.InvoiceNo,
		"payment_method": primaryMethod,
		"payments":       paymentsToProcess,
		"total_amount":   cart.Total,
		"total_paid":     totalPaid,
		"change":         totalCashChange,
	}
	idempotencySave(c, "cart.checkout", response)
	incrementTransactionQuota(store.CompanyID)
	dispatchWebhookEvent(store.CompanyID, "sale.created", response)
	RespondCreated(c, response, "Checkout completed successfully")
}

// GetHoldOrders implements docs/kasir.md GET /cashier/hold-orders?cashierId=
func GetHoldOrders(c *gin.Context) {
	companyID := getAuthCompanyID(c)
	var list []models.Cart
	q := database.DB.Preload("Items").Where("status = 'held'")
	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	}
	if cashierID := c.Query("cashier_id"); cashierID != "" {
		q = q.Where("employee_id = ?", cashierID)
	}

	if c.Query("page") != "" || c.Query("per_page") != "" {
		var total int64
		q.Count(&total)
		page, perPage, offset := GetPagination(c)
		q.Order("created_at desc").Limit(perPage).Offset(offset).Find(&list)
		RespondPaginated(c, list, page, perPage, total)
		return
	}

	q.Order("created_at desc").Find(&list)
	c.JSON(http.StatusOK, list)
}
