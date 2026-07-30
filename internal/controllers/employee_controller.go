package controllers

import (
	"net/http"
	"posku/internal/config"
	"posku/internal/database"
	"posku/internal/models"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type CreateEmployeeRequest struct {
	Name      string `json:"name" binding:"required"`
	Username  string `json:"username" binding:"required,alphanum"`
	Password  string `json:"password" binding:"required,min=6"`
	Role      string `json:"role" binding:"required,oneof=admin user"`
	CompanyID uint   `json:"company_id" binding:"required"`
	StoreID   *uint  `json:"store_id"`
}

type UpdateEmployeeRequest struct {
	Name      string `json:"name" binding:"required"`
	Username  string `json:"username" binding:"required,alphanum"`
	Password  string `json:"password" binding:"omitempty,min=6"`
	Role      string `json:"role" binding:"required,oneof=admin user"`
	CompanyID uint   `json:"company_id" binding:"required"`
	StoreID   *uint  `json:"store_id"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func CreateEmployee(c *gin.Context) {
	var req CreateEmployeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var company models.Company
	if err := database.DB.First(&company, req.CompanyID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "company not found"})
		return
	}

	if req.StoreID != nil {
		var store models.Store
		if err := database.DB.First(&store, *req.StoreID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "store not found"})
			return
		}
		if store.CompanyID != req.CompanyID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "store must belong to the same company"})
			return
		}
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	e := models.Employee{
		Name:      req.Name,
		Username:  req.Username,
		Password:  string(hashed),
		Role:      req.Role,
		CompanyID: req.CompanyID,
		StoreID:   req.StoreID,
	}

	if err := database.DB.Create(&e).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create employee"})
		return
	}

	e.Password = ""
	c.JSON(http.StatusCreated, e)
}

func GetEmployees(c *gin.Context) {
	var list []models.Employee
	if err := database.DB.Preload("Company").Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch employees"})
		return
	}
	for i := range list {
		list[i].Password = ""
	}
	c.JSON(http.StatusOK, list)
}

func GetEmployee(c *gin.Context) {
	id := c.Param("id")
	var e models.Employee
	if err := database.DB.Preload("Company").Preload("Store").First(&e, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Employee not found"})
		return
	}
	e.Password = ""
	c.JSON(http.StatusOK, e)
}

func UpdateEmployee(c *gin.Context) {
	id := c.Param("id")
	var e models.Employee
	if err := database.DB.First(&e, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Employee not found"})
		return
	}

	var req UpdateEmployeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	e.Name = req.Name
	e.Username = req.Username
	if req.Password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
			return
		}
		e.Password = string(hashed)
	}
	e.Role = req.Role
	e.CompanyID = req.CompanyID
	e.StoreID = req.StoreID

	if err := database.DB.Save(&e).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update employee"})
		return
	}

	e.Password = ""
	c.JSON(http.StatusOK, e)
}

func DeleteEmployee(c *gin.Context) {
	id := c.Param("id")
	var e models.Employee
	if err := database.DB.First(&e, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Employee not found"})
		return
	}

	if err := database.DB.Delete(&e).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete employee"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Employee deleted"})
}

func EmployeeLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var e models.Employee
	if err := database.DB.
		Preload("Company").
		Preload("Store").
		Where("username = ?", req.Username).
		First(&e).Error; err != nil {

		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	cleanPassword := strings.Trim(req.Password, " \r\n\t")
	if err := bcrypt.CompareHashAndPassword([]byte(e.Password), []byte(cleanPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials password"})
		return
	}

	// create JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"employee_id": e.ID,
		"username":    e.Username,
		"role":        e.Role,
		"exp":         time.Now().Add(time.Hour * 24).Unix(),
	})

	signed, err := token.SignedString([]byte(config.JWTSecret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create token"})
		return
	}

	e.Password = ""
	c.JSON(http.StatusOK, gin.H{"token": signed, "employee": e})
}
