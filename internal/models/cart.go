package models

import "time"

// Cart implements docs/kasir.md persisted cart (hold order support).
type Cart struct {
	ID              uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	CompanyID       uint       `json:"company_id" gorm:"not null;index"`
	ShiftID         uint       `json:"shift_id" gorm:"not null"`
	EmployeeID      uint       `json:"employee_id" gorm:"not null"`
	CustomerID      *uint      `json:"customer_id,omitempty"`
	StoreID         uint       `json:"store_id" gorm:"not null"`
	Status          string     `json:"status" gorm:"type:varchar(20);not null;default:'active'"` // active|held|converted|abandoned
	Subtotal        float64    `json:"subtotal" gorm:"type:decimal(15,2);not null;default:0"`
	Discount        float64    `json:"discount" gorm:"type:decimal(15,2);not null;default:0"`
	Tax             float64    `json:"tax" gorm:"type:decimal(15,2);not null;default:0"`
	Total           float64    `json:"total" gorm:"type:decimal(15,2);not null;default:0"`
	VoucherCode     string     `json:"voucher_code"`
	VoucherDiscount float64    `json:"voucher_discount" gorm:"type:decimal(15,2);not null;default:0"`
	PaymentMethod   string     `json:"payment_method"`
	Items           []CartItem `json:"items" gorm:"foreignKey:CartID"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type CartItem struct {
	ID           uint    `json:"id" gorm:"primaryKey;autoIncrement"`
	CartID       uint    `json:"cart_id" gorm:"not null;index"`
	ProductID    uint    `json:"product_id" gorm:"not null"`
	SKUSnapshot  string  `json:"sku_snapshot"`
	NameSnapshot string  `json:"name_snapshot"`
	Price        float64 `json:"price" gorm:"type:decimal(15,2);not null"`
	Qty          float64 `json:"qty" gorm:"type:decimal(15,3);not null"`
	DiscountPct  float64 `json:"discount_pct" gorm:"type:decimal(5,2);not null;default:0"`
	Note         string  `json:"note"`
}
