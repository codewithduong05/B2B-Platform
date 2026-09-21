package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/atlas-platform/backend/internal/database"
	"github.com/atlas-platform/backend/internal/database/queries/catalog"
	"github.com/atlas-platform/backend/internal/modules/catalog/repository"
	"github.com/atlas-platform/backend/internal/modules/catalog/schema"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrProductNotFound      = errors.New("product not found")
	ErrCategoryNotFound     = errors.New("category not found")
	ErrBrandNotFound        = errors.New("brand not found")
	ErrUnitNotFound         = errors.New("unit not found")
	ErrSupplierNotFound     = errors.New("supplier not found")
	ErrInvalidSlug          = errors.New("slug already exists")
	ErrInvalidCategory      = errors.New("invalid category")
	ErrInvalidBrand         = errors.New("invalid brand")
	ErrInvalidUnit          = errors.New("invalid unit")
	ErrInvalidSupplier      = errors.New("invalid supplier")
	ErrInvalidHandlingClass = errors.New("invalid handling class")
	ErrProductNotPublished  = errors.New("product is not published")
)

type ProductService struct {
	productRepo  *repository.ProductRepository
	categoryRepo *repository.CategoryRepository
	brandRepo    *repository.BrandRepository
	unitRepo     *repository.UnitRepository
	supplierRepo *repository.SupplierRepository
	facetRepo    *repository.FacetRepository
}

func NewProductService() *ProductService {
	return &ProductService{}
}

func (s *ProductService) SetDependencies(
	productRepo *repository.ProductRepository,
	categoryRepo *repository.CategoryRepository,
	brandRepo *repository.BrandRepository,
	unitRepo *repository.UnitRepository,
	supplierRepo *repository.SupplierRepository,
	facetRepo *repository.FacetRepository,
) {
	s.productRepo = productRepo
	s.categoryRepo = categoryRepo
	s.brandRepo = brandRepo
	s.unitRepo = unitRepo
	s.supplierRepo = supplierRepo
	s.facetRepo = facetRepo
}

func (s *ProductService) ListProducts(ctx context.Context, req schema.ProductListRequest) (*schema.ProductListResponse, error) {
	limit := int32(req.PageSize)
	offset := int32((req.Page - 1) * req.PageSize)

	var categoryID *int64
	var brandID *int64
	var supplierID *int64

	if req.Category != nil {
		cat, err := s.categoryRepo.GetCategoryByCode(ctx, *req.Category)
		if err == nil {
			categoryID = &cat.ID
		}
	}
	if req.Brand != nil {
		brand, err := s.brandRepo.GetBrandByCode(ctx, *req.Brand)
		if err == nil {
			brandID = &brand.ID
		}
	}
	if req.Supplier != nil {
		supplier, err := s.supplierRepo.GetSupplierByCode(ctx, *req.Supplier)
		if err == nil {
			supplierID = &supplier.ID
		}
	}

	filters := repository.ProductListFilters{
		CategoryID:    categoryID,
		BrandID:       brandID,
		HandlingClass: (*catalog.CatalogHandlingClassType)(req.HandlingClass),
		SupplierID:    supplierID,
		Query:         req.Query,
		Sort:          req.Sort,
	}

	products, err := s.productRepo.ListProducts(ctx, limit, offset, filters)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}

	total, err := s.productRepo.CountProducts(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("count products: %w", err)
	}

	var resp []schema.ProductSummary
	for _, p := range products {
		resp = append(resp, s.toProductSummary(p))
	}

	facets, err := s.facetRepo.GetFilteredFacets(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("get facets: %w", err)
	}

	return &schema.ProductListResponse{
		Items:    resp,
		Page:     req.Page,
		PageSize: req.PageSize,
		Total:    total,
		HasNext:  int64(req.Page*req.PageSize) < total,
		Facets:   s.toFacetsFromFiltered(facets),
	}, nil
}

