package controllers

import (
	"net/http"
	"posku/internal/database"
	"posku/internal/models"

	"github.com/gin-gonic/gin"
)

type CreateProductRequest struct {
	CompanyID   uint    `json:"company_id"`
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
	IsActive    *bool   `json:"is_active"`
}

func CreateProduct(c *gin.Context) {
	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, err.Error())
		return
	}

	companyID := getAuthCompanyID(c)
	if companyID == 0 {
		companyID = req.CompanyID
	}
	if companyID == 0 {
		RespondError(c, http.StatusBadRequest, "company_id is required")
		return
	}

	var company models.Company
	if err := database.DB.First(&company, companyID).Error; err != nil {
		RespondError(c, http.StatusBadRequest, "company not found")
		return
	}
	if ok, msg := checkPlanLimit(companyID, "products"); !ok {
		RespondError(c, http.StatusPaymentRequired, msg)
		return
	}
	if req.CategoryID != nil {
		var cat models.Category
		if err := database.DB.First(&cat, *req.CategoryID).Error; err != nil || cat.CompanyID != companyID {
			RespondError(c, http.StatusBadRequest, "category not found or not owned by company")
			return
		}
	}

	p := models.Product{
		CompanyID:   companyID,
		CategoryID:  req.CategoryID,
		SKU:         req.SKU,
		Barcode:     req.Barcode,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Cost:        req.Cost,
		Unit:        req.Unit,
		IsActive:    true,
	}

	if err := database.DB.Create(&p).Error; err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to create product")
		return
	}

	logAudit(c, p.CompanyID, "product_create", "product", p.ID, p.Name, nil)
	RespondCreated(c, p, "Product created successfully")
}

func GetProducts(c *gin.Context) {
	companyID := getAuthCompanyID(c)
	q := database.DB.Model(&models.Product{}).Preload("Category")

	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	} else if qCID := c.Query("company_id"); qCID != "" {
		q = q.Where("company_id = ?", qCID)
	}

	if categoryID := c.Query("category_id"); categoryID != "" {
		q = q.Where("category_id = ?", categoryID)
	}
	if active := c.Query("is_active"); active != "" {
		if active == "true" {
			q = q.Where("is_active = true")
		} else if active == "false" {
			q = q.Where("is_active = false")
		}
	}
	if search := c.Query("q"); search != "" {
		q = q.Where("name ILIKE ? OR sku ILIKE ? OR barcode ILIKE ?", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	if c.Query("page") != "" || c.Query("per_page") != "" {
		var total int64
		q.Count(&total)
		page, perPage, offset := GetPagination(c)
		var list []models.Product
		if err := q.Order("id asc").Limit(perPage).Offset(offset).Find(&list).Error; err != nil {
			RespondError(c, http.StatusInternalServerError, "Failed to fetch products")
			return
		}
		RespondPaginated(c, list, page, perPage, total)
		return
	}

	var list []models.Product
	if err := q.Order("id asc").Find(&list).Error; err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to fetch products")
		return
	}
	c.JSON(http.StatusOK, list)
}

func GetProduct(c *gin.Context) {
	id := c.Param("id")
	companyID := getAuthCompanyID(c)

	var p models.Product
	q := database.DB.Preload("Category")
	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	}
	if err := q.First(&p, id).Error; err != nil {
		RespondError(c, http.StatusNotFound, "Product not found")
		return
	}
	RespondSuccess(c, p)
}

func UpdateProduct(c *gin.Context) {
	id := c.Param("id")
	companyID := getAuthCompanyID(c)

	var p models.Product
	q := database.DB
	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	}
	if err := q.First(&p, id).Error; err != nil {
		RespondError(c, http.StatusNotFound, "Product not found")
		return
	}
	var input UpdateProductRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if input.CategoryID != nil {
		var cat models.Category
		if err := database.DB.First(&cat, *input.CategoryID).Error; err != nil || cat.CompanyID != p.CompanyID {
			RespondError(c, http.StatusBadRequest, "category not found or not owned by company")
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
	if input.IsActive != nil {
		p.IsActive = *input.IsActive
	}

	if err := database.DB.Save(&p).Error; err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to update product")
		return
	}

	logAudit(c, p.CompanyID, "product_update", "product", p.ID, p.Name, nil)
	RespondSuccess(c, p, "Product updated successfully")
}

