package routes

import (
	"posku/internal/controllers"
	"posku/internal/middlewares"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	// Add CORS middleware to all routes
	r.Use(middlewares.CORSMiddleware())

	r.GET("/", controllers.Home)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Public / auth
	r.POST("/employees/login", controllers.EmployeeLogin)
	r.POST("/employees/register", controllers.CreateEmployee)
	r.POST("/auth/login", controllers.EmployeeLogin)
	r.POST("/auth/refresh", controllers.RefreshAccessToken)
	r.POST("/auth/logout", controllers.EmployeeLogout)
	r.GET("/auth/me", middlewares.AuthRequired(), controllers.EmployeeMe)
	r.POST("/auth/change-password", middlewares.AuthRequired(), controllers.EmployeeChangePassword)
	r.POST("/auth/forgot-password", controllers.ForgotPassword)
	r.POST("/auth/reset-password", controllers.ResetPasswordViaToken)
	r.POST("/auth/2fa/setup", middlewares.AuthRequired(), controllers.SetupTwoFactor)
	r.POST("/auth/2fa/enable", middlewares.AuthRequired(), controllers.EnableTwoFactor)
	r.POST("/auth/2fa/disable", middlewares.AuthRequired(), controllers.DisableTwoFactor)
	r.POST("/auth/2fa/verify-login", controllers.VerifyLoginTwoFactor)

	// ===========================
	// Company routes
	// ===========================
	r.POST("/companies", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.CreateCompany)
	r.GET("/companies", controllers.GetCompanies)

	// Specific company route FIRST
	r.GET("/companies/:id/stores", controllers.GetStoresByCompany)

	// Generic company routes
	r.GET("/companies/:id", controllers.GetCompany)
	r.PUT("/companies/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.UpdateCompany)
	r.DELETE("/companies/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.DeleteCompany)

	// ===========================
	// Store routes
	// ===========================
	r.POST("/stores", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.CreateStore)
	r.GET("/stores", controllers.GetStores)

	// Specific store route FIRST (must match wildcard naming)
	r.GET("/stores/:id/warehouse", controllers.GetWarehouseByStore)

	// Generic store routes
	r.GET("/stores/:id", controllers.GetStore)
	r.PUT("/stores/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.UpdateStore)
	r.DELETE("/stores/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.DeleteStore)
	r.POST("/stores/:id/toggle-status", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.ToggleStoreStatus)
	r.GET("/stores/:id/metrics", middlewares.AuthRequired(), controllers.GetStoreMetrics)
	r.POST("/stores/:id/metrics/refresh", middlewares.AuthRequired(), controllers.RefreshOutletMetricsCache)
	r.GET("/stores/options", controllers.GetStoreOptions)

	// ===========================
	// Supplier routes
	// ===========================
	r.POST("/suppliers", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.CreateSupplier)
	r.GET("/suppliers", middlewares.AuthRequired(), controllers.GetSuppliers)
	r.GET("/suppliers/:id", middlewares.AuthRequired(), controllers.GetSupplier)
	r.PATCH("/suppliers/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.UpdateSupplier)
	r.DELETE("/suppliers/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.DeleteSupplier)
	r.POST("/suppliers/:id/toggle-status", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.ToggleSupplierStatus)
	r.POST("/suppliers/:id/assign-outlets", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.AssignSupplierOutlets)
	r.GET("/suppliers/options", middlewares.AuthRequired(), controllers.GetSupplierOptions)
	r.POST("/suppliers/:id/purchases", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.RecordSupplierPurchase)

	// ===========================
	// Warehouse routes
	// ===========================
	r.POST("/warehouses", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.CreateWarehouse)
	r.GET("/warehouses", controllers.GetWarehouses)
	r.GET("/warehouses/:id", controllers.GetWarehouse)
	r.PUT("/warehouses/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.UpdateWarehouse)
	r.DELETE("/warehouses/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.DeleteWarehouse)

	// ===========================
	// Product routes
	// ===========================
	r.POST("/products", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.CreateProduct)
	r.GET("/products", controllers.GetProducts)
	r.GET("/products/:id", controllers.GetProduct)
	r.PUT("/products/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.UpdateProduct)
	r.DELETE("/products/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.DeleteProduct)
	r.POST("/products/bulk-activate", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.BulkActivateProducts)
	r.POST("/products/bulk-deactivate", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.BulkDeactivateProducts)
	r.POST("/products/bulk-delete", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.BulkDeleteProducts)
	r.POST("/products/:id/barcodes", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.AddProductBarcode)
	r.GET("/products/:id/barcodes", middlewares.AuthRequired(), controllers.GetProductBarcodes)
	r.GET("/products/:id/price-history", middlewares.AuthRequired(), controllers.GetProductPriceHistory)
	r.POST("/products/:id/images", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.AddProductImage)
	r.GET("/products/:id/images", middlewares.AuthRequired(), controllers.GetProductImages)
	r.DELETE("/products/:id/images/:imageId", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.DeleteProductImage)

	// ===========================
	// Employee routes
	// ===========================
	r.POST("/employees", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.CreateEmployee)
	r.GET("/employees", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.GetEmployees)
	r.GET("/employees/:id", middlewares.AuthRequired(), controllers.GetEmployee)
	r.PUT("/employees/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.UpdateEmployee)
	r.DELETE("/employees/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.DeleteEmployee)
	r.POST("/employees/:id/toggle-status", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.ToggleEmployeeStatus)
	r.POST("/employees/:id/assign-outlets", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.AssignEmployeeOutlets)
	r.POST("/employees/:id/reset-password", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.ResetEmployeePassword)
	r.GET("/users/options", middlewares.AuthRequired(), controllers.GetEmployeeOptions)
	r.GET("/users/by-outlet/:outletId", middlewares.AuthRequired(), controllers.GetEmployeesByOutlet)
	r.POST("/users/invite", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.InviteEmployee)
	r.POST("/users/accept-invite", controllers.AcceptInvite)
	r.POST("/users/bulk-import", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.BulkImportEmployees)
	r.POST("/users/:id/metrics/refresh", middlewares.AuthRequired(), controllers.RefreshUserMetrics)

	// ===========================
	// Role & Akses routes (RBAC)
	// ===========================
	r.POST("/roles", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.CreateRole)
	r.GET("/roles", middlewares.AuthRequired(), controllers.GetRoles)
	r.GET("/roles/:id", middlewares.AuthRequired(), controllers.GetRole)
	r.PATCH("/roles/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.UpdateRole)
	r.PATCH("/roles/:id/permissions", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.UpdateRolePermissions)
	r.POST("/roles/:id/toggle-status", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.ToggleRoleStatus)
	r.DELETE("/roles/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.DeleteRole)
	r.GET("/permissions/catalog", middlewares.AuthRequired(), controllers.GetPermissionsCatalog)

	// ===========================
	// Audit logs
	// ===========================
	r.GET("/audit", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.GetAuditLogs)
	r.GET("/audit/export", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.ExportAuditLogs)
	r.GET("/audit/entity/:entityType/:entityId", middlewares.AuthRequired(), controllers.GetAuditByEntity)
	r.GET("/audit/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.GetAuditLog)

	// ===========================
	// Category routes (Kategori)
	// ===========================
	r.POST("/categories", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.CreateCategory)
	r.GET("/categories", controllers.GetCategories)
	r.GET("/categories/:id", controllers.GetCategory)
	r.GET("/categories/:id/kpi", middlewares.AuthRequired(), controllers.GetCategoryKPI)
	r.POST("/categories/:id/kpi/refresh", middlewares.AuthRequired(), controllers.RefreshCategoryKPICache)
	r.GET("/categories/:id/kpi/cache", middlewares.AuthRequired(), controllers.GetCategoryKPICache)
	r.PATCH("/categories/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.UpdateCategory)
	r.DELETE("/categories/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.DeleteCategory)

	// ===========================
	// Customer routes
	// ===========================
	r.POST("/customers", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.CreateCustomer)
	r.GET("/customers", middlewares.AuthRequired(), controllers.GetCustomers)
	r.GET("/customers/:id", middlewares.AuthRequired(), controllers.GetCustomer)
	r.PUT("/customers/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.UpdateCustomer)
	r.DELETE("/customers/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.DeleteCustomer)
	r.POST("/customers/:id/points", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.AdjustCustomerPoints)
	r.GET("/customers/options", middlewares.AuthRequired(), controllers.GetCustomerOptions)
	r.GET("/customers/:id/transactions", middlewares.AuthRequired(), controllers.GetCustomerTransactions)
	r.POST("/customers/:id/metrics/refresh", middlewares.AuthRequired(), controllers.RefreshCustomerMetrics)

	// ===========================
	// Sales Transactions
	// ===========================
	r.POST("/sales", middlewares.AuthRequired(), controllers.CreateSalesTransaction)
	r.GET("/sales", middlewares.AuthRequired(), controllers.GetSalesTransactions)
	r.GET("/sales/:id", middlewares.AuthRequired(), controllers.GetSalesTransaction)
	r.DELETE("/sales/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.DeleteSalesTransaction)

	// Sales Items
	r.POST("/sales/items", middlewares.AuthRequired(), controllers.CreateSalesItem)
	r.GET("/sales/items", middlewares.AuthRequired(), controllers.GetSalesItems)
	r.GET("/sales/items/:id", middlewares.AuthRequired(), controllers.GetSalesItem)
	r.PUT("/sales/items/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.UpdateSalesItem)
	r.DELETE("/sales/items/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.DeleteSalesItem)

	// ===========================
	// Payments
	// ===========================
	r.POST("/payments", middlewares.AuthRequired(), controllers.CreatePayment)
	r.GET("/payments", middlewares.AuthRequired(), controllers.GetPayments)
	r.GET("/payments/:id", middlewares.AuthRequired(), controllers.GetPayment)
	r.PUT("/payments/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.UpdatePayment)
	r.DELETE("/payments/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.DeletePayment)

	// ===========================
	// Stock Transfers
	// ===========================
	r.POST("/stock_transfers", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.CreateStockTransfer)
	r.GET("/stock_transfers", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.GetStockTransfers)
	r.GET("/stock_transfers/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.GetStockTransfer)
	r.PUT("/stock_transfers/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.UpdateStockTransfer)
	r.DELETE("/stock_transfers/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.DeleteStockTransfer)

	// Transfer Items
	r.POST("/transfer_items", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.CreateTransferItem)
	r.GET("/transfer_items", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.GetTransferItems)
	r.GET("/transfer_items/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.GetTransferItem)
	r.PUT("/transfer_items/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.UpdateTransferItem)
	r.DELETE("/transfer_items/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.DeleteTransferItem)

	// ===========================
	// Returns (Retur & Refund)
	// ===========================
	r.POST("/returns", middlewares.AuthRequired(), controllers.CreateReturn)
	r.GET("/returns", middlewares.AuthRequired(), controllers.GetReturns)
	r.GET("/returns/:id", middlewares.AuthRequired(), controllers.GetReturn)
	r.POST("/returns/:id/approve", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.ApproveReturn)
	r.POST("/returns/:id/reject", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.RejectReturn)
	r.POST("/returns/:id/process", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.ProcessReturn)
	r.POST("/returns/:id/complete", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.CompleteReturn)
	r.POST("/returns/:id/refund", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.RefundReturn)
	r.GET("/returns/:id/refund-payments", middlewares.AuthRequired(), controllers.GetReturnRefundPayments)

	// ===========================
	// Sales void/refund (Transaksi)
	// ===========================
	r.POST("/sales/:id/void", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.VoidSalesTransaction)
	r.POST("/sales/:id/refund", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.RefundSalesTransaction)
	r.POST("/sales/:id/sync", middlewares.AuthRequired(), controllers.SyncSalesTransaction)
	r.GET("/sales/export", middlewares.AuthRequired(), controllers.ExportSalesTransactions)
	r.POST("/sales/:id/print", middlewares.AuthRequired(), controllers.PrintSalesTransaction)
	r.GET("/sales/:id/audit", middlewares.AuthRequired(), controllers.GetSaleAudits)
	r.GET("/sales/:id/refunds", middlewares.AuthRequired(), controllers.GetSaleRefunds)

	// ===========================
	// Stock (movements/adjust/restock) - stok.md
	// ===========================
	r.GET("/stocks/:productId/movements", middlewares.AuthRequired(), controllers.GetStockMovements)
	r.GET("/stocks/:productId/by-store", middlewares.AuthRequired(), controllers.GetStockByStore)
	r.POST("/stocks/adjust", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.AdjustStock)
	r.POST("/stocks/:productId/restock", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.RestockProduct)
	r.POST("/stocks/adjustments", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.CreateStockAdjustment)
	r.GET("/stocks/adjustments", middlewares.AuthRequired(), controllers.GetStockAdjustments)
	r.POST("/stocks/adjustments/:id/submit", middlewares.AuthRequired(), controllers.SubmitStockAdjustment)
	r.POST("/stocks/adjustments/:id/approve", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.ApproveStockAdjustment)
	r.POST("/stocks/adjustments/:id/reject", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.RejectStockAdjustment)

	// ===========================
	// Kasir: Shifts, Vouchers, Promotions
	// ===========================
	r.POST("/shifts/open", middlewares.AuthRequired(), controllers.OpenShift)
	r.POST("/shifts/:id/close", middlewares.AuthRequired(), controllers.CloseShift)
	r.POST("/shifts/:id/cash-movement", middlewares.AuthRequired(), controllers.ShiftCashMovementHandler)
	r.GET("/shifts/active", middlewares.AuthRequired(), controllers.GetActiveShift)
	r.GET("/shifts", middlewares.AuthRequired(), controllers.GetShifts)

	r.POST("/vouchers", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.CreateVoucher)
	r.GET("/vouchers", middlewares.AuthRequired(), controllers.GetVouchers)
	r.GET("/cashier/vouchers", middlewares.AuthRequired(), controllers.ValidateVoucher)
	r.PATCH("/vouchers/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.UpdateVoucher)
	r.DELETE("/vouchers/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.DeleteVoucher)

	r.POST("/promotions", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.CreatePromotion)
	r.GET("/promotions", middlewares.AuthRequired(), controllers.GetPromotions)
	r.PATCH("/promotions/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.UpdatePromotion)
	r.DELETE("/promotions/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.DeletePromotion)

	// Carts (persisted, hold order, checkout)
	r.POST("/carts", middlewares.AuthRequired(), controllers.CreateCart)
	r.PATCH("/carts/:id/items", middlewares.AuthRequired(), controllers.UpdateCartItems)
	r.POST("/carts/:id/apply-voucher", middlewares.AuthRequired(), controllers.ApplyCartVoucher)
	r.POST("/carts/:id/checkout", middlewares.AuthRequired(), controllers.CheckoutCart)
	r.GET("/cashier/hold-orders", middlewares.AuthRequired(), controllers.GetHoldOrders)

	// ===========================
	// Finance: Expenses, Payrolls (laporan-keuangan.md)
	// ===========================
	r.POST("/expenses", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.CreateExpense)
	r.GET("/expenses", middlewares.AuthRequired(), controllers.GetExpenses)
	r.PATCH("/expenses/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.UpdateExpense)
	r.DELETE("/expenses/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.DeleteExpense)

	r.POST("/payrolls", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.CreatePayroll)
	r.GET("/payrolls", middlewares.AuthRequired(), controllers.GetPayrolls)
	r.PATCH("/payrolls/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.UpdatePayroll)
	r.DELETE("/payrolls/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.DeletePayroll)
	r.POST("/payrolls/:id/pay", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.PayPayroll)

	r.POST("/rent-contracts", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.CreateRentContract)
	r.GET("/rent-contracts", middlewares.AuthRequired(), controllers.GetRentContracts)
	r.PATCH("/rent-contracts/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.UpdateRentContract)
	r.DELETE("/rent-contracts/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.DeleteRentContract)

	// ===========================
	// Reports (laporan-penjualan, laporan-stok, laporan-keuangan)
	// ===========================
	r.GET("/reports/sales/summary", middlewares.AuthRequired(), controllers.GetSalesSummaryReport)
	r.GET("/reports/sales/timeseries", middlewares.AuthRequired(), controllers.GetSalesTimeseriesReport)
	r.GET("/reports/sales/by-payment", middlewares.AuthRequired(), controllers.GetSalesByPaymentReport)
	r.GET("/reports/sales/by-store", middlewares.AuthRequired(), controllers.GetSalesByStoreReport)
	r.GET("/reports/sales/top-products", middlewares.AuthRequired(), controllers.GetTopProductsReport)

	r.GET("/reports/stock/summary", middlewares.AuthRequired(), controllers.GetStockSummaryReport)
	r.GET("/reports/stock/by-category", middlewares.AuthRequired(), controllers.GetStockByCategoryReport)
	r.GET("/reports/stock/movement", middlewares.AuthRequired(), controllers.GetStockMovementReport)
	r.GET("/reports/stock/top-low", middlewares.AuthRequired(), controllers.GetTopLowStockReport)

	r.GET("/reports/finance/pl", middlewares.AuthRequired(), controllers.GetFinancePLReport)
	r.GET("/reports/finance/expense-breakdown", middlewares.AuthRequired(), controllers.GetFinanceExpenseBreakdownReport)
	r.GET("/reports/finance/cashflow", middlewares.AuthRequired(), controllers.GetFinanceCashflowReport)
	r.GET("/reports/sales/export", middlewares.AuthRequired(), controllers.ExportSalesReport)
	r.GET("/reports/stock/export", middlewares.AuthRequired(), controllers.ExportStockReport)
	r.GET("/reports/finance/export", middlewares.AuthRequired(), controllers.ExportFinanceReport)
	r.POST("/reports/sales/daily-cache/refresh", middlewares.AuthRequired(), controllers.RefreshSalesDailyCache)
	r.GET("/reports/sales/daily-cache", middlewares.AuthRequired(), controllers.GetSalesDailyCache)
	r.POST("/reports/stock/summary-cache/refresh", middlewares.AuthRequired(), controllers.RefreshStockSummaryCache)
	r.GET("/reports/stock/summary-cache", middlewares.AuthRequired(), controllers.GetStockSummaryCache)

	// ===========================
	// Dashboard (monitoring)
	// ===========================
	r.GET("/dashboard/summary", middlewares.AuthRequired(), controllers.GetDashboardSummary)
	r.GET("/dashboard/alerts", middlewares.AuthRequired(), controllers.GetDashboardAlerts)
	r.POST("/alerts/:id/acknowledge", middlewares.AuthRequired(), controllers.AcknowledgeAlert)
	r.POST("/alerts/:id/resolve", middlewares.AuthRequired(), controllers.ResolveAlert)
	r.GET("/dashboard/shifts/active", middlewares.AuthRequired(), controllers.GetDashboardShiftsActive)
	r.GET("/dashboard/payment-mix", middlewares.AuthRequired(), controllers.GetPaymentMix)
	r.GET("/dashboard/saas-metrics", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.GetSaasMetrics)
	r.POST("/dashboard/alerts/generate", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.GenerateDashboardAlerts)
	r.POST("/devices", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.CreateDevice)
	r.GET("/devices", middlewares.AuthRequired(), controllers.GetDevices)
	r.POST("/devices/:id/heartbeat", controllers.DeviceHeartbeat)

	// ===========================
	// Integrations
	// ===========================
	r.GET("/integrations/catalog", middlewares.AuthRequired(), controllers.GetIntegrationCatalog)
	r.GET("/integrations", middlewares.AuthRequired(), controllers.GetIntegrations)
	r.GET("/integrations/:id", middlewares.AuthRequired(), controllers.GetIntegration)
	r.POST("/integrations/:id/connect", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.ConnectIntegration)
	r.POST("/integrations/:id/disconnect", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.DisconnectIntegration)
	r.POST("/integrations/:id/test", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.TestIntegration)
	r.GET("/integrations/:id/logs", middlewares.AuthRequired(), controllers.GetIntegrationLogs)
	r.POST("/webhooks/in/:provider", controllers.InboundWebhook)
	r.POST("/webhooks", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.CreateWebhook)
	r.GET("/webhooks", middlewares.AuthRequired(), controllers.GetWebhooks)
	r.DELETE("/webhooks/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.DeleteWebhook)

	// ===========================
	// Subscription & Billing
	// ===========================
	r.GET("/billing/plans", controllers.GetPlans)
	r.GET("/billing/subscription", middlewares.AuthRequired(), controllers.GetSubscription)
	r.POST("/billing/subscription", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.CreateSubscription)
	r.POST("/billing/subscription/change-plan", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.ChangeSubscriptionPlan)
	r.POST("/billing/subscription/cancel", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.CancelSubscription)
	r.GET("/billing/invoices", middlewares.AuthRequired(), controllers.GetInvoices)
	r.GET("/billing/invoices/:id", middlewares.AuthRequired(), controllers.GetInvoice)
	r.POST("/billing/invoices/:id/pay", middlewares.AuthRequired(), controllers.PayInvoice)
	r.POST("/billing/invoices/generate", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.GenerateInvoice)
	r.GET("/billing/payment-methods", middlewares.AuthRequired(), controllers.GetPaymentMethods)
	r.POST("/billing/payment-methods", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.CreatePaymentMethod)
	r.DELETE("/billing/payment-methods/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.DeletePaymentMethod)
	r.POST("/billing/coupons", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.CreateCoupon)
	r.GET("/billing/coupons", middlewares.AuthRequired(), controllers.GetCoupons)
	r.GET("/billing/coupons/validate", middlewares.AuthRequired(), controllers.ValidateCoupon)
	r.GET("/billing/usage", middlewares.AuthRequired(), controllers.GetUsage)
}