func (s *ProductService) GetProduct(ctx context.Context, slug string) (*schema.ProductDetail, error) {
	product, err := s.productRepo.GetProductBySlug(ctx, slug)
	if err != nil {
		return nil, ErrProductNotFound
	}

	if product.Status != catalog.CatalogProductStatusPublished || !product.IsActive {
		return nil, ErrProductNotPublished
	}

	detail := s.toProductDetailFromSlugRow(product)

	media, err := s.productRepo.ListProductMedia(ctx, product.ID)
	if err == nil {
		detail.Media = make([]schema.ProductMediaSummary, 0, len(media))
		for _, m := range media {
			detail.Media = append(detail.Media, schema.ProductMediaSummary{
				Code:      m.Code,
				URL:       m.Url,
				AltText:   m.AltText.String,
				MediaType: m.MediaType,
				SortOrder: int(m.SortOrder),
				IsPrimary: m.IsPrimary,
			})
		}
	}

	attrs, err := s.productRepo.ListProductAttributes(ctx, product.ID)
	if err == nil {
		detail.Attributes = make([]schema.ProductAttributeSummary, 0, len(attrs))
		for _, a := range attrs {
			valName := a.AttributeValueName.String
			var boolVal *bool
			if valName == "" {
				if a.TextValue.Valid {
					valName = a.TextValue.String
				} else if a.NumberValue.Valid {
					f, _ := a.NumberValue.Float64Value()
					valName = fmt.Sprintf("%g", f.Float64)
				} else if a.BooleanValue.Valid {
					b := a.BooleanValue.Bool
					boolVal = &b
					if b {
						valName = "Yes"
					} else {
						valName = "No"
					}
				}
			}
			detail.Attributes = append(detail.Attributes, schema.ProductAttributeSummary{
				AttributeID:   fmt.Sprintf("%d", a.AttributeID),
				AttributeName: a.AttributeName,
				AttributeType: a.AttributeType,
				ValueName:     valName,
				TextValue:     a.TextValue.String,
				BooleanValue:  boolVal,
			})
		}
	}

	units, err := s.productRepo.ListProductUnits(ctx, product.ID)
	if err == nil {
		detail.Units = make([]schema.ProductUnitSummary, 0, len(units))
		for _, u := range units {
			var convFactor float64
			if u.ConversionFactor.Valid {
				f, _ := u.ConversionFactor.Float64Value()
				convFactor = f.Float64
			}
			detail.Units = append(detail.Units, schema.ProductUnitSummary{
				Code:             u.Code,
				UnitCode:         u.Code,
				UnitName:         u.Name,
				UnitSymbol:       u.Symbol,
				ConversionFactor: convFactor,
				IsDefault:        u.IsDefault,
			})
		}
	}

	return detail, nil
}

func (s *ProductService) toProductDetailFromSlugRow(p catalog.GetProductBySlugRow) *schema.ProductDetail {
	detail := &schema.ProductDetail{
		ProductSummary: schema.ProductSummary{
			ID:               p.ID,
			Code:             p.Code,
			Slug:             p.Slug,
			Name:             p.Name,
			ShortDescription: p.ShortDescription.String,
			CategoryCode:     p.CategorySlug.String,
			BrandCode:        p.BrandSlug.String,
			HandlingClass:    string(p.HandlingClass),
			BaseUnitCode:     p.BaseUnitSymbol.String,
			SupplierCode:     p.SupplierSlug.String,
			Status:           string(p.Status),
			IsActive:         p.IsActive,
			IsFeatured:       p.IsFeatured,
			BasePriceMinor:   &p.BasePriceMinor.Int64,
			Currency:         p.Currency,
			TrackInventory:   p.TrackInventory,
			CreatedAt:        p.CreatedAt,
			UpdatedAt:        p.UpdatedAt,
			PublishedAt:      &p.PublishedAt.Time,
		},
		Description:   p.Description.String,
		HandlingClass: string(p.HandlingClass),
	}

	if p.WeightGrams.Valid {
		v := int(p.WeightGrams.Int32)
		detail.WeightGrams = &v
	}
	if p.LengthMm.Valid && p.WidthMm.Valid && p.HeightMm.Valid {
		l, w, h := int(p.LengthMm.Int32), int(p.WidthMm.Int32), int(p.HeightMm.Int32)
		detail.LengthMM = &l
		detail.WidthMM = &w
		detail.HeightMM = &h
	}
	if p.Gtin.Valid {
		detail.GTIN = p.Gtin.String
	}
	if p.Sku.Valid {
		detail.SKU = p.Sku.String
	}

	detail.Category = schema.CategorySummary{
		Code: p.CategorySlug.String,
		Name: p.CategoryName.String,
		Slug: p.CategorySlug.String,
	}
	detail.Brand = schema.BrandSummary{
		Code: p.BrandSlug.String,
		Name: p.BrandName.String,
		Slug: p.BrandSlug.String,
	}
	detail.BaseUnit = schema.UnitSummary{
		Code:   p.BaseUnitName.String,
		Name:   p.BaseUnitName.String,
		Symbol: p.BaseUnitSymbol.String,
	}
	detail.Supplier = schema.SupplierSummary{
		Code: p.SupplierSlug.String,
		Name: p.SupplierName.String,
		Slug: p.SupplierSlug.String,
	}

	return detail
}

