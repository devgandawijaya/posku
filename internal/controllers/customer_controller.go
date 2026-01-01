package controllers

import (
	"net/http"
	"posku/internal/database"
	"posku/internal/models"

	"github.com/gin-gonic/gin"
)

type CreateCustomerRequest struct {
	CompanyID uint   `json:"company_id" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Email     string `json:"email" binding:"omitempty,email"`
	Phone     string `json:"phone" binding:"omitempty"`
}

type UpdateCustomerRequest struct {
	CompanyID uint   `json:"company_id" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Email     string `json:"email" binding:"omitempty,email"`
	Phone     string `json:"phone" binding:"omitempty"`
}

func CreateCustomer(c *gin.Context) {
	var req CreateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cu := models.Customer{
		CompanyID: req.CompanyID,
		Name:      req.Name,
		Email:     req.Email,
		Phone:     req.Phone,
	}

	if err := database.DB.Create(&cu).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create customer"})
		return
	}

	c.JSON(http.StatusCreated, cu)
}

func GetCustomers(c *gin.Context) {
	var list []models.Customer
	if err := database.DB.Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch customers"})
		return
	}
	c.JSON(http.StatusOK, list)
}

func GetCustomer(c *gin.Context) {
	id := c.Param("id")
	var cu models.Customer
	if err := database.DB.First(&cu, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
		return
	}
	c.JSON(http.StatusOK, cu)
}

func UpdateCustomer(c *gin.Context) {
	id := c.Param("id")
	var cu models.Customer
	if err := database.DB.First(&cu, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
		return
	}

	var input UpdateCustomerRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cu.Name = input.Name
	cu.Email = input.Email
	cu.Phone = input.Phone
	cu.CompanyID = input.CompanyID

	if err := database.DB.Save(&cu).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update customer"})
		return
	}

	c.JSON(http.StatusOK, cu)
}

func DeleteCustomer(c *gin.Context) {
	id := c.Param("id")
	var cu models.Customer
	if err := database.DB.First(&cu, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
		return
	}

	if err := database.DB.Delete(&cu).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete customer"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Customer deleted"})
}
