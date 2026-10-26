package controllers

import (
	"net/http"

	"posku/internal/database"
	"posku/internal/models"

	"github.com/gin-gonic/gin"
)

func currentEmployeeID(c *gin.Context) (uint, bool) {
	v, ok := c.Get("employee_id")
	if !ok {
		return 0, false
	}
	f, ok := v.(float64)
	if !ok {
		return 0, false
	}
	return uint(f), true
}

// SetupTwoFactor implements docs/auth.md 2FA TOTP setup (owner/admin). Generates a secret in
// disabled state; the client must confirm with POST /auth/2fa/enable before it takes effect.
func SetupTwoFactor(c *gin.Context) {
	empID, ok := currentEmployeeID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var e models.Employee
	if err := database.DB.First(&e, empID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Employee not found"})
		return
	}
	secret, err := generateTOTPSecret()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate secret"})
		return
	}

	var rec models.EmployeeTOTP
	if err := database.DB.Where("employee_id = ?", empID).First(&rec).Error; err != nil {
		rec = models.EmployeeTOTP{EmployeeID: empID, Secret: secret, Enabled: false}
		database.DB.Create(&rec)
	} else {
		rec.Secret = secret
		rec.Enabled = false
		database.DB.Save(&rec)
	}

	c.JSON(http.StatusOK, gin.H{
		"secret":           secret,
		"provisioning_uri": totpProvisioningURI("POSKU", e.Username, secret),
	})
}

type TwoFactorCodeRequest struct {
	Code string `json:"code" binding:"required,len=6"`
}

// EnableTwoFactor confirms setup by validating a code generated from the pending secret.
func EnableTwoFactor(c *gin.Context) {
	empID, ok := currentEmployeeID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req TwoFactorCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var rec models.EmployeeTOTP
	if err := database.DB.Where("employee_id = ?", empID).First(&rec).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "belum ada setup 2FA, panggil /auth/2fa/setup dahulu"})
		return
	}
	if !validateTOTPCode(rec.Secret, req.Code) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "kode tidak valid"})
		return
	}
	rec.Enabled = true
	database.DB.Save(&rec)
	logAudit(c, 0, "2fa_enable", "user", empID, "", nil)
	c.JSON(http.StatusOK, gin.H{"message": "2FA aktif"})
}

// DisableTwoFactor turns off 2FA for the authenticated employee.
func DisableTwoFactor(c *gin.Context) {
	empID, ok := currentEmployeeID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	database.DB.Model(&models.EmployeeTOTP{}).Where("employee_id = ?", empID).Update("enabled", false)
	logAudit(c, 0, "2fa_disable", "user", empID, "", nil)
	c.JSON(http.StatusOK, gin.H{"message": "2FA dinonaktifkan"})
}

type VerifyLoginTwoFactorRequest struct {
	EmployeeID uint   `json:"employee_id" binding:"required"`
	Code       string `json:"code" binding:"required,len=6"`
}

// VerifyLoginTwoFactor completes login when EmployeeLogin responded with requires_2fa=true.
func VerifyLoginTwoFactor(c *gin.Context) {
	var req VerifyLoginTwoFactorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var rec models.EmployeeTOTP
	if err := database.DB.Where("employee_id = ? AND enabled = true", req.EmployeeID).First(&rec).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "2FA tidak aktif untuk akun ini"})
		return
	}
	if !validateTOTPCode(rec.Secret, req.Code) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "kode tidak valid"})
		return
	}
	var e models.Employee
	if err := database.DB.Preload("Role").First(&e, req.EmployeeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Employee not found"})
		return
	}
	signed, err := generateJWT(e, e.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create token"})
		return
	}
	refreshToken, err := generateRefreshToken(e.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create refresh token"})
		return
	}
	e.Password = ""
	c.JSON(http.StatusOK, gin.H{
		"token":         signed,
		"refresh_token": refreshToken,
		"expires_in":    3600,
		"employee":      e,
		"permissions":   rolePermissionCodes(e.Role),
	})
}
