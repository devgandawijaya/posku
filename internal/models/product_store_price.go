package models

import "time"

// ProductStorePrice enables custom price and cost per store/outlet.
type ProductStorePrice struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	CompanyID uint      `json:"company_id" gorm:"not null;index"`
	ProductID uint      `json:"product_id" gorm:"not null;uniqueIndex:idx_product_store"`
	Product   *Product  `json:"product,omitempty" gorm:"foreignKey:ProductID;references:ID;constraint:OnDelete:CASCADE"`
	StoreID   uint      `json:"store_id" gorm:"not null;uniqueIndex:idx_product_store"`
	Store     *Store    `json:"store,omitempty" gorm:"foreignKey:StoreID;references:ID;constraint:OnDelete:CASCADE"`
	Price     float64   `json:"price" gorm:"type:decimal(10,2);not null"`
	Cost      float64   `json:"cost" gorm:"type:decimal(10,2);not null;default:0"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

