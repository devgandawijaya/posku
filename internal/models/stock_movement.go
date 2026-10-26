package models

import "time"

// StockMovement implements the audit trail from docs/stok.md.
type StockMovement struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	CompanyID uint      `json:"company_id" gorm:"not null;index"`
	ProductID uint      `json:"product_id" gorm:"not null;index"`
	Product   Product   `json:"product" gorm:"foreignKey:ProductID;references:ID;constraint:OnDelete:RESTRICT"`
	StoreID   uint      `json:"store_id" gorm:"not null;index"`
	Store     Store     `json:"store" gorm:"foreignKey:StoreID;references:ID;constraint:OnDelete:CASCADE"`
	Delta     float64   `json:"delta" gorm:"type:decimal(15,3);not null"`
	Reason    string    `json:"reason" gorm:"type:varchar(30);not null"`
	RefType   string    `json:"ref_type"`
	RefID     *uint     `json:"ref_id,omitempty"`
	Note      string    `json:"note" gorm:"type:text"`
	CreatedBy uint      `json:"created_by"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime;index"`
}
