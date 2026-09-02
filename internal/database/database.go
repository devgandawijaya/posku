package database

import (
	"fmt"
	"log"
	"posku/internal/config"
	"posku/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect() {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		config.DBHost, config.DBUser, config.DBPass, config.DBName, config.DBPort)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	log.Println("Database connected successfully")
}

func Migrate() {
	// Create enum types for PostgreSQL
	DB.Exec(`DO $$ BEGIN CREATE TYPE payment_method AS ENUM ('cash','transfer','wallet'); EXCEPTION WHEN duplicate_object THEN null; END $$;`)
	DB.Exec(`DO $$ BEGIN CREATE TYPE transfer_status AS ENUM ('pending','completed','cancelled'); EXCEPTION WHEN duplicate_object THEN null; END $$;`)

	err := DB.AutoMigrate(
		&models.Company{},
		&models.Role{},
		&models.Store{},
		&models.Warehouse{},
		&models.Category{},
		&models.Product{},
		&models.Stock{},
		&models.Employee{},
		&models.Customer{},
		&models.SalesTransaction{},
		&models.SalesItem{},
		&models.Payment{},
		&models.StockTransfer{},
		&models.TransferItem{},
		&models.Return{},
		&models.ReturnItem{},
		&models.AuditLog{},
		&models.RefreshToken{},
		&models.Supplier{},
		&models.StockMovement{},
		&models.Shift{},
		&models.ShiftCashMovement{},
		&models.Voucher{},
		&models.Promotion{},
		&models.Expense{},
		&models.Payroll{},
		&models.Alert{},
		&models.Integration{},
		&models.IntegrationInstallation{},
		&models.IntegrationLog{},
		&models.Plan{},
		&models.Subscription{},
		&models.Invoice{},
		&models.BillingPayment{},
		&models.PointsLedger{},
		&models.LoginAudit{},
		&models.PasswordReset{},
		&models.EmployeeInvitation{},
		&models.Cart{},
		&models.CartItem{},
		&models.StockAdjustment{},
		&models.StockAdjustmentItem{},
		&models.ProductBarcode{},
		&models.ProductPriceHistory{},
		&models.TenantPaymentMethod{},
		&models.Coupon{},
		&models.Webhook{},
		&models.WebhookInboundLog{},
		&models.ReturnApproval{},
		&models.EmployeeTOTP{},
		&models.SaleRefund{},
		&models.SaleAudit{},
		&models.RefundPayment{},
		&models.ProductImage{},
		&models.CategoryKPICache{},
		&models.OutletMetricsCache{},
		&models.CustomerMetricsCache{},
		&models.SupplierMetricsCache{},
		&models.SupplierPurchase{},
		&models.IdempotencyKey{},
		&models.Device{},
		&models.PlanQuota{},
		&models.UserMetricsCache{},
		&models.RentContract{},
		&models.SalesDailyCache{},
		&models.StockSummaryCache{},
	)
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	seedIntegrationCatalog()
	seedPlans()
	SeedSampleData()

	log.Println("Database migrated successfully")
}

// seedIntegrationCatalog inserts the static provider catalog (docs/integrasi.md) if not present.
func seedIntegrationCatalog() {
	apiKeySchema := `[{"key":"api_key","required":true},{"key":"webhook_url","required":false}]`
	noConfigSchema := `[]`
	catalog := []models.Integration{
		{ID: "midtrans", Provider: "Midtrans", Category: "payment", DisplayName: "Midtrans", Description: "Payment gateway Indonesia", Icon: "fa-credit-card", ConfigSchema: apiKeySchema},
		{ID: "xendit", Provider: "Xendit", Category: "payment", DisplayName: "Xendit", Description: "Payment gateway Indonesia", Icon: "fa-credit-card", ConfigSchema: apiKeySchema},
		{ID: "gojek", Provider: "Gojek", Category: "delivery", DisplayName: "GoSend", Description: "Layanan pengiriman", Icon: "fa-motorcycle", ConfigSchema: apiKeySchema},
		{ID: "whatsapp", Provider: "Meta", Category: "notification", DisplayName: "WhatsApp Business", Description: "Notifikasi pelanggan", Icon: "fa-whatsapp", ConfigSchema: apiKeySchema},
		{ID: "accurate", Provider: "Accurate", Category: "accounting", DisplayName: "Accurate Online", Description: "Sinkronisasi akuntansi", Icon: "fa-calculator", ConfigSchema: apiKeySchema},
		{ID: "shopee", Provider: "Shopee", Category: "marketplace", DisplayName: "Shopee", Description: "Marketplace", Icon: "fa-store", ConfigSchema: apiKeySchema},
		{ID: "tokopedia", Provider: "Tokopedia", Category: "ecommerce", DisplayName: "Tokopedia", Description: "E-commerce", Icon: "fa-store", ConfigSchema: apiKeySchema},
		{ID: "generic-api", Provider: "POSKU", Category: "api", DisplayName: "Public API", Description: "Akses API publik", Icon: "fa-code", ConfigSchema: noConfigSchema},
	}
	for _, item := range catalog {
		DB.FirstOrCreate(&item, models.Integration{ID: item.ID})
	}
}

// seedPlans inserts the default subscription plan catalog (docs/subscription-billing.md) if not present.
func seedPlans() {
	plans := []models.Plan{
		{Code: "free", Name: "Free", PriceMonthly: 0, PriceYearly: 0, Features: "{}", IsActive: true, SortOrder: 1},
		{Code: "pro", Name: "Pro", PriceMonthly: 299000, PriceYearly: 2990000, Features: "{\"multi_outlet\":true}", IsActive: true, SortOrder: 2},
		{Code: "enterprise", Name: "Enterprise", PriceMonthly: 999000, PriceYearly: 9990000, Features: "{\"multi_outlet\":true,\"custom_domain\":true}", IsActive: true, SortOrder: 3},
	}
	for _, p := range plans {
		var existing models.Plan
		if err := DB.Where("code = ?", p.Code).First(&existing).Error; err != nil {
			DB.Create(&p)
		}
	}
}