func DeleteProduct(c *gin.Context) {
	id := c.Param("id")
	companyID := getAuthCompanyID(c)

	var p models.Product
	q := database.DB
	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	}
	if err := q.First(&p, id).Error; err != nil {
		RespondError(c, http.StatusNotFound, "Product not found")
		return
	}

	if err := database.DB.Delete(&p).Error; err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to delete product")
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
		RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	database.DB.Model(&models.Product{}).Where("id IN ?", req.ProductIDs).Update("is_active", true)
	c.JSON(http.StatusOK, gin.H{"message": "products activated"})
}

// BulkDeactivateProducts implements docs/stok.md POST /stocks/bulk-deactivate
func BulkDeactivateProducts(c *gin.Context) {
	var req BulkProductIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	database.DB.Model(&models.Product{}).Where("id IN ?", req.ProductIDs).Update("is_active", false)
	c.JSON(http.StatusOK, gin.H{"message": "products deactivated"})
}

// BulkDeleteProducts implements docs/stok.md POST /stocks/bulk-delete
func BulkDeleteProducts(c *gin.Context) {
	var req BulkProductIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	database.DB.Where("id IN ?", req.ProductIDs).Delete(&models.Product{})
	c.JSON(http.StatusOK, gin.H{"message": "products deleted"})
}

type AddProductBarcodeRequest struct {
	Barcode string `json:"barcode" binding:"required"`
}

func AddProductBarcode(c *gin.Context) {
	id := c.Param("id")
	var req AddProductBarcodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	bc := models.ProductBarcode{ProductID: parseUintParam(id), Barcode: req.Barcode}
	if err := database.DB.Create(&bc).Error; err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to add barcode")
		return
	}
	RespondCreated(c, bc)
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

func AddProductImage(c *gin.Context) {
	id := c.Param("id")
	var req AddProductImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	img := models.ProductImage{ProductID: parseUintParam(id), URL: req.URL, SortOrder: req.SortOrder}
	if err := database.DB.Create(&img).Error; err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to add image")
		return
	}
	RespondCreated(c, img)
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
		RespondError(c, http.StatusInternalServerError, "Failed to delete image")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Image deleted"})
}

// ---- Multi-Store Pricing (ProductStorePrice) ----

type SetProductStorePriceRequest struct {
	StoreID uint    `json:"store_id" binding:"required"`
	Price   float64 `json:"price" binding:"required,gt=0"`
	Cost    float64 `json:"cost"`
}

// SetProductStorePrice sets custom selling price and cost for a specific store/outlet.
func SetProductStorePrice(c *gin.Context) {
	productID := parseUintParam(c.Param("id"))
	companyID := getAuthCompanyID(c)

	var p models.Product
	if err := database.DB.First(&p, productID).Error; err != nil {
		RespondError(c, http.StatusNotFound, "Product not found")
		return
	}
	if companyID > 0 && p.CompanyID != companyID {
		RespondError(c, http.StatusForbidden, "Unauthorized product access")
		return
	}

	var req SetProductStorePriceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, err.Error())
		return
	}

	// Validate store
	var store models.Store
	if err := database.DB.Where("id = ? AND company_id = ?", req.StoreID, p.CompanyID).First(&store).Error; err != nil {
		RespondError(c, http.StatusBadRequest, "store not found or does not belong to company")
		return
	}

	var sp models.ProductStorePrice
	err := database.DB.Where("product_id = ? AND store_id = ?", productID, req.StoreID).First(&sp).Error
	if err != nil {
		sp = models.ProductStorePrice{
			CompanyID: p.CompanyID,
			ProductID: productID,
			StoreID:   req.StoreID,
			Price:     req.Price,
			Cost:      req.Cost,
		}
		database.DB.Create(&sp)
	} else {
		sp.Price = req.Price
		sp.Cost = req.Cost
		database.DB.Save(&sp)
	}

	RespondSuccess(c, sp, "Store price set successfully")
}

// GetProductStorePrices returns all custom store prices for a product.
func GetProductStorePrices(c *gin.Context) {
	productID := parseUintParam(c.Param("id"))
	var list []models.ProductStorePrice
	database.DB.Preload("Store").Where("product_id = ?", productID).Find(&list)
	c.JSON(http.StatusOK, list)
}

// DeleteProductStorePrice removes store-specific price override, reverting to base product price.
func DeleteProductStorePrice(c *gin.Context) {
	productID := parseUintParam(c.Param("id"))
	storeID := parseUintParam(c.Param("storeId"))

	database.DB.Where("product_id = ? AND store_id = ?", productID, storeID).Delete(&models.ProductStorePrice{})
	c.JSON(http.StatusOK, gin.H{"message": "Store price override removed, reverted to default price"})
}
