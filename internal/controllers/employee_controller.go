package controllers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
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

var errRoleCompanyMismatch = errors.New("role does not belong to company")

type CreateEmployeeRequest struct {
	Name      string `json:"name" binding:"required"`
	Username  string `json:"username" binding:"required,alphanum"`
	Email     string `json:"email"`
	Password  string `json:"password" binding:"required,min=6"`
	RoleID    uint   `json:"role_id" binding:"required"`
	CompanyID uint   `json:"company_id" binding:"required"`
	StoreID   *uint  `json:"store_id"`
	StoreIDs  []uint `json:"store_ids"`
}

type UpdateEmployeeRequest struct {
	Name      string `json:"name" binding:"required"`
	Username  string `json:"username" binding:"required,alphanum"`
	Email     string `json:"email"`
	Password  string `json:"password" binding:"omitempty,min=6"`
	RoleID    uint   `json:"role_id" binding:"required"`
	CompanyID uint   `json:"company_id" binding:"required"`
	StoreID   *uint  `json:"store_id"`
	StoreIDs  []uint `json:"store_ids"`
	Status    string `json:"status" binding:"omitempty,oneof=aktif nonaktif"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func rolePermissionCodes(role models.Role) []string {
	var matrix map[string]map[string]bool
	if err := json.Unmarshal([]byte(role.Permissions), &matrix); err != nil {
		return []string{}
	}
	codes := make([]string, 0)
	for module, actions := range matrix {
		for action, granted := range actions {
			if granted {
				codes = append(codes, module+"."+action)
			}
		}
	}
	return codes
}

func validateRoleAndStores(companyID, roleID uint, storeIDs []uint) (*models.Role, error) {
	var role models.Role
	if err := database.DB.First(&role, roleID).Error; err != nil {
		return nil, err
	}
	if role.CompanyID != companyID {
		return nil, errRoleCompanyMismatch
	}
	_ = storeIDs
	return &role, nil
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
	if ok, msg := checkPlanLimit(req.CompanyID, "users"); !ok {
		c.JSON(http.StatusPaymentRequired, gin.H{"error": msg})
		return
	}

	role, err := validateRoleAndStores(req.CompanyID, req.RoleID, req.StoreIDs)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role not found or not owned by company"})
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
		Email:     req.Email,
		Password:  string(hashed),
		RoleID:    role.ID,
		CompanyID: req.CompanyID,
		StoreID:   req.StoreID,
		StoreIDs:  encodeIDs(req.StoreIDs),
		Status:    "aktif",
	}

	if err := database.DB.Create(&e).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create employee"})
		return
	}

	logAudit(c, e.CompanyID, "user_create", "user", e.ID, e.Name, nil)
	e.Password = ""
	c.JSON(http.StatusCreated, e)
}

func GetEmployees(c *gin.Context) {
	var list []models.Employee
	q := database.DB.Preload("Company").Preload("Role")
	if companyID := c.Query("company_id"); companyID != "" {
		q = q.Where("company_id = ?", companyID)
	}
	if roleID := c.Query("role_id"); roleID != "" {
		q = q.Where("role_id = ?", roleID)
	}
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	if search := c.Query("q"); search != "" {
		q = q.Where("name ILIKE ? OR username ILIKE ? OR email ILIKE ?", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}
	if err := q.Find(&list).Error; err != nil {
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
	if err := database.DB.Preload("Company").Preload("Store").Preload("Role").First(&e, id).Error; err != nil {
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

	role, err := validateRoleAndStores(req.CompanyID, req.RoleID, req.StoreIDs)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role not found or not owned by company"})
		return
	}

	e.Name = req.Name
	e.Username = req.Username
	e.Email = req.Email
	if req.Password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
			return
		}
		e.Password = string(hashed)
	}
	e.RoleID = role.ID
	e.CompanyID = req.CompanyID
	e.StoreID = req.StoreID
	if req.StoreIDs != nil {
		e.StoreIDs = encodeIDs(req.StoreIDs)
	}
	if req.Status != "" {
		e.Status = req.Status
	}

	if err := database.DB.Save(&e).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update employee"})
		return
	}

	logAudit(c, e.CompanyID, "user_update", "user", e.ID, e.Name, nil)
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

	logAudit(c, e.CompanyID, "user_delete", "user", e.ID, e.Name, nil)
	c.JSON(http.StatusOK, gin.H{"message": "Employee deleted"})
}

// ToggleEmployeeStatus flips status between aktif/nonaktif (docs/karyawan.md).
func ToggleEmployeeStatus(c *gin.Context) {
	id := c.Param("id")
	var e models.Employee
	if err := database.DB.First(&e, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Employee not found"})
		return
	}
	if e.Status == "aktif" {
		e.Status = "nonaktif"
	} else {
		e.Status = "aktif"
	}
	database.DB.Save(&e)
	logAudit(c, e.CompanyID, "user_toggle", "user", e.ID, e.Name, nil)
	e.Password = ""
	c.JSON(http.StatusOK, e)
}

type AssignOutletsRequest struct {
	StoreIDs []uint `json:"store_ids" binding:"required"`
}

// AssignEmployeeOutlets replaces the employee's multi-outlet placement (docs/karyawan.md).
func AssignEmployeeOutlets(c *gin.Context) {
	id := c.Param("id")
	var e models.Employee
	if err := database.DB.First(&e, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Employee not found"})
		return
	}
	var req AssignOutletsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	for _, sid := range req.StoreIDs {
		var store models.Store
		if err := database.DB.First(&store, sid).Error; err != nil || store.CompanyID != e.CompanyID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "store must belong to the same company"})
			return
		}
	}
	e.StoreIDs = encodeIDs(req.StoreIDs)
	database.DB.Save(&e)
	logAudit(c, e.CompanyID, "store_assign", "user", e.ID, e.Name, req.StoreIDs)
	e.Password = ""
	c.JSON(http.StatusOK, e)
}

type ResetPasswordRequest struct {
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

func ResetEmployeePassword(c *gin.Context) {
	id := c.Param("id")
	var e models.Employee
	if err := database.DB.First(&e, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Employee not found"})
		return
	}
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}
	e.Password = string(hashed)
	database.DB.Save(&e)
	logAudit(c, e.CompanyID, "password_change", "user", e.ID, e.Name, nil)
	c.JSON(http.StatusOK, gin.H{"message": "password reset"})
}

// ---- Auth: login / refresh / logout / change-password ----

func generateJWT(e models.Employee, role models.Role) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"employee_id":   e.ID,
		"employee_name": e.Name,
		"company_id":    e.CompanyID,
		"username":      e.Username,
		"role":          role.Name,
		"role_id":       role.ID,
		"permissions":   rolePermissionCodes(role),
		"exp":           time.Now().Add(time.Hour * 1).Unix(),
	})
	return token.SignedString([]byte(config.JWTSecret))
}

func generateRandomToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func generateRefreshToken(employeeID uint) (string, error) {
	plain, err := generateRandomToken()
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256([]byte(plain))

	rt := models.RefreshToken{
		EmployeeID: employeeID,
		TokenHash:  hex.EncodeToString(hash[:]),
		IssuedAt:   time.Now(),
		ExpiresAt:  time.Now().Add(30 * 24 * time.Hour),
	}
	if err := database.DB.Create(&rt).Error; err != nil {
		return "", err
	}
	return plain, nil
}

func EmployeeLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	recordLogin := func(companyID *uint, employeeID *uint, result string) {
		database.DB.Create(&models.LoginAudit{
			CompanyID:  companyID,
			EmployeeID: employeeID,
			Username:   req.Username,
			Result:     result,
			IP:         c.ClientIP(),
			UserAgent:  c.GetHeader("User-Agent"),
		})
	}

	var e models.Employee
	if err := database.DB.
		Preload("Company").
		Preload("Store").
		Preload("Role").
		Where("username = ?", req.Username).
		First(&e).Error; err != nil {

		recordLogin(nil, nil, "not_found")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if e.Status != "aktif" {
		recordLogin(&e.CompanyID, &e.ID, "disabled")
		c.JSON(http.StatusForbidden, gin.H{"error": "akun nonaktif"})
		return
	}

	if e.LockedUntil != nil && time.Now().Before(*e.LockedUntil) {
		recordLogin(&e.CompanyID, &e.ID, "locked")
		c.JSON(http.StatusForbidden, gin.H{"error": "akun terkunci sementara, coba lagi nanti"})
		return
	}

	cleanPassword := strings.Trim(req.Password, " \r\n\t")
	if err := bcrypt.CompareHashAndPassword([]byte(e.Password), []byte(cleanPassword)); err != nil {
		e.FailedLoginCount++
		if e.FailedLoginCount >= 10 {
			lockUntil := time.Now().Add(15 * time.Minute)
			e.LockedUntil = &lockUntil
			e.FailedLoginCount = 0
		}
		database.DB.Save(&e)
		recordLogin(&e.CompanyID, &e.ID, "invalid_credentials")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials password"})
		return
	}

	var totp models.EmployeeTOTP
	if err := database.DB.Where("employee_id = ? AND enabled = true", e.ID).First(&totp).Error; err == nil {
		recordLogin(&e.CompanyID, &e.ID, "two_factor_required")
		c.JSON(http.StatusOK, gin.H{"requires_2fa": true, "employee_id": e.ID})
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

	now := time.Now()
	e.LastLoginAt = &now
	e.FailedLoginCount = 0
	e.LockedUntil = nil
	database.DB.Save(&e)
	recordLogin(&e.CompanyID, &e.ID, "success")

	e.Password = ""
	c.JSON(http.StatusOK, gin.H{
		"token":         signed,
		"refresh_token": refreshToken,
		"expires_in":    3600,
		"employee":      e,
		"permissions":   rolePermissionCodes(e.Role),
	})
}

type ForgotPasswordRequest struct {
	Username string `json:"username" binding:"required"`
}

// ForgotPassword implements docs/auth.md POST /auth/forgot-password (generates a one-time reset token).
func ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var e models.Employee
	if err := database.DB.Where("username = ?", req.Username).First(&e).Error; err != nil {
		// don't leak whether the account exists
		c.JSON(http.StatusOK, gin.H{"message": "jika akun ditemukan, instruksi reset telah dikirim"})
		return
	}
	plain, err := generateRandomToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create reset token"})
		return
	}
	hash := sha256.Sum256([]byte(plain))
	database.DB.Create(&models.PasswordReset{
		EmployeeID: e.ID,
		TokenHash:  hex.EncodeToString(hash[:]),
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	})
	// NOTE: in production this token is emailed to the user, not returned in the response.
	c.JSON(http.StatusOK, gin.H{"message": "instruksi reset telah dikirim", "reset_token": plain})
}

type ResetPasswordViaTokenRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// ResetPasswordViaToken implements docs/auth.md POST /auth/reset-password
func ResetPasswordViaToken(c *gin.Context) {
	var req ResetPasswordViaTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	hash := sha256.Sum256([]byte(req.Token))
	hashHex := hex.EncodeToString(hash[:])

	var reset models.PasswordReset
	if err := database.DB.Where("token_hash = ? AND used_at IS NULL AND expires_at > ?", hashHex, time.Now()).First(&reset).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token tidak valid atau kedaluwarsa"})
		return
	}
	var e models.Employee
	if err := database.DB.First(&e, reset.EmployeeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Employee not found"})
		return
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}
	e.Password = string(hashed)
	database.DB.Save(&e)
	now := time.Now()
	reset.UsedAt = &now
	database.DB.Save(&reset)
	logAudit(c, e.CompanyID, "password_change", "user", e.ID, e.Name, nil)
	c.JSON(http.StatusOK, gin.H{"message": "password berhasil direset"})
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func RefreshAccessToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hash := sha256.Sum256([]byte(req.RefreshToken))
	hashHex := hex.EncodeToString(hash[:])

	var rt models.RefreshToken
	if err := database.DB.Where("token_hash = ? AND revoked_at IS NULL AND expires_at > ?", hashHex, time.Now()).First(&rt).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired refresh token"})
		return
	}

	var e models.Employee
	if err := database.DB.Preload("Role").First(&e, rt.EmployeeID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "employee not found"})
		return
	}

	// rotate: revoke old, issue new
	now := time.Now()
	rt.RevokedAt = &now
	database.DB.Save(&rt)

	signed, err := generateJWT(e, e.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create token"})
		return
	}
	newRefresh, err := generateRefreshToken(e.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create refresh token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": signed, "refresh_token": newRefresh, "expires_in": 3600})
}

func EmployeeLogout(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	hash := sha256.Sum256([]byte(req.RefreshToken))
	hashHex := hex.EncodeToString(hash[:])

	now := time.Now()
	database.DB.Model(&models.RefreshToken{}).
		Where("token_hash = ? AND revoked_at IS NULL", hashHex).
		Update("revoked_at", now)

	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

func EmployeeChangePassword(c *gin.Context) {
	empIDVal, _ := c.Get("employee_id")
	empID, ok := empIDVal.(float64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var e models.Employee
	if err := database.DB.First(&e, uint(empID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Employee not found"})
		return
	}
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(e.Password), []byte(req.OldPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "password lama salah"})
		return
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}
	e.Password = string(hashed)
	database.DB.Save(&e)
	logAudit(c, e.CompanyID, "password_change", "user", e.ID, e.Name, nil)
	c.JSON(http.StatusOK, gin.H{"message": "password updated"})
}

// EmployeeMe returns the authenticated employee's profile + role + permissions (docs/auth.md GET /api/auth/me).
func EmployeeMe(c *gin.Context) {
	empIDVal, _ := c.Get("employee_id")
	empID, ok := empIDVal.(float64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var e models.Employee
	if err := database.DB.Preload("Company").Preload("Store").Preload("Role").First(&e, uint(empID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Employee not found"})
		return
	}
	e.Password = ""
	c.JSON(http.StatusOK, gin.H{
		"employee":    e,
		"role":        e.Role,
		"permissions": rolePermissionCodes(e.Role),
	})
}

// RefreshUserMetrics implements docs/karyawan.md user_metrics_cache (manual/scheduled trigger).
func RefreshUserMetrics(c *gin.Context) {
	id := c.Param("id")
	var e models.Employee
	if err := database.DB.First(&e, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Employee not found"})
		return
	}
	var storeIDs []uint
	_ = json.Unmarshal([]byte(e.StoreIDs), &storeIDs)
	outletCount := len(storeIDs)
	if e.StoreID != nil {
		outletCount++
	}

	var txCount int64
	database.DB.Model(&models.SalesTransaction{}).
		Where("employee_id = ? AND transaction_date >= ?", e.ID, time.Now().AddDate(0, 0, -30)).
		Count(&txCount)

	cache := models.UserMetricsCache{
		EmployeeID: e.ID, OutletCount: outletCount, TransactionCount30d: txCount,
		LastActiveAt: e.LastLoginAt, UpdatedAt: time.Now(),
	}
	database.DB.Save(&cache)
	c.JSON(http.StatusOK, cache)
}

// GetEmployeeOptions implements docs/karyawan.md GET /users/options
func GetEmployeeOptions(c *gin.Context) {
	type option struct {
		ID       uint   `json:"id"`
		Name     string `json:"name"`
		Username string `json:"username"`
		RoleName string `json:"role_name"`
	}
	var list []option
	q := database.DB.Table("employees").
		Select("employees.id, employees.name, employees.username, roles.name as role_name").
		Joins("LEFT JOIN roles ON roles.id = employees.role_id").
		Where("employees.status = 'aktif'")
	if companyID := c.Query("company_id"); companyID != "" {
		q = q.Where("employees.company_id = ?", companyID)
	}
	if roleID := c.Query("role"); roleID != "" {
		q = q.Where("employees.role_id = ?", roleID)
	}
	if search := c.Query("q"); search != "" {
		q = q.Where("employees.name ILIKE ?", "%"+search+"%")
	}
	q.Find(&list)
	c.JSON(http.StatusOK, list)
}

// GetEmployeesByOutlet implements docs/karyawan.md GET /users/by-outlet/:outletId
func GetEmployeesByOutlet(c *gin.Context) {
	outletID := c.Param("outletId")
	var list []models.Employee
	database.DB.Where("store_id = ? OR store_ids LIKE ?", outletID, "%\""+outletID+"\"%").Find(&list)
	for i := range list {
		list[i].Password = ""
	}
	c.JSON(http.StatusOK, list)
}

type InviteEmployeeRequest struct {
	CompanyID uint   `json:"company_id" binding:"required"`
	Email     string `json:"email" binding:"required,email"`
	RoleID    *uint  `json:"role_id"`
	StoreIDs  []uint `json:"store_ids"`
	InvitedBy uint   `json:"invited_by"`
}

// InviteEmployee implements docs/karyawan.md POST /users/invite
func InviteEmployee(c *gin.Context) {
	var req InviteEmployeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	plain, err := generateRandomToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create invite token"})
		return
	}
	hash := sha256.Sum256([]byte(plain))
	inv := models.EmployeeInvitation{
		CompanyID: req.CompanyID,
		Email:     req.Email,
		RoleID:    req.RoleID,
		StoreIDs:  encodeIDs(req.StoreIDs),
		TokenHash: hex.EncodeToString(hash[:]),
		InvitedBy: req.InvitedBy,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	database.DB.Create(&inv)
	logAudit(c, req.CompanyID, "user_create", "user", inv.ID, req.Email, nil)
	// NOTE: in production the token is emailed; returned here since there is no mail transport configured.
	c.JSON(http.StatusCreated, gin.H{"message": "invitation created", "invite_token": plain})
}

type AcceptInviteRequest struct {
	Token    string `json:"token" binding:"required"`
	Name     string `json:"name" binding:"required"`
	Username string `json:"username" binding:"required,alphanum"`
	Password string `json:"password" binding:"required,min=6"`
}

// AcceptInvite implements docs/karyawan.md POST /users/accept-invite (public)
func AcceptInvite(c *gin.Context) {
	var req AcceptInviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	hash := sha256.Sum256([]byte(req.Token))
	hashHex := hex.EncodeToString(hash[:])

	var inv models.EmployeeInvitation
	if err := database.DB.Where("token_hash = ? AND accepted_at IS NULL AND expires_at > ?", hashHex, time.Now()).First(&inv).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token tidak valid atau kedaluwarsa"})
		return
	}
	if inv.RoleID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "undangan belum memiliki role, hubungi admin"})
		return
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}
	e := models.Employee{
		CompanyID: inv.CompanyID,
		Name:      req.Name,
		Username:  req.Username,
		Email:     inv.Email,
		Password:  string(hashed),
		RoleID:    *inv.RoleID,
		StoreIDs:  inv.StoreIDs,
		Status:    "aktif",
	}
	if err := database.DB.Create(&e).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create employee"})
		return
	}
	now := time.Now()
	inv.AcceptedAt = &now
	database.DB.Save(&inv)
	e.Password = ""
	c.JSON(http.StatusCreated, e)
}

type BulkImportEmployeeItem struct {
	Name     string `json:"name" binding:"required"`
	Username string `json:"username" binding:"required,alphanum"`
	Email    string `json:"email"`
	Password string `json:"password" binding:"required,min=6"`
	RoleID   uint   `json:"role_id" binding:"required"`
}

type BulkImportEmployeesRequest struct {
	CompanyID uint                     `json:"company_id" binding:"required"`
	Employees []BulkImportEmployeeItem `json:"employees" binding:"required,dive,required"`
}

// BulkImportEmployees implements docs/karyawan.md POST /users/bulk-import.
// Simplified to accept a JSON array in the request body instead of multipart CSV.
func BulkImportEmployees(c *gin.Context) {
	var req BulkImportEmployeesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	created := make([]models.Employee, 0, len(req.Employees))
	errorsList := make([]string, 0)
	for _, item := range req.Employees {
		hashed, err := bcrypt.GenerateFromPassword([]byte(item.Password), bcrypt.DefaultCost)
		if err != nil {
			errorsList = append(errorsList, item.Username+": "+err.Error())
			continue
		}
		e := models.Employee{
			CompanyID: req.CompanyID,
			Name:      item.Name,
			Username:  item.Username,
			Email:     item.Email,
			Password:  string(hashed),
			RoleID:    item.RoleID,
			Status:    "aktif",
		}
		if err := database.DB.Create(&e).Error; err != nil {
			errorsList = append(errorsList, item.Username+": "+err.Error())
			continue
		}
		e.Password = ""
		created = append(created, e)
	}
	c.JSON(http.StatusOK, gin.H{"created": created, "errors": errorsList})
}
