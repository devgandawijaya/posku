package models

// ProductImage implements docs/product.md product gallery (URL-only, no file storage backend).
type ProductImage struct {
	ID        uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	ProductID uint   `json:"product_id" gorm:"not null;index"`
	URL       string `json:"url" gorm:"not null"`
	SortOrder int    `json:"sort_order" gorm:"not null;default:0"`
}
