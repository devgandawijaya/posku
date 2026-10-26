package controllers

import (
	"net/http"
	"time"

	"posku/internal/database"
	"posku/internal/models"

	"github.com/gin-gonic/gin"
)

type CreateExpenseRequest struct {
	CompanyID uint      `json:"company_id" binding:"required"`
	StoreID   *uint     `json:"store_id"`
	Category  string    `json:"category" binding:"required"`
	Amount    float64   `json:"amount" binding:"required,gt=0"`
	Date      time.Time `json:"date" binding:"required"`
	Note      string    `json:"note"`
	CreatedBy uint      `json:"created_by"`
}

func CreateExpense(c *gin.Context) {
	var req CreateExpenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	e := models.Expense{
		CompanyID: req.CompanyID,
		StoreID:   req.StoreID,
		Category:  req.Category,
		Amount:    req.Amount,
		Date:      req.Date,
		Note:      req.Note,
		CreatedBy: req.CreatedBy,
	}
	if err := database.DB.Create(&e).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create expense"})
		return
	}
	logAudit(c, e.CompanyID, "create", "expense", e.ID, e.Category, nil)
	c.JSON(http.StatusCreated, e)
}

func GetExpenses(c *gin.Context) {
	var list []models.Expense
	q := database.DB.Model(&models.Expense{})
	if companyID := c.Query("company_id"); companyID != "" {
		q = q.Where("company_id = ?", companyID)
	}
	if storeID := c.Query("store_id"); storeID != "" {
		q = q.Where("store_id = ?", storeID)
	}
	if dateFrom := c.Query("date_from"); dateFrom != "" {
		q = q.Where("date >= ?", dateFrom)
	}
	if dateTo := c.Query("date_to"); dateTo != "" {
		q = q.Where("date <= ?", dateTo)
	}
	if err := q.Order("date desc").Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch expenses"})
		return
	}
	c.JSON(http.StatusOK, list)
}

func UpdateExpense(c *gin.Context) {
	id := c.Param("id")
	var e models.Expense
	if err := database.DB.First(&e, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Expense not found"})
		return
	}
	var req CreateExpenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	e.StoreID = req.StoreID
	e.Category = req.Category
	e.Amount = req.Amount
	e.Date = req.Date
	e.Note = req.Note
	database.DB.Save(&e)
	c.JSON(http.StatusOK, e)
}

func DeleteExpense(c *gin.Context) {
	id := c.Param("id")
	if err := database.DB.Delete(&models.Expense{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete expense"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Expense deleted"})
}

type CreatePayrollRequest struct {
	CompanyID  uint    `json:"company_id" binding:"required"`
	StoreID    uint    `json:"store_id" binding:"required"`
	EmployeeID uint    `json:"employee_id" binding:"required"`
	Period     string  `json:"period" binding:"required"`
	BaseSalary float64 `json:"base_salary" binding:"required,gt=0"`
	Allowance  float64 `json:"allowance"`
	Deduction  float64 `json:"deduction"`
}

func CreatePayroll(c *gin.Context) {
	var req CreatePayrollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	net := req.BaseSalary + req.Allowance - req.Deduction
	p := models.Payroll{
		CompanyID:  req.CompanyID,
		StoreID:    req.StoreID,
		EmployeeID: req.EmployeeID,
		Period:     req.Period,
		BaseSalary: req.BaseSalary,
		Allowance:  req.Allowance,
		Deduction:  req.Deduction,
		Net:        net,
		Status:     "draft",
	}
	if err := database.DB.Create(&p).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create payroll"})
		return
	}
	c.JSON(http.StatusCreated, p)
}

func GetPayrolls(c *gin.Context) {
	var list []models.Payroll
	q := database.DB.Model(&models.Payroll{})
	if companyID := c.Query("company_id"); companyID != "" {
		q = q.Where("company_id = ?", companyID)
	}
	if period := c.Query("period"); period != "" {
		q = q.Where("period = ?", period)
	}
	if err := q.Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch payrolls"})
		return
	}
	c.JSON(http.StatusOK, list)
}

func UpdatePayroll(c *gin.Context) {
	id := c.Param("id")
	var p models.Payroll
	if err := database.DB.First(&p, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Payroll not found"})
		return
	}
	var req CreatePayrollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p.BaseSalary = req.BaseSalary
	p.Allowance = req.Allowance
	p.Deduction = req.Deduction
	p.Net = req.BaseSalary + req.Allowance - req.Deduction
	database.DB.Save(&p)
	c.JSON(http.StatusOK, p)
}

func DeletePayroll(c *gin.Context) {
	id := c.Param("id")
	if err := database.DB.Delete(&models.Payroll{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete payroll"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Payroll deleted"})
}

// PayPayroll marks a payroll as paid.
func PayPayroll(c *gin.Context) {
	id := c.Param("id")
	var p models.Payroll
	if err := database.DB.First(&p, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Payroll not found"})
		return
	}
	now := time.Now()
	p.Status = "paid"
	p.PaidAt = &now
	database.DB.Save(&p)
	c.JSON(http.StatusOK, p)
}

type CreateRentContractRequest struct {
	CompanyID   uint       `json:"company_id" binding:"required"`
	StoreID     uint       `json:"store_id" binding:"required"`
	MonthlyRent float64    `json:"monthly_rent" binding:"required,gt=0"`
	StartDate   time.Time  `json:"start_date" binding:"required"`
	EndDate     *time.Time `json:"end_date"`
}

// CreateRentContract implements docs/laporan-keuangan.md rent_contracts.
func CreateRentContract(c *gin.Context) {
	var req CreateRentContractRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	rc := models.RentContract{
		CompanyID: req.CompanyID, StoreID: req.StoreID, MonthlyRent: req.MonthlyRent,
		StartDate: req.StartDate, EndDate: req.EndDate, Status: "aktif",
	}
	if err := database.DB.Create(&rc).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create rent contract"})
		return
	}
	c.JSON(http.StatusCreated, rc)
}

func GetRentContracts(c *gin.Context) {
	var list []models.RentContract
	q := database.DB.Model(&models.RentContract{})
	if companyID := c.Query("company_id"); companyID != "" {
		q = q.Where("company_id = ?", companyID)
	}
	if storeID := c.Query("store_id"); storeID != "" {
		q = q.Where("store_id = ?", storeID)
	}
	q.Find(&list)
	c.JSON(http.StatusOK, list)
}

func UpdateRentContract(c *gin.Context) {
	id := c.Param("id")
	var rc models.RentContract
	if err := database.DB.First(&rc, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Rent contract not found"})
		return
	}
	var req CreateRentContractRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	rc.MonthlyRent = req.MonthlyRent
	rc.StartDate = req.StartDate
	rc.EndDate = req.EndDate
	database.DB.Save(&rc)
	c.JSON(http.StatusOK, rc)
}

func DeleteRentContract(c *gin.Context) {
	id := c.Param("id")
	if err := database.DB.Delete(&models.RentContract{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete rent contract"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Rent contract deleted"})
}