func (s *ProductService) GetProductByCode(ctx context.Context, code string) (*schema.ProductDetail, error) {
	product, err := s.productRepo.GetProductByCode(ctx, code)
	if err != nil {
		return nil, ErrProductNotFound
	}
	// Get the full product by ID to get the GetProductByIDRow type
	fullProduct, err := s.productRepo.GetProductByID(ctx, product.ID)
	if err != nil {
		return nil, ErrProductNotFound
	}
	return s.toProductDetailFromProduct(ctx, fullProduct), nil
}

func (s *ProductService) toProductDetailFromProduct(ctx context.Context, p catalog.GetProductByIDRow) *schema.ProductDetail {
	categoryCode := ""
	if p.CategoryID != 0 {
		category, err := s.categoryRepo.GetCategoryByID(ctx, p.CategoryID)
		if err == nil {
			categoryCode = category.Code
		}
	}

	brandCode := ""
	if p.BrandID.Valid {
		brand, err := s.brandRepo.GetBrandByID(ctx, p.BrandID.Int64)
		if err == nil {
			brandCode = brand.Code
		}
	}

	supplierCode := ""
	if p.SupplierID.Valid {
		supplier, err := s.supplierRepo.GetSupplierByID(ctx, p.SupplierID.Int64)
		if err == nil {
			supplierCode = supplier.Code
		}
	}

	baseUnitCode := ""
	if p.BaseUnitID != 0 {
		unit, err := s.unitRepo.GetUnitByID(ctx, p.BaseUnitID)
		if err == nil {
			baseUnitCode = unit.Code
		}
	}

	return &schema.ProductDetail{
		ProductSummary: schema.ProductSummary{
			ID:               p.ID,
			Code:             p.Code,
			Slug:             p.Slug,
			Name:             p.Name,
			ShortDescription: p.ShortDescription.String,
			CategoryCode:     categoryCode,
			BrandCode:        brandCode,
			HandlingClass:    string(p.HandlingClass),
			BaseUnitCode:     baseUnitCode,
			SupplierCode:     supplierCode,
			Status:           string(p.Status),
			IsActive:         p.IsActive,
			IsFeatured:       p.IsFeatured,
			BasePriceMinor:   &p.BasePriceMinor.Int64,
			Currency:         p.Currency,
			TrackInventory:   p.TrackInventory,
			CreatedAt:        p.CreatedAt,
			UpdatedAt:        p.UpdatedAt,
			PublishedAt:      &p.PublishedAt.Time,
		},
	}
}
func (s *ProductService) CreateProduct(ctx context.Context, req schema.CreateProductRequest) (*schema.ProductDetail, error) {
	// Validate category
	if _, err := s.categoryRepo.GetCategoryByID(ctx, req.CategoryID); err != nil {
		return nil, ErrInvalidCategory
	}

	// Validate brand if provided
	if req.BrandID != nil {
		if _, err := s.brandRepo.GetBrandByID(ctx, *req.BrandID); err != nil {
			return nil, ErrInvalidBrand
		}
	}

	// Validate unit
	if _, err := s.unitRepo.GetUnitByID(ctx, req.BaseUnitID); err != nil {
		return nil, ErrInvalidUnit
	}

	// Validate supplier if provided
	if req.SupplierID != nil {
		if _, err := s.supplierRepo.GetSupplierByID(ctx, *req.SupplierID); err != nil {
			return nil, ErrInvalidSupplier
		}
	}

	// Validate handling class
	if !s.isValidHandlingClass(req.HandlingClass) {
		return nil, ErrInvalidHandlingClass
	}

	// Check slug uniqueness
	existing, err := s.productRepo.GetProductBySlug(ctx, req.Slug)
	if err == nil && existing.ID != 0 {
		return nil, ErrInvalidSlug
	}

	// Create product
	params := catalog.CreateProductParams{
		Code:             req.Code,
		Slug:             req.Slug,
		Name:             req.Name,
		Description:      pgtype.Text{String: req.Description, Valid: req.Description != ""},
		ShortDescription: pgtype.Text{String: req.ShortDescription, Valid: req.ShortDescription != ""},
		CategoryID:       req.CategoryID,
		BaseUnitID:       req.BaseUnitID,
		Status:           catalog.CatalogProductStatusDraft,
		IsActive:         true,
		IsFeatured:       false,
		BasePriceMinor:   pgtype.Int8{Int64: 0, Valid: req.BasePriceMinor != nil},
		Currency:         req.Currency,
		TrackInventory:   req.TrackInventory,
		WeightGrams:      pgtype.Int4{Int32: int32(*req.WeightGrams), Valid: req.WeightGrams != nil},
		LengthMm:         pgtype.Int4{Int32: int32(*req.LengthMM), Valid: req.LengthMM != nil},
		WidthMm:          pgtype.Int4{Int32: int32(*req.WidthMM), Valid: req.WidthMM != nil},
		HeightMm:         pgtype.Int4{Int32: int32(*req.HeightMM), Valid: req.HeightMM != nil},
		Gtin:             pgtype.Text{String: req.GTIN, Valid: req.GTIN != ""},
		Sku:              pgtype.Text{String: req.SKU, Valid: req.SKU != ""},
		SeoTitle:         pgtype.Text{String: req.SEOTitle, Valid: req.SEOTitle != ""},
		SeoDescription:   pgtype.Text{String: req.SEODescription, Valid: req.SEODescription != ""},
		SeoKeywords:      pgtype.Text{String: req.SEOKeywords, Valid: req.SEOKeywords != ""},
	}

	if req.BrandID != nil {
		params.BrandID = pgtype.Int8{Int64: *req.BrandID, Valid: true}
	}
	if req.SupplierID != nil {
		params.SupplierID = pgtype.Int8{Int64: *req.SupplierID, Valid: true}
	}
	params.HandlingClass = catalog.CatalogHandlingClassType(req.HandlingClass)

	productRow, err := s.productRepo.CreateProduct(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("create product: %w", err)
	}

	product, err := s.productRepo.GetProductByID(ctx, productRow.ID)
	if err != nil {
		return nil, fmt.Errorf("get created product: %w", err)
	}

	return s.toProductDetailFromProduct(ctx, product), nil
}

