package database

import (
	"log"
	"time"
	"posku/internal/models"
)

func SeedSampleData() {
	var companyCount int64
	DB.Model(&models.Company{}).Count(&companyCount)
	if companyCount > 0 {
		return
	}

	comp := models.Company{Name: "PT Posku Utama", Address: "Jl. Sudirman No. 1, Jakarta"}
	DB.Create(&comp)

	roleAdmin := models.Role{CompanyID: comp.ID, Name: "admin", Scope: "company", Permissions: `["*"]`}
	roleCashier := models.Role{CompanyID: comp.ID, Name: "kasir", Scope: "store", Permissions: `["cashier.*"]`}
	DB.Create(&roleAdmin)
	DB.Create(&roleCashier)

	store := models.Store{CompanyID: comp.ID, Code: "STR-001", Name: "Toko Utama", Address: "Jl. Sudirman No. 1", Phone: "08123456789", Status: "aktif"}
	DB.Create(&store)

	wh := models.Warehouse{StoreID: store.ID, Name: "Gudang Utama", Location: "Jl. Sudirman No. 1"}
	DB.Create(&wh)

	cat := models.Category{CompanyID: comp.ID, Name: "Makanan & Minuman", Code: "FNB"}
	DB.Create(&cat)

	prod := models.Product{CompanyID: comp.ID, CategoryID: &cat.ID, SKU: "PRD-001", Barcode: "899123456789", Name: "Kopi Kenangan", Description: "Kopi Susu Gula Aren", Price: 18000, Cost: 8000, Unit: "cup", IsActive: true}
	DB.Create(&prod)

	bc := models.ProductBarcode{ProductID: prod.ID, Barcode: "899123456789"}
	DB.Create(&bc)

	stk := models.Stock{ProductID: prod.ID, WarehouseID: wh.ID, Quantity: 100}
	DB.Create(&stk)

	empAdmin := models.Employee{CompanyID: comp.ID, StoreID: &store.ID, RoleID: roleAdmin.ID, Name: "Admin Utama", Username: "admin", Email: "admin@posku.com", Password: "$2a$10$wWvH3a7m5N6Cj7.yJ1s2e.V9YJkKx.R9Jq4C8N7K6M5L4O3P2Q1R", Status: "aktif"}
	empCashier := models.Employee{CompanyID: comp.ID, StoreID: &store.ID, RoleID: roleCashier.ID, Name: "Kasir Budi", Username: "kasir", Email: "kasir@posku.com", Password: "$2a$10$wWvH3a7m5N6Cj7.yJ1s2e.V9YJkKx.R9Jq4C8N7K6M5L4O3P2Q1R", Status: "aktif"}
	DB.Create(&empAdmin)
	DB.Create(&empCashier)

	cust := models.Customer{CompanyID: comp.ID, MemberCode: "CUST-001", Name: "Pelanggan Setia", Email: "cust@gmail.com", Phone: "08987654321", Tier: "bronze", PointsBalance: 50}
	DB.Create(&cust)

	supp := models.Supplier{CompanyID: comp.ID, Code: "SUP-001", Name: "CV Kopi Nusantara", ContactPerson: "Bpk. Budi", Phone: "08111122233", Status: "aktif"}
	DB.Create(&supp)

	shift := models.Shift{CompanyID: comp.ID, StoreID: store.ID, EmployeeID: empCashier.ID, OpeningCash: 100000, Status: "open"}
	DB.Create(&shift)

	vouch := models.Voucher{CompanyID: comp.ID, Code: "DISC10", Label: "Diskon 10%", Type: "percent", Value: 10, MinSpend: 20000, IsActive: true}
	promo := models.Promotion{CompanyID: comp.ID, Name: "Promo Makanan", Type: "percent", Value: 5, IsActive: true}
	DB.Create(&vouch)
	DB.Create(&promo)

	trx := models.SalesTransaction{CompanyID: comp.ID, StoreID: store.ID, EmployeeID: empCashier.ID, CustomerID: &cust.ID, InvoiceNo: "INV-20260902-0001", Status: "completed", PaymentMethod: "cash", Subtotal: 18000, Tax: 0, Discount: 0, TotalAmount: 18000}
	DB.Create(&trx)

	item := models.SalesItem{SalesTransactionID: trx.ID, ProductID: prod.ID, Quantity: 1, Price: 18000, Subtotal: 18000}
	DB.Create(&item)

	pay := models.Payment{SalesTransactionID: trx.ID, Amount: 18000, Method: "cash"}
	DB.Create(&pay)

	exp := models.Expense{CompanyID: comp.ID, StoreID: &store.ID, Category: "operasional", Amount: 50000, Note: "Beli Alat Tulis", Date: time.Now()}
	payr := models.Payroll{CompanyID: comp.ID, StoreID: store.ID, EmployeeID: empCashier.ID, Period: "2026-08", BaseSalary: 3000000, Net: 3000000, Status: "paid"}
	rent := models.RentContract{CompanyID: comp.ID, StoreID: store.ID, MonthlyRent: 1000000, Status: "aktif"}
	DB.Create(&exp)
	DB.Create(&payr)
	DB.Create(&rent)

	adj := models.StockAdjustment{CompanyID: comp.ID, StoreID: store.ID, Status: "approved", Note: "Stok awal opname"}
	DB.Create(&adj)

	mov := models.StockMovement{CompanyID: comp.ID, ProductID: prod.ID, StoreID: store.ID, Delta: 100, Reason: "initial", CreatedBy: empAdmin.ID}
	DB.Create(&mov)

	ret := models.Return{CompanyID: comp.ID, StoreID: store.ID, EmployeeID: empCashier.ID, SalesTransactionID: trx.ID, CustomerID: &cust.ID, Reason: "Salah pesanan", Status: "completed", TotalRefund: 18000}
	DB.Create(&ret)

	integ := models.IntegrationInstallation{CompanyID: comp.ID, IntegrationID: "midtrans", Status: "connected"}
	webh := models.Webhook{CompanyID: comp.ID, URL: "https://example.com/webhook", Events: `["sale.created"]`, IsActive: true}
	DB.Create(&integ)
	DB.Create(&webh)

	var proPlan models.Plan
	DB.Where("code = ?", "pro").First(&proPlan)
	planID := uint(1)
	if proPlan.ID > 0 {
		planID = proPlan.ID
	}

	sub := models.Subscription{CompanyID: comp.ID, PlanID: planID, Status: "active", BillingCycle: "monthly"}
	DB.Create(&sub)

	inv := models.Invoice{CompanyID: comp.ID, SubscriptionID: sub.ID, InvoiceNo: "INV-SUB-001", Subtotal: 299000, Total: 299000, Status: "paid"}
	dev := models.Device{CompanyID: comp.ID, StoreID: store.ID, Name: "Kasir Tablet 1", Type: "tablet", Status: "online"}
	DB.Create(&inv)
	DB.Create(&dev)

	audit := models.AuditLog{CompanyID: comp.ID, ActorID: empAdmin.ID, ActorName: empAdmin.Name, Action: "user_login", EntityType: "auth", EntityID: 1, EntityName: "login", Diff: `{"status":"success"}`}
	DB.Create(&audit)

	log.Println("Sample data seeded successfully")
}
