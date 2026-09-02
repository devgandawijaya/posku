package controllers

import (
	"net/http"
	"posku/internal/database"
	"posku/internal/models"

	"github.com/gin-gonic/gin"
)

type CreateProductRequest struct {
	CompanyID   uint    `json:"company_id" binding:"required"`
	CategoryID  *uint   `json:"category_id"`
	SKU         string  `json:"sku"`
	Barcode     string  `json:"barcode"`
	Name        string  `json:"name" binding:"required"`
	Description string  `json:"description"`
	Price       float64 `json:"price" binding:"required,gt=0"`
	Cost        float64 `json:"cost"`
	Unit        string  `json:"unit"`
}

type UpdateProductRequest struct {
	CategoryID  *uint   `json:"category_id"`
	SKU         string  `json:"sku"`
	Barcode     string  `json:"barcode"`
	Name        string  `json:"name" binding:"required"`
	Description string  `json:"description"`
	Price       float64 `json:"price" binding:"required,gt=0"`
	Cost        float64 `json:"cost"`
	Unit        string  `json:"unit"`
}

func CreateProduct(c *gin.Context) {
	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var company models.Company
	if err := database.DB.First(&company, req.CompanyID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "company not found"})
		return
	}
	if ok, msg := checkPlanLimit(req.CompanyID, "products"); !ok {
		c.JSON(http.StatusPaymentRequired, gin.H{"error": msg})
		return
	}
	if req.CategoryID != nil {
		var cat models.Category
		if err := database.DB.First(&cat, *req.CategoryID).Error; err != nil || cat.CompanyID != req.CompanyID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "category not found or not owned by company"})
			return
		}
	}

	p := models.Product{
		CompanyID:   req.CompanyID,
		CategoryID:  req.CategoryID,
		SKU:         req.SKU,
		Barcode:     req.Barcode,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Cost:        req.Cost,
		Unit:        req.Unit,
	}

	if err := database.DB.Create(&p).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create product"})
		return
	}

	logAudit(c, p.CompanyID, "product_create", "product", p.ID, p.Name, nil)
	c.JSON(http.StatusCreated, p)
}

func GetProducts(c *gin.Context) {
	var list []models.Product
	q := database.DB.Preload("Category")
	if companyID := c.Query("company_id"); companyID != "" {
		q = q.Where("company_id = ?", companyID)
	}
	if categoryID := c.Query("category_id"); categoryID != "" {
		q = q.Where("category_id = ?", categoryID)
	}
	if search := c.Query("q"); search != "" {
		q = q.Where("name ILIKE ? OR sku ILIKE ? OR barcode ILIKE ?", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}
	if err := q.Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch products"})
		return
	}
	c.JSON(http.StatusOK, list)
}

func GetProduct(c *gin.Context) {
	id := c.Param("id")
	var p models.Product
	if err := database.DB.Preload("Category").First(&p, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}
	c.JSON(http.StatusOK, p)
}

func UpdateProduct(c *gin.Context) {
	id := c.Param("id")
	var p models.Product
	if err := database.DB.First(&p, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}
	var input UpdateProductRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if input.CategoryID != nil {
		var cat models.Category
		if err := database.DB.First(&cat, *input.CategoryID).Error; err != nil || cat.CompanyID != p.CompanyID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "category not found or not owned by company"})
			return
		}
	}

	p.CategoryID = input.CategoryID
	p.SKU = input.SKU
	p.Barcode = input.Barcode
	p.Name = input.Name
	p.Description = input.Description
	if input.Price != p.Price {
		database.DB.Create(&models.ProductPriceHistory{ProductID: p.ID, OldPrice: p.Price, NewPrice: input.Price})
	}
	p.Price = input.Price
	p.Cost = input.Cost
	p.Unit = input.Unit

	if err := database.DB.Save(&p).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update product"})
		return
	}

	logAudit(c, p.CompanyID, "product_update", "product", p.ID, p.Name, nil)
	c.JSON(http.StatusOK, p)
}

func DeleteProduct(c *gin.Context) {
	id := c.Param("id")
	var p models.Product
	if err := database.DB.First(&p, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	if err := database.DB.Delete(&p).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete product"})
		return
	}

	logAudit(c, p.CompanyID, "product_delete", "product", p.ID, p.Name, nil)
	c.JSON(http.StatusOK, gin.H{"message": "Product deleted"})
}

type BulkProductIDsRequest struct {
	ProductIDs []uint `json:"product_ids" binding:"required"`
}

// BulkActivateProducts implements docs/stok.md POST /stocks/bulk-activate
func BulkActivateProducts(c *gin.Context) {
	var req BulkProductIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	database.DB.Model(&models.Product{}).Where("id IN ?", req.ProductIDs).Update("is_active", true)
	c.JSON(http.StatusOK, gin.H{"message": "products activated"})
}

// BulkDeactivateProducts implements docs/stok.md POST /stocks/bulk-deactivate
func BulkDeactivateProducts(c *gin.Context) {
	var req BulkProductIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	database.DB.Model(&models.Product{}).Where("id IN ?", req.ProductIDs).Update("is_active", false)
	c.JSON(http.StatusOK, gin.H{"message": "products deactivated"})
}

// BulkDeleteProducts implements docs/stok.md POST /stocks/bulk-delete
func BulkDeleteProducts(c *gin.Context) {
	var req BulkProductIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	database.DB.Where("id IN ?", req.ProductIDs).Delete(&models.Product{})
	c.JSON(http.StatusOK, gin.H{"message": "products deleted"})
}

type AddProductBarcodeRequest struct {
	Barcode string `json:"barcode" binding:"required"`
}

// AddProductBarcode implements docs/product.md multi-barcode support.
func AddProductBarcode(c *gin.Context) {
	id := c.Param("id")
	var req AddProductBarcodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	bc := models.ProductBarcode{ProductID: parseUintParam(id), Barcode: req.Barcode}
	if err := database.DB.Create(&bc).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add barcode"})
		return
	}
	c.JSON(http.StatusCreated, bc)
}

func GetProductBarcodes(c *gin.Context) {
	id := c.Param("id")
	var list []models.ProductBarcode
	database.DB.Where("product_id = ?", id).Find(&list)
	c.JSON(http.StatusOK, list)
}

func GetProductPriceHistory(c *gin.Context) {
	id := c.Param("id")
	var list []models.ProductPriceHistory
	database.DB.Where("product_id = ?", id).Order("changed_at desc").Find(&list)
	c.JSON(http.StatusOK, list)
}

type AddProductImageRequest struct {
	URL       string `json:"url" binding:"required"`
	SortOrder int    `json:"sort_order"`
}

// AddProductImage implements docs/product.md product image gallery (URL-only).
func AddProductImage(c *gin.Context) {
	id := c.Param("id")
	var req AddProductImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	img := models.ProductImage{ProductID: parseUintParam(id), URL: req.URL, SortOrder: req.SortOrder}
	if err := database.DB.Create(&img).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add image"})
		return
	}
	c.JSON(http.StatusCreated, img)
}

func GetProductImages(c *gin.Context) {
	id := c.Param("id")
	var list []models.ProductImage
	database.DB.Where("product_id = ?", id).Order("sort_order").Find(&list)
	c.JSON(http.StatusOK, list)
}

func DeleteProductImage(c *gin.Context) {
	id := c.Param("imageId")
	if err := database.DB.Delete(&models.ProductImage{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete image"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Image deleted"})
}