func (s *ProductService) UpdateProduct(ctx context.Context, productID int64, req schema.UpdateProductRequest) (*schema.ProductDetail, error) {
	updates := make(map[string]interface{})
	if req.Slug != nil {
		updates["slug"] = *req.Slug
	}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.ShortDescription != nil {
		updates["short_description"] = *req.ShortDescription
	}
	if req.CategoryID != nil {
		updates["category_id"] = *req.CategoryID
	}
	if req.BrandID != nil {
		updates["brand_id"] = *req.BrandID
	}
	if req.HandlingClass != nil {
		updates["handling_class"] = *req.HandlingClass
	}
	if req.BaseUnitID != nil {
		updates["base_unit_id"] = *req.BaseUnitID
	}
	if req.SupplierID != nil {
		updates["supplier_id"] = *req.SupplierID
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if req.IsFeatured != nil {
		updates["is_featured"] = *req.IsFeatured
	}
	if req.SortOrder != nil {
		updates["sort_order"] = *req.SortOrder
	}
	if req.BasePriceMinor != nil {
		updates["base_price_minor"] = *req.BasePriceMinor
	}
	if req.Currency != nil {
		updates["currency"] = *req.Currency
	}
	if req.TrackInventory != nil {
		updates["track_inventory"] = *req.TrackInventory
	}
	if req.WeightGrams != nil {
		updates["weight_grams"] = *req.WeightGrams
	}
	if req.LengthMM != nil {
		updates["length_mm"] = *req.LengthMM
	}
	if req.WidthMM != nil {
		updates["width_mm"] = *req.WidthMM
	}
	if req.HeightMM != nil {
		updates["height_mm"] = *req.HeightMM
	}
	if req.GTIN != nil {
		updates["gtin"] = *req.GTIN
	}
	if req.SKU != nil {
		updates["sku"] = *req.SKU
	}
	if req.SEOTitle != nil {
		updates["seo_title"] = *req.SEOTitle
	}
	if req.SEODescription != nil {
		updates["seo_description"] = *req.SEODescription
	}
	if req.SEOKeywords != nil {
		updates["seo_keywords"] = *req.SEOKeywords
	}

	updated, err := s.productRepo.UpdateProduct(ctx, productID, updates)
	if err != nil {
		return nil, fmt.Errorf("update product: %w", err)
	}

	product, err := s.productRepo.GetProductByID(ctx, updated.ID)
	if err != nil {
		return nil, fmt.Errorf("get updated product: %w", err)
	}

	return s.toProductDetailFromProduct(ctx, product), nil
}

func (s *ProductService) DeleteProduct(ctx context.Context, productID int64) error {
	return s.productRepo.SoftDeleteProduct(ctx, productID)
}

func (s *ProductService) PublishProduct(ctx context.Context, productID int64) error {
	return s.productRepo.PublishProduct(ctx, productID)
}

func (s *ProductService) ArchiveProduct(ctx context.Context, productID int64) error {
	return s.productRepo.ArchiveProduct(ctx, productID)
}

func (s *ProductService) isValidHandlingClass(class string) bool {
	validClasses := []string{"ambient", "chilled", "frozen"}
	for _, c := range validClasses {
		if c == class {
			return true
		}
	}
	return false
}

func (s *ProductService) toProductSummary(p catalog.ListProductsRow) schema.ProductSummary {
	categoryCode := ""
	brandCode := ""
	supplierCode := p.SupplierCode.String
	baseUnitCode := ""

	var basePriceMinor *int64
	if p.BasePriceMinor.Valid {
		basePriceMinor = &p.BasePriceMinor.Int64
	}

	return schema.ProductSummary{
		ID:               p.ID,
		Code:             p.Code,
		Slug:             p.Slug,
		Name:             p.Name,
		ShortDescription: p.ShortDescription.String,
		CategoryCode:     categoryCode,
		BrandCode:        brandCode,
		HandlingClass:    string(p.HandlingClass),
		BaseUnitCode:     baseUnitCode,
		SupplierCode:     supplierCode,
		SupplierName:     p.SupplierName.String,
		Status:           string(p.Status),
		IsActive:         p.IsActive,
		IsFeatured:       p.IsFeatured,
		BasePriceMinor:   basePriceMinor,
		Currency:         p.Currency,
		TrackInventory:   p.TrackInventory,
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
		PublishedAt:      nil,
	}
}

func (s *ProductService) toProductDetail(p catalog.GetProductBySlugRow) *schema.ProductDetail {
	return &schema.ProductDetail{
		ProductSummary: schema.ProductSummary{
			ID:               p.ID,
			Code:             p.Code,
			Slug:             p.Slug,
			Name:             p.Name,
			ShortDescription: p.ShortDescription.String,
			CategoryCode:     p.CategorySlug.String,
			BrandCode:        p.BrandSlug.String,
			HandlingClass:    string(p.HandlingClass),
			BaseUnitCode:     p.BaseUnitSymbol.String,
			SupplierCode:     p.SupplierSlug.String,
			Status:           string(p.Status),
			IsActive:         p.IsActive,
			IsFeatured:       p.IsFeatured,
			BasePriceMinor:   &p.BasePriceMinor.Int64,
			Currency:         p.Currency,
			TrackInventory:   p.TrackInventory,
			CreatedAt:        p.CreatedAt,
			UpdatedAt:        p.UpdatedAt,
			PublishedAt:      &p.PublishedAt.Time,
		},
	}
}

func (s *ProductService) toFacets(facets []catalog.GetFacetsRow) []schema.Facet {
	var resp []schema.Facet
	for _, f := range facets {
		resp = append(resp, schema.Facet{
			FacetType: f.FacetType,
			ValueID:   f.ValueID,
			ValueName: f.ValueName,
			Count:     f.Count,
		})
	}
	return resp
}

func (s *ProductService) toFacetsFromFiltered(facets []catalog.GetFilteredFacetsRow) []schema.Facet {
	var resp []schema.Facet
	for _, f := range facets {
		resp = append(resp, schema.Facet{
			FacetType: f.FacetType,
			ValueID:   f.ValueID,
			ValueName: f.ValueName,
			Count:     f.Count,
		})
	}
	return resp
}

type CategoryService struct {
	categoryRepo *repository.CategoryRepository
}

func NewCategoryService() *CategoryService {
	return &CategoryService{}
}

func (s *CategoryService) SetDependencies(categoryRepo *repository.CategoryRepository) {
	s.categoryRepo = categoryRepo
}

func (s *CategoryService) ListCategories(ctx context.Context, req schema.CategoryListRequest) (*schema.CategoryListResponse, error) {
	limit := int32(req.PageSize)
	offset := int32((req.Page - 1) * req.PageSize)

	var parentID *int64
	if req.Parent != nil {
		parent, err := s.categoryRepo.GetCategoryByCode(ctx, *req.Parent)
		if err == nil {
			parentIDVal := parent.ID
			parentID = &parentIDVal
		}
	}

	categories, err := s.categoryRepo.ListCategories(ctx, limit, offset, parentID)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}

	total, err := s.categoryRepo.CountCategories(ctx, parentID)
	if err != nil {
		return nil, fmt.Errorf("count categories: %w", err)
	}

	var resp []schema.CategorySummary
	for _, c := range categories {
		resp = append(resp, *s.toCategorySummary(c))
	}

	return &schema.CategoryListResponse{
		Items:    resp,
		Page:     req.Page,
		PageSize: req.PageSize,
		Total:    total,
		HasNext:  int64(req.Page*req.PageSize) < total,
	}, nil
}

