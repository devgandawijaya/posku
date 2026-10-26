package models

import "time"

// StockAdjustment implements docs/stok.md stock opname draft->submit->approve workflow.
type StockAdjustment struct {
	ID         uint                  `json:"id" gorm:"primaryKey;autoIncrement"`
	CompanyID  uint                  `json:"company_id" gorm:"not null;index"`
	StoreID    uint                  `json:"store_id" gorm:"not null"`
	Status     string                `json:"status" gorm:"type:varchar(20);not null;default:'draft'"` // draft|submitted|approved|rejected
	Note       string                `json:"note" gorm:"type:text"`
	CreatedBy  uint                  `json:"created_by"`
	ApprovedBy *uint                 `json:"approved_by,omitempty"`
	Items      []StockAdjustmentItem `json:"items" gorm:"foreignKey:AdjustmentID"`
	CreatedAt  time.Time             `json:"created_at"`
	ApprovedAt *time.Time            `json:"approved_at,omitempty"`
}

type StockAdjustmentItem struct {
	ID           uint    `json:"id" gorm:"primaryKey;autoIncrement"`
	AdjustmentID uint    `json:"adjustment_id" gorm:"not null;index"`
	ProductID    uint    `json:"product_id" gorm:"not null"`
	SystemQty    float64 `json:"system_qty" gorm:"type:decimal(15,3);not null"`
	ActualQty    float64 `json:"actual_qty" gorm:"type:decimal(15,3);not null"`
	Note         string  `json:"note"`
}
