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
	Qty         float64 `json:"qty" binding:"required,gt=0"`
	Price       float64 `json:"price" binding:"required,gt=0"`
	DiscountPct float64 `json:"discount_pct"`
	Note        string  `json:"note"`
}

type CreateCartRequest struct {
	CompanyID  uint            `json:"company_id" binding:"required"`
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

// CreateCart implements docs/kasir.md POST /carts (create / hold).
func CreateCart(c *gin.Context) {
	var req CreateCartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	status := req.Status
	if status == "" {
		status = "active"
	}
	cart := models.Cart{
		CompanyID:  req.CompanyID,
		ShiftID:    req.ShiftID,
		EmployeeID: req.EmployeeID,
		StoreID:    req.StoreID,
		CustomerID: req.CustomerID,
		Status:     status,
	}
	for _, it := range req.Items {
		var product models.Product
		if err := database.DB.First(&product, it.ProductID).Error; err == nil {
			cart.Items = append(cart.Items, models.CartItem{
				ProductID:    it.ProductID,
				SKUSnapshot:  product.SKU,
				NameSnapshot: product.Name,
				Price:        it.Price,
				Qty:          it.Qty,
				DiscountPct:  it.DiscountPct,
				Note:         it.Note,
			})
		}
	}
	recalcCartTotals(&cart)
	if err := database.DB.Create(&cart).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create cart"})
		return
	}
	c.JSON(http.StatusCreated, cart)
}

type UpdateCartItemsRequest struct {
	Items []CartItemInput `json:"items" binding:"required"`
}

// UpdateCartItems implements docs/kasir.md PATCH /carts/:id/items (replace items).
func UpdateCartItems(c *gin.Context) {
	id := c.Param("id")
	var cart models.Cart
	if err := database.DB.Preload("Items").First(&cart, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cart not found"})
		return
	}
	var req UpdateCartItemsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	database.DB.Where("cart_id = ?", cart.ID).Delete(&models.CartItem{})
	cart.Items = nil
	for _, it := range req.Items {
		var product models.Product
		if err := database.DB.First(&product, it.ProductID).Error; err != nil {
			continue
		}
		item := models.CartItem{
			CartID:       cart.ID,
			ProductID:    it.ProductID,
			SKUSnapshot:  product.SKU,
			NameSnapshot: product.Name,
			Price:        it.Price,
			Qty:          it.Qty,
			DiscountPct:  it.DiscountPct,
			Note:         it.Note,
		}
		database.DB.Create(&item)
		cart.Items = append(cart.Items, item)
	}
	recalcCartTotals(&cart)
	database.DB.Save(&cart)
	c.JSON(http.StatusOK, cart)
}

type ApplyCartVoucherRequest struct {
	Code string `json:"code" binding:"required"`
}

// ApplyCartVoucher implements docs/kasir.md POST /carts/:id/apply-voucher
func ApplyCartVoucher(c *gin.Context) {
	id := c.Param("id")
	var cart models.Cart
	if err := database.DB.Preload("Items").First(&cart, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cart not found"})
		return
	}
	var req ApplyCartVoucherRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var voucher models.Voucher
	if err := database.DB.Where("code = ? AND is_active = true", req.Code).First(&voucher).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "voucher tidak ditemukan atau tidak aktif"})
		return
	}
	recalcCartTotals(&cart)
	if cart.Subtotal < voucher.MinSpend {
		c.JSON(http.StatusBadRequest, gin.H{"error": "belum memenuhi minimum belanja voucher"})
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
	c.JSON(http.StatusOK, cart)
}

type CheckoutCartRequest struct {
	PaymentMethod string  `json:"payment_method" binding:"required,oneof=cash qris debit credit ewallet"`
	CashReceived  float64 `json:"cash_received"`
}

// CheckoutCart implements docs/kasir.md POST /carts/:id/checkout (convert cart -> sales + payment, decrement stock).
func CheckoutCart(c *gin.Context) {
	if cached, ok := idempotencyReplay(c, "cart.checkout"); ok {
		c.JSON(http.StatusCreated, cached)
		return
	}

	id := c.Param("id")
	var cart models.Cart
	if err := database.DB.Preload("Items").First(&cart, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cart not found"})
		return
	}
	if cart.Status == "converted" {
		c.JSON(http.StatusConflict, gin.H{"error": "cart sudah checkout"})
		return
	}
	var req CheckoutCartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var store models.Store
	if err := database.DB.First(&store, cart.StoreID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "store not found"})
		return
	}

	tx := database.DB.Begin()
	var wh models.Warehouse
	if err := tx.Where("store_id = ?", cart.StoreID).First(&wh).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "warehouse not found for store"})
		return
	}

	recalcCartTotals(&cart)
	st := models.SalesTransaction{
		CompanyID:     store.CompanyID,
		StoreID:       cart.StoreID,
		ShiftID:       &cart.ShiftID,
		EmployeeID:    cart.EmployeeID,
		CustomerID:    cart.CustomerID,
		Subtotal:      cart.Subtotal,
		VoucherAmount: cart.VoucherDiscount,
		TotalAmount:   cart.Total,
		PaymentMethod: req.PaymentMethod,
		Status:        "lunas",
		SyncStatus:    "synced",
	}
	if err := tx.Create(&st).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create sales transaction"})
		return
	}
	st.InvoiceNo = fmt.Sprintf("INV-%06d", st.ID)
	tx.Model(&st).Update("invoice_no", st.InvoiceNo)

	for _, it := range cart.Items {
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
		qty := int(it.Qty)
		if stock.Quantity < qty {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient stock"})
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

	change := req.CashReceived - cart.Total
	tx.Create(&models.Payment{SalesTransactionID: st.ID, Method: req.PaymentMethod, Amount: cart.Total})

	if cart.VoucherCode != "" {
		tx.Model(&models.Voucher{}).Where("code = ?", cart.VoucherCode).UpdateColumn("used_count", gorm.Expr("used_count + 1"))
	}

	cart.Status = "converted"
	tx.Save(&cart)
	tx.Commit()

	response := gin.H{"sale_id": st.ID, "invoice_no": st.InvoiceNo, "payment": req.PaymentMethod, "change": change}
	idempotencySave(c, "cart.checkout", response)
	incrementTransactionQuota(store.CompanyID)
	dispatchWebhookEvent(store.CompanyID, "sale.created", response)
	c.JSON(http.StatusCreated, response)
}

// GetHoldOrders implements docs/kasir.md GET /cashier/hold-orders?cashierId=
func GetHoldOrders(c *gin.Context) {
	var list []models.Cart
	q := database.DB.Preload("Items").Where("status = 'held'")
	if cashierID := c.Query("cashier_id"); cashierID != "" {
		q = q.Where("employee_id = ?", cashierID)
	}
	q.Find(&list)
	c.JSON(http.StatusOK, list)
}