func (s *CategoryService) GetCategory(ctx context.Context, code string) (*schema.CategoryDetail, error) {
	category, err := s.categoryRepo.GetCategoryByCode(ctx, code)
	if err != nil {
		return nil, ErrCategoryNotFound
	}

	children, err := s.categoryRepo.GetCategoryChildren(ctx, category.ID)
	if err != nil {
		return nil, fmt.Errorf("get children: %w", err)
	}

	return &schema.CategoryDetail{
		CategorySummary: *s.toCategorySummary(category),
		Children:        s.toCategorySummaries(children),
	}, nil
}

func (s *CategoryService) CreateCategory(ctx context.Context, req schema.CreateCategoryRequest) (*schema.CategorySummary, error) {
	if req.ParentID != nil {
		if _, err := s.categoryRepo.GetCategoryByID(ctx, *req.ParentID); err != nil {
			return nil, ErrCategoryNotFound
		}
	}

	params := catalog.CreateCategoryParams{
		Code:           req.Code,
		Name:           req.Name,
		Slug:           req.Slug,
		Description:    pgtype.Text{String: req.Description, Valid: req.Description != ""},
		ParentID:       pgtype.Int8{Int64: *req.ParentID, Valid: req.ParentID != nil},
		SortOrder:      int32(req.SortOrder),
		SeoTitle:       pgtype.Text{String: req.SEOTitle, Valid: req.SEOTitle != ""},
		SeoDescription: pgtype.Text{String: req.SEODescription, Valid: req.SEODescription != ""},
	}

	categoryRow, err := s.categoryRepo.CreateCategory(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("create category: %w", err)
	}

	category, err := s.categoryRepo.GetCategoryByID(ctx, categoryRow.ID)
	if err != nil {
		return nil, fmt.Errorf("get created category: %w", err)
	}

	return s.toCategorySummary(category), nil
}

