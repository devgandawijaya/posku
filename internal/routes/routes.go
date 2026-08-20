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

	// ===========================
	// Employee routes
	// ===========================
	r.POST("/employees", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.CreateEmployee)
	r.GET("/employees", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.GetEmployees)
	r.GET("/employees/:id", middlewares.AuthRequired(), controllers.GetEmployee)
	r.PUT("/employees/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.UpdateEmployee)
	r.DELETE("/employees/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.DeleteEmployee)

	// ===========================
	// Customer routes
	// ===========================
	r.POST("/customers", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.CreateCustomer)
	r.GET("/customers", middlewares.AuthRequired(), controllers.GetCustomers)
	r.GET("/customers/:id", middlewares.AuthRequired(), controllers.GetCustomer)
	r.PUT("/customers/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.UpdateCustomer)
	r.DELETE("/customers/:id", middlewares.AuthRequired(), middlewares.RequireRoles("admin"), controllers.DeleteCustomer)

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
}
