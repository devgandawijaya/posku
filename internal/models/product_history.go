package models

import "time"

// ProductBarcode implements docs/product.md multi-barcode support.
type ProductBarcode struct {
	ID        uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	ProductID uint   `json:"product_id" gorm:"not null;index"`
	Barcode   string `json:"barcode" gorm:"not null;index"`
}

// ProductPriceHistory implements docs/product.md price change history.
type ProductPriceHistory struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	ProductID uint      `json:"product_id" gorm:"not null;index"`
	OldPrice  float64   `json:"old_price" gorm:"type:decimal(15,2);not null"`
	NewPrice  float64   `json:"new_price" gorm:"type:decimal(15,2);not null"`
	ChangedBy uint      `json:"changed_by"`
	ChangedAt time.Time `json:"changed_at" gorm:"autoCreateTime"`
}