func (s *CategoryService) UpdateCategory(ctx context.Context, categoryID int64, req schema.UpdateCategoryRequest) (*schema.CategorySummary, error) {
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Slug != nil {
		updates["slug"] = *req.Slug
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.ParentID != nil {
		updates["parent_id"] = *req.ParentID
	}
	if req.SortOrder != nil {
		updates["sort_order"] = *req.SortOrder
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if req.SEOTitle != nil {
		updates["seo_title"] = *req.SEOTitle
	}
	if req.SEODescription != nil {
		updates["seo_description"] = *req.SEODescription
	}

	updated, err := s.categoryRepo.UpdateCategory(ctx, categoryID, updates)
	if err != nil {
		return nil, fmt.Errorf("update category: %w", err)
	}

	category, err := s.categoryRepo.GetCategoryByID(ctx, updated.ID)
	if err != nil {
		return nil, fmt.Errorf("get updated category: %w", err)
	}

	return s.toCategorySummary(category), nil
}

func (s *CategoryService) DeleteCategory(ctx context.Context, categoryID int64) error {
	return s.categoryRepo.SoftDeleteCategory(ctx, categoryID)
}

func (s *CategoryService) toCategorySummary(c catalog.CatalogCategory) *schema.CategorySummary {
	return &schema.CategorySummary{
		Code:        c.Code,
		Name:        c.Name,
		Slug:        c.Slug,
		Description: c.Description.String,
		SortOrder:   int(c.SortOrder),
		IsActive:    c.IsActive,
	}
}

func (s *CategoryService) toCategorySummaries(categories []catalog.CatalogCategory) []schema.CategorySummary {
	var resp []schema.CategorySummary
	for _, c := range categories {
		resp = append(resp, *s.toCategorySummary(c))
	}
	return resp
}

type BrandService struct {
	brandRepo *repository.BrandRepository
}

func NewBrandService() *BrandService {
	return &BrandService{}
}

func (s *BrandService) SetDependencies(brandRepo *repository.BrandRepository) {
	s.brandRepo = brandRepo
}

func (s *BrandService) ListBrands(ctx context.Context, req schema.BrandListRequest) (*schema.BrandListResponse, error) {
	limit := int32(req.PageSize)
	offset := int32((req.Page - 1) * req.PageSize)

	brands, err := s.brandRepo.ListBrands(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list brands: %w", err)
	}

	total, err := s.brandRepo.CountBrands(ctx)
	if err != nil {
		return nil, fmt.Errorf("count brands: %w", err)
	}

	var resp []schema.BrandSummary
	for _, b := range brands {
		resp = append(resp, schema.BrandSummary{
			Code:        b.Code,
			Name:        b.Name,
			Slug:        b.Slug,
			Description: b.Description.String,
			LogoURL:     b.LogoUrl.String,
			WebsiteURL:  b.WebsiteUrl.String,
			IsActive:    b.IsActive,
		})
	}

	return &schema.BrandListResponse{
		Items:    resp,
		Page:     req.Page,
		PageSize: req.PageSize,
		Total:    total,
		HasNext:  int64(req.Page*req.PageSize) < total,
	}, nil
}

func (s *BrandService) GetBrand(ctx context.Context, code string) (*schema.BrandSummary, error) {
	brand, err := s.brandRepo.GetBrandByCode(ctx, code)
	if err != nil {
		return nil, ErrBrandNotFound
	}
	return &schema.BrandSummary{
		Code:        brand.Code,
		Name:        brand.Name,
		Slug:        brand.Slug,
		Description: brand.Description.String,
		LogoURL:     brand.LogoUrl.String,
		WebsiteURL:  brand.WebsiteUrl.String,
		IsActive:    brand.IsActive,
	}, nil
}

func (s *BrandService) CreateBrand(ctx context.Context, req schema.CreateBrandRequest) (*schema.BrandSummary, error) {
	params := catalog.CreateBrandParams{
		Code:        req.Code,
		Name:        req.Name,
		Slug:        req.Slug,
		Description: pgtype.Text{String: req.Description, Valid: req.Description != ""},
		LogoUrl:     pgtype.Text{String: req.LogoURL, Valid: req.LogoURL != ""},
		WebsiteUrl:  pgtype.Text{String: req.WebsiteURL, Valid: req.WebsiteURL != ""},
	}

	brandRow, err := s.brandRepo.CreateBrand(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("create brand: %w", err)
	}

	brand, err := s.brandRepo.GetBrandByID(ctx, brandRow.ID)
	if err != nil {
		return nil, fmt.Errorf("get created brand: %w", err)
	}

	return &schema.BrandSummary{
		Code:        brand.Code,
		Name:        brand.Name,
		Slug:        brand.Slug,
		Description: brand.Description.String,
		LogoURL:     brand.LogoUrl.String,
		WebsiteURL:  brand.WebsiteUrl.String,
		IsActive:    brand.IsActive,
	}, nil
}

func (s *BrandService) UpdateBrand(ctx context.Context, brandID int64, req schema.UpdateBrandRequest) (*schema.BrandSummary, error) {
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Slug != nil {
		updates["slug"] = *req.Slug
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.LogoURL != nil {
		updates["logo_url"] = *req.LogoURL
	}
	if req.WebsiteURL != nil {
		updates["website_url"] = *req.WebsiteURL
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	updated, err := s.brandRepo.UpdateBrand(ctx, brandID, updates)
	if err != nil {
		return nil, fmt.Errorf("update brand: %w", err)
	}

	brand, err := s.brandRepo.GetBrandByID(ctx, updated.ID)
	if err != nil {
		return nil, fmt.Errorf("get updated brand: %w", err)
	}

	return &schema.BrandSummary{
		Code:        brand.Code,
		Name:        brand.Name,
		Slug:        brand.Slug,
		Description: brand.Description.String,
		LogoURL:     brand.LogoUrl.String,
		WebsiteURL:  brand.WebsiteUrl.String,
		IsActive:    brand.IsActive,
	}, nil
}

func (s *BrandService) DeleteBrand(ctx context.Context, brandID int64) error {
	return s.brandRepo.SoftDeleteBrand(ctx, brandID)
}

type UnitService struct {
	unitRepo *repository.UnitRepository
}

func NewUnitService() *UnitService {
	return &UnitService{}
}

func (s *UnitService) SetDependencies(unitRepo *repository.UnitRepository) {
	s.unitRepo = unitRepo
}

func (s *UnitService) ListUnits(ctx context.Context, limit, offset int32) ([]catalog.CatalogUnit, error) {
	return s.unitRepo.ListUnits(ctx, limit, offset)
}

func (s *UnitService) CountUnits(ctx context.Context) (int64, error) {
	return s.unitRepo.CountUnits(ctx)
}

func (s *UnitService) GetUnitByID(ctx context.Context, id int64) (catalog.CatalogUnit, error) {
	return s.unitRepo.GetUnitByID(ctx, id)
}

type SupplierService struct {
	supplierRepo *repository.SupplierRepository
}

func NewSupplierService() *SupplierService {
	return &SupplierService{}
}

func (s *SupplierService) SetDependencies(supplierRepo *repository.SupplierRepository) {
	s.supplierRepo = supplierRepo
}

func (s *SupplierService) ListSuppliers(ctx context.Context, limit, offset int32) ([]catalog.CatalogSupplier, error) {
	return s.supplierRepo.ListSuppliers(ctx, limit, offset)
}

func (s *SupplierService) ListSuppliersAdmin(ctx context.Context, limit, offset int32) ([]catalog.CatalogSupplier, error) {
	return s.supplierRepo.ListSuppliersAdmin(ctx, limit, offset)
}

func (s *SupplierService) CountSuppliers(ctx context.Context) (int64, error) {
	return s.supplierRepo.CountSuppliers(ctx)
}

type HandlingClassService struct {
	handlingClassRepo *repository.HandlingClassRepository
}

func NewHandlingClassService() *HandlingClassService {
	return &HandlingClassService{}
}

func (s *HandlingClassService) SetDependencies(handlingClassRepo *repository.HandlingClassRepository) {
	s.handlingClassRepo = handlingClassRepo
}

func (s *HandlingClassService) ListHandlingClasses(ctx context.Context) ([]catalog.CatalogHandlingClass, error) {
	return s.handlingClassRepo.ListHandlingClasses(ctx)
}

type FacetService struct {
	facetRepo *repository.FacetRepository
}

func NewFacetService() *FacetService {
	return &FacetService{}
}

func (s *FacetService) SetDependencies(facetRepo *repository.FacetRepository) {
	s.facetRepo = facetRepo
}

func (s *FacetService) GetFacets(ctx context.Context) ([]catalog.GetFacetsRow, error) {
	return s.facetRepo.GetFacets(ctx)
}

func (s *FacetService) GetFilteredFacets(ctx context.Context, filters repository.ProductListFilters) ([]catalog.GetFilteredFacetsRow, error) {
	return s.facetRepo.GetFilteredFacets(ctx, filters)
}

type Services struct {
	Product       *ProductService
	Category      *CategoryService
	Brand         *BrandService
	Unit          *UnitService
	Supplier      *SupplierService
	HandlingClass *HandlingClassService
	Facet         *FacetService
}

func NewServices(db *database.DB) *Services {
	// Create repositories
	productRepo := repository.NewProductRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	brandRepo := repository.NewBrandRepository(db)
	unitRepo := repository.NewUnitRepository(db)
	supplierRepo := repository.NewSupplierRepository(db)
	facetRepo := repository.NewFacetRepository(db)
	handlingClassRepo := repository.NewHandlingClassRepository(db)

	// Create services
	product := NewProductService()
	product.SetDependencies(productRepo, categoryRepo, brandRepo, unitRepo, supplierRepo, facetRepo)

	category := NewCategoryService()
	category.SetDependencies(categoryRepo)

	brand := NewBrandService()
	brand.SetDependencies(brandRepo)

	unit := NewUnitService()
	unit.SetDependencies(unitRepo)

	supplier := NewSupplierService()
	supplier.SetDependencies(supplierRepo)

	handlingClass := NewHandlingClassService()
	handlingClass.SetDependencies(handlingClassRepo)

	facet := NewFacetService()
	facet.SetDependencies(facetRepo)

	return &Services{
		Product:       product,
		Category:      category,
		Brand:         brand,
		Unit:          unit,
		Supplier:      supplier,
		HandlingClass: handlingClass,
		Facet:         facet,
	}
}
