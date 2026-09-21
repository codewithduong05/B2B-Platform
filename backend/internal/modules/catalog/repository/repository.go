package repository

import (
	"context"

	"github.com/atlas-platform/backend/internal/database"
	"github.com/atlas-platform/backend/internal/database/queries/catalog"
	"github.com/jackc/pgx/v5/pgtype"
)

type ProductRepository struct {
	db *database.DB
	q  *catalog.Queries
}

func NewProductRepository(db *database.DB) *ProductRepository {
	return &ProductRepository{
		db: db,
		q:  catalog.New(db.Pool),
	}
}

func (r *ProductRepository) CreateProduct(ctx context.Context, params catalog.CreateProductParams) (catalog.CreateProductRow, error) {
	return r.q.CreateProduct(ctx, params)
}

func (r *ProductRepository) GetProductByID(ctx context.Context, id int64) (catalog.GetProductByIDRow, error) {
	return r.q.GetProductByID(ctx, id)
}

func (r *ProductRepository) GetProductByCode(ctx context.Context, code string) (catalog.GetProductByCodeRow, error) {
	return r.q.GetProductByCode(ctx, code)
}

func (r *ProductRepository) GetProductBySlug(ctx context.Context, slug string) (catalog.GetProductBySlugRow, error) {
	return r.q.GetProductBySlug(ctx, slug)
}

func (r *ProductRepository) UpdateProduct(ctx context.Context, id int64, req map[string]interface{}) (catalog.UpdateProductRow, error) {
	params := catalog.UpdateProductParams{ID: id}
	if v, ok := req["slug"].(string); ok {
		params.Slug = v
	}
	if v, ok := req["name"].(string); ok {
		params.Name = v
	}
	if v, ok := req["description"].(string); ok {
		params.Description = pgtype.Text{String: v, Valid: true}
	}
	if v, ok := req["short_description"].(string); ok {
		params.ShortDescription = pgtype.Text{String: v, Valid: true}
	}
	if v, ok := req["category_id"].(int64); ok {
		params.CategoryID = v
	}
	if v, ok := req["brand_id"].(int64); ok {
		params.BrandID = pgtype.Int8{Int64: v, Valid: true}
	}
	if v, ok := req["handling_class"].(string); ok {
		params.HandlingClass = catalog.CatalogHandlingClassType(v)
	}
	if v, ok := req["base_unit_id"].(int64); ok {
		params.BaseUnitID = v
	}
	if v, ok := req["supplier_id"].(int64); ok {
		params.SupplierID = pgtype.Int8{Int64: v, Valid: true}
	}
	if v, ok := req["status"].(string); ok {
		params.Status = catalog.CatalogProductStatus(v)
	}
	if v, ok := req["is_active"].(bool); ok {
		params.IsActive = v
	}
	if v, ok := req["is_featured"].(bool); ok {
		params.IsFeatured = v
	}
	if v, ok := req["sort_order"].(int); ok {
		params.SortOrder = int32(v)
	}
	if v, ok := req["base_price_minor"].(int64); ok {
		params.BasePriceMinor = pgtype.Int8{Int64: v, Valid: true}
	}
	if v, ok := req["currency"].(string); ok {
		params.Currency = v
	}
	if v, ok := req["track_inventory"].(bool); ok {
		params.TrackInventory = v
	}
	if v, ok := req["weight_grams"].(int); ok {
		params.WeightGrams = pgtype.Int4{Int32: int32(v), Valid: true}
	}
	if v, ok := req["length_mm"].(int); ok {
		params.LengthMm = pgtype.Int4{Int32: int32(v), Valid: true}
	}
	if v, ok := req["width_mm"].(int); ok {
		params.WidthMm = pgtype.Int4{Int32: int32(v), Valid: true}
	}
	if v, ok := req["height_mm"].(int); ok {
		params.HeightMm = pgtype.Int4{Int32: int32(v), Valid: true}
	}
	if v, ok := req["gtin"].(string); ok {
		params.Gtin = pgtype.Text{String: v, Valid: true}
	}
	if v, ok := req["sku"].(string); ok {
		params.Sku = pgtype.Text{String: v, Valid: true}
	}
	if v, ok := req["seo_title"].(string); ok {
		params.SeoTitle = pgtype.Text{String: v, Valid: true}
	}
	if v, ok := req["seo_description"].(string); ok {
		params.SeoDescription = pgtype.Text{String: v, Valid: true}
	}
	if v, ok := req["seo_keywords"].(string); ok {
		params.SeoKeywords = pgtype.Text{String: v, Valid: true}
	}
	return r.q.UpdateProduct(ctx, params)
}

func (r *ProductRepository) PublishProduct(ctx context.Context, id int64) error {
	return r.q.PublishProduct(ctx, id)
}

func (r *ProductRepository) ArchiveProduct(ctx context.Context, id int64) error {
	return r.q.ArchiveProduct(ctx, id)
}

func (r *ProductRepository) SoftDeleteProduct(ctx context.Context, id int64) error {
	return r.q.SoftDeleteProduct(ctx, id)
}

type ProductListFilters struct {
	Status        *catalog.CatalogProductStatus
	CategoryID    *int64
	BrandID       *int64
	HandlingClass *catalog.CatalogHandlingClassType
	SupplierID    *int64
	Query         *string
	Sort          *string
}

func (r *ProductRepository) ListProducts(ctx context.Context, limit, offset int32, filters ProductListFilters) ([]catalog.ListProductsRow, error) {
	var status catalog.CatalogProductStatus
	var categoryID int64
	var brandID int64
	var handlingClass catalog.CatalogHandlingClassType
	var supplierID int64
	var query string
	var sort interface{}

	if filters.Status != nil {
		status = *filters.Status
	}
	if filters.CategoryID != nil {
		categoryID = *filters.CategoryID
	}
	if filters.BrandID != nil {
		brandID = *filters.BrandID
	}
	if filters.HandlingClass != nil {
		handlingClass = *filters.HandlingClass
	}
	if filters.SupplierID != nil {
		supplierID = *filters.SupplierID
	}
	if filters.Query != nil && *filters.Query != "" {
		query = *filters.Query
	}
	if filters.Sort != nil && *filters.Sort != "" {
		sort = *filters.Sort
	}

	params := catalog.ListProductsParams{
		Column1: string(status),
		Column2: categoryID,
		Column3: brandID,
		Column4: string(handlingClass),
		Column5: supplierID,
		Column6: query,
		Column7: sort,
		Limit:   limit,
		Offset:  offset,
	}

	return r.q.ListProducts(ctx, params)
}

func (r *ProductRepository) CountProducts(ctx context.Context, filters ProductListFilters) (int64, error) {
	var status catalog.CatalogProductStatus
	var categoryID int64
	var brandID int64
	var handlingClass catalog.CatalogHandlingClassType
	var supplierID int64
	var query string

	if filters.Status != nil {
		status = *filters.Status
	}
	if filters.CategoryID != nil {
		categoryID = *filters.CategoryID
	}
	if filters.BrandID != nil {
		brandID = *filters.BrandID
	}
	if filters.HandlingClass != nil {
		handlingClass = *filters.HandlingClass
	}
	if filters.SupplierID != nil {
		supplierID = *filters.SupplierID
	}
	if filters.Query != nil && *filters.Query != "" {
		query = *filters.Query
	}

	params := catalog.CountProductsParams{
		Column1: string(status),
		Column2: categoryID,
		Column3: brandID,
		Column4: string(handlingClass),
		Column5: supplierID,
		Column6: query,
	}

	return r.q.CountProducts(ctx, params)
}

func (r *ProductRepository) SearchProducts(ctx context.Context, query string, filters ProductListFilters, limit, offset int32) ([]catalog.SearchProductsRow, error) {
	var categoryID int64
	var brandID int64
	var handlingClass catalog.CatalogHandlingClassType
	var supplierID int64

	if filters.CategoryID != nil {
		categoryID = *filters.CategoryID
	}
	if filters.BrandID != nil {
		brandID = *filters.BrandID
	}
	if filters.HandlingClass != nil {
		handlingClass = *filters.HandlingClass
	}
	if filters.SupplierID != nil {
		supplierID = *filters.SupplierID
	}

	params := catalog.SearchProductsParams{
		PlaintoTsquery: query,
		Column2:        categoryID,
		Column3:        brandID,
		Column4:        handlingClass,
		Column5:        supplierID,
		Limit:          limit,
		Offset:         offset,
	}

	return r.q.SearchProducts(ctx, params)
}

func (r *ProductRepository) CountSearchProducts(ctx context.Context, query string, filters ProductListFilters) (int64, error) {
	var categoryID int64
	var brandID int64
	var handlingClass catalog.CatalogHandlingClassType
	var supplierID int64

	if filters.CategoryID != nil {
		categoryID = *filters.CategoryID
	}
	if filters.BrandID != nil {
		brandID = *filters.BrandID
	}
	if filters.HandlingClass != nil {
		handlingClass = *filters.HandlingClass
	}
	if filters.SupplierID != nil {
		supplierID = *filters.SupplierID
	}

	params := catalog.CountSearchProductsParams{
		Column1: categoryID,
		Column2: brandID,
		Column3: handlingClass,
		Column4: supplierID,
		Column5: query,
	}

	return r.q.CountSearchProducts(ctx, params)
}

func (r *ProductRepository) ListProductUnits(ctx context.Context, productID int64) ([]catalog.ListProductUnitsRow, error) {
	return r.q.ListProductUnits(ctx, productID)
}

func (r *ProductRepository) ListProductMedia(ctx context.Context, productID int64) ([]catalog.CatalogProductMedium, error) {
	return r.q.ListProductMedia(ctx, productID)
}

func (r *ProductRepository) ListProductAttributes(ctx context.Context, productID int64) ([]catalog.ListProductAttributesRow, error) {
	return r.q.ListProductAttributes(ctx, productID)
}

func (r *ProductRepository) WithTx(ctx context.Context, fn func(*ProductRepository) error) error {
	return r.db.WithTx(ctx, func(tx *database.Tx) error {
		txRepo := &ProductRepository{
			db: r.db,
			q:  catalog.New(tx),
		}
		return fn(txRepo)
	})
}

type CategoryRepository struct {
	db *database.DB
	q  *catalog.Queries
}

func NewCategoryRepository(db *database.DB) *CategoryRepository {
	return &CategoryRepository{
		db: db,
		q:  catalog.New(db.Pool),
	}
}

func (r *CategoryRepository) CreateCategory(ctx context.Context, params catalog.CreateCategoryParams) (catalog.CreateCategoryRow, error) {
	return r.q.CreateCategory(ctx, params)
}

func (r *CategoryRepository) GetCategoryByID(ctx context.Context, id int64) (catalog.CatalogCategory, error) {
	return r.q.GetCategoryByID(ctx, id)
}

func (r *CategoryRepository) GetCategoryByCode(ctx context.Context, code string) (catalog.CatalogCategory, error) {
	return r.q.GetCategoryByCode(ctx, code)
}

func (r *CategoryRepository) GetCategoryBySlug(ctx context.Context, slug string) (catalog.CatalogCategory, error) {
	return r.q.GetCategoryBySlug(ctx, slug)
}

func (r *CategoryRepository) GetCategoryTree(ctx context.Context) ([]catalog.GetCategoryTreeRow, error) {
	return r.q.GetCategoryTree(ctx)
}

func (r *CategoryRepository) GetCategoryChildren(ctx context.Context, parentID int64) ([]catalog.CatalogCategory, error) {
	return r.q.GetCategoryChildren(ctx, pgtype.Int8{Int64: parentID, Valid: true})
}

func (r *CategoryRepository) GetCategoryBreadcrumbs(ctx context.Context, categoryID int64) ([]catalog.GetCategoryBreadcrumbsRow, error) {
	return r.q.GetCategoryBreadcrumbs(ctx, categoryID)
}

func (r *CategoryRepository) UpdateCategory(ctx context.Context, id int64, req map[string]interface{}) (catalog.UpdateCategoryRow, error) {
	params := catalog.UpdateCategoryParams{ID: id}
	if v, ok := req["name"].(string); ok {
		params.Name = v
	}
	if v, ok := req["slug"].(string); ok {
		params.Slug = v
	}
	if v, ok := req["description"].(string); ok {
		params.Description = pgtype.Text{String: v, Valid: true}
	}
	if v, ok := req["parent_id"].(int64); ok {
		params.ParentID = pgtype.Int8{Int64: v, Valid: true}
	}
	if v, ok := req["sort_order"].(int); ok {
		params.SortOrder = int32(v)
	}
	if v, ok := req["is_active"].(bool); ok {
		params.IsActive = v
	}
	if v, ok := req["seo_title"].(string); ok {
		params.SeoTitle = pgtype.Text{String: v, Valid: true}
	}
	if v, ok := req["seo_description"].(string); ok {
		params.SeoDescription = pgtype.Text{String: v, Valid: true}
	}
	return r.q.UpdateCategory(ctx, params)
}

func (r *CategoryRepository) SoftDeleteCategory(ctx context.Context, id int64) error {
	return r.q.SoftDeleteCategory(ctx, id)
}

func (r *CategoryRepository) ListCategories(ctx context.Context, limit, offset int32, parentID *int64) ([]catalog.CatalogCategory, error) {
	var pid int64
	if parentID != nil {
		pid = *parentID
	}
	return r.q.ListCategories(ctx, catalog.ListCategoriesParams{
		Column1: pid,
		Limit:   limit,
		Offset:  offset,
	})
}

func (r *CategoryRepository) CountCategories(ctx context.Context, parentID *int64) (int64, error) {
	var pid int64
	if parentID != nil {
		pid = *parentID
	}
	return r.q.CountCategories(ctx, pid)
}

type BrandRepository struct {
	db *database.DB
	q  *catalog.Queries
}

func NewBrandRepository(db *database.DB) *BrandRepository {
	return &BrandRepository{
		db: db,
		q:  catalog.New(db.Pool),
	}
}

func (r *BrandRepository) CreateBrand(ctx context.Context, params catalog.CreateBrandParams) (catalog.CreateBrandRow, error) {
	return r.q.CreateBrand(ctx, params)
}

func (r *BrandRepository) GetBrandByID(ctx context.Context, id int64) (catalog.CatalogBrand, error) {
	return r.q.GetBrandByID(ctx, id)
}

func (r *BrandRepository) GetBrandByCode(ctx context.Context, code string) (catalog.CatalogBrand, error) {
	return r.q.GetBrandByCode(ctx, code)
}

func (r *BrandRepository) GetBrandBySlug(ctx context.Context, slug string) (catalog.CatalogBrand, error) {
	return r.q.GetBrandBySlug(ctx, slug)
}

func (r *BrandRepository) UpdateBrand(ctx context.Context, id int64, req map[string]interface{}) (catalog.UpdateBrandRow, error) {
	params := catalog.UpdateBrandParams{ID: id}
	if v, ok := req["name"].(string); ok {
		params.Name = v
	}
	if v, ok := req["slug"].(string); ok {
		params.Slug = v
	}
	if v, ok := req["description"].(string); ok {
		params.Description = pgtype.Text{String: v, Valid: true}
	}
	if v, ok := req["logo_url"].(string); ok {
		params.LogoUrl = pgtype.Text{String: v, Valid: true}
	}
	if v, ok := req["website_url"].(string); ok {
		params.WebsiteUrl = pgtype.Text{String: v, Valid: true}
	}
	if v, ok := req["is_active"].(bool); ok {
		params.IsActive = v
	}
	return r.q.UpdateBrand(ctx, params)
}

func (r *BrandRepository) SoftDeleteBrand(ctx context.Context, id int64) error {
	return r.q.SoftDeleteBrand(ctx, id)
}

func (r *BrandRepository) ListBrands(ctx context.Context, limit, offset int32) ([]catalog.CatalogBrand, error) {
	return r.q.ListBrands(ctx, catalog.ListBrandsParams{
		Limit:  limit,
		Offset: offset,
	})
}

func (r *BrandRepository) ListBrandsAdmin(ctx context.Context, limit, offset int32) ([]catalog.CatalogBrand, error) {
	return r.q.ListBrandsAdmin(ctx, catalog.ListBrandsAdminParams{
		Limit:  limit,
		Offset: offset,
	})
}

func (r *BrandRepository) CountBrands(ctx context.Context) (int64, error) {
	return r.q.CountBrands(ctx)
}

type UnitRepository struct {
	db *database.DB
	q  *catalog.Queries
}

func NewUnitRepository(db *database.DB) *UnitRepository {
	return &UnitRepository{
		db: db,
		q:  catalog.New(db.Pool),
	}
}

func (r *UnitRepository) CreateUnit(ctx context.Context, params catalog.CreateUnitParams) (catalog.CreateUnitRow, error) {
	return r.q.CreateUnit(ctx, params)
}

func (r *UnitRepository) GetUnitByID(ctx context.Context, id int64) (catalog.CatalogUnit, error) {
	return r.q.GetUnitByID(ctx, id)
}

func (r *UnitRepository) GetUnitByCode(ctx context.Context, code string) (catalog.CatalogUnit, error) {
	return r.q.GetUnitByCode(ctx, code)
}

func (r *UnitRepository) UpdateUnit(ctx context.Context, id int64, req map[string]interface{}) (catalog.UpdateUnitRow, error) {
	params := catalog.UpdateUnitParams{ID: id}
	if v, ok := req["name"].(string); ok {
		params.Name = v
	}
	if v, ok := req["symbol"].(string); ok {
		params.Symbol = v
	}
	if v, ok := req["unit_type"].(string); ok {
		params.UnitType = catalog.CatalogUnitType(v)
	}
	if v, ok := req["base_unit_id"].(int64); ok {
		params.BaseUnitID = pgtype.Int8{Int64: v, Valid: true}
	}
	if v, ok := req["conversion_factor"].(float64); ok {
		params.ConversionFactor = pgtype.Numeric{Valid: true}
		_ = v
	}
	if v, ok := req["is_active"].(bool); ok {
		params.IsActive = v
	}
	return r.q.UpdateUnit(ctx, params)
}

func (r *UnitRepository) SoftDeleteUnit(ctx context.Context, id int64) error {
	return r.q.SoftDeleteUnit(ctx, id)
}

func (r *UnitRepository) ListUnits(ctx context.Context, limit, offset int32) ([]catalog.CatalogUnit, error) {
	return r.q.ListUnits(ctx, catalog.ListUnitsParams{
		Limit:  limit,
		Offset: offset,
	})
}

func (r *UnitRepository) CountUnits(ctx context.Context) (int64, error) {
	return r.q.CountUnits(ctx)
}

type HandlingClassRepository struct {
	db *database.DB
	q  *catalog.Queries
}

func NewHandlingClassRepository(db *database.DB) *HandlingClassRepository {
	return &HandlingClassRepository{
		db: db,
		q:  catalog.New(db.Pool),
	}
}

func (r *HandlingClassRepository) ListHandlingClasses(ctx context.Context) ([]catalog.CatalogHandlingClass, error) {
	return r.q.ListHandlingClasses(ctx)
}

func (r *HandlingClassRepository) GetHandlingClassByID(ctx context.Context, id int64) (catalog.CatalogHandlingClass, error) {
	return r.q.GetHandlingClassByID(ctx, id)
}

func (r *HandlingClassRepository) GetHandlingClassByCode(ctx context.Context, code string) (catalog.CatalogHandlingClass, error) {
	return r.q.GetHandlingClassByCode(ctx, code)
}

type SupplierRepository struct {
	db *database.DB
	q  *catalog.Queries
}

func NewSupplierRepository(db *database.DB) *SupplierRepository {
	return &SupplierRepository{
		db: db,
		q:  catalog.New(db.Pool),
	}
}

func (r *SupplierRepository) CreateSupplier(ctx context.Context, params catalog.CreateSupplierParams) (catalog.CreateSupplierRow, error) {
	return r.q.CreateSupplier(ctx, params)
}

func (r *SupplierRepository) GetSupplierByID(ctx context.Context, id int64) (catalog.CatalogSupplier, error) {
	return r.q.GetSupplierByID(ctx, id)
}

func (r *SupplierRepository) GetSupplierByCode(ctx context.Context, code string) (catalog.CatalogSupplier, error) {
	return r.q.GetSupplierByCode(ctx, code)
}

func (r *SupplierRepository) GetSupplierBySlug(ctx context.Context, slug string) (catalog.CatalogSupplier, error) {
	return r.q.GetSupplierBySlug(ctx, slug)
}

func (r *SupplierRepository) UpdateSupplier(ctx context.Context, id int64, req map[string]interface{}) (catalog.UpdateSupplierRow, error) {
	params := catalog.UpdateSupplierParams{ID: id}
	if v, ok := req["name"].(string); ok {
		params.Name = v
	}
	if v, ok := req["slug"].(string); ok {
		params.Slug = v
	}
	if v, ok := req["description"].(string); ok {
		params.Description = pgtype.Text{String: v, Valid: true}
	}
	if v, ok := req["logo_url"].(string); ok {
		params.LogoUrl = pgtype.Text{String: v, Valid: true}
	}
	if v, ok := req["is_active"].(bool); ok {
		params.IsActive = v
	}
	return r.q.UpdateSupplier(ctx, params)
}

func (r *SupplierRepository) SoftDeleteSupplier(ctx context.Context, id int64) error {
	return r.q.SoftDeleteSupplier(ctx, id)
}

func (r *SupplierRepository) ListSuppliers(ctx context.Context, limit, offset int32) ([]catalog.CatalogSupplier, error) {
	return r.q.ListSuppliers(ctx, catalog.ListSuppliersParams{
		Limit:  limit,
		Offset: offset,
	})
}

func (r *SupplierRepository) ListSuppliersAdmin(ctx context.Context, limit, offset int32) ([]catalog.CatalogSupplier, error) {
	return r.q.ListSuppliersAdmin(ctx, catalog.ListSuppliersAdminParams{
		Limit:  limit,
		Offset: offset,
	})
}

func (r *SupplierRepository) CountSuppliers(ctx context.Context) (int64, error) {
	return r.q.CountSuppliers(ctx)
}

type FacetRepository struct {
	db *database.DB
	q  *catalog.Queries
}

func NewFacetRepository(db *database.DB) *FacetRepository {
	return &FacetRepository{
		db: db,
		q:  catalog.New(db.Pool),
	}
}

func (r *FacetRepository) GetFacets(ctx context.Context) ([]catalog.GetFacetsRow, error) {
	return r.q.GetFacets(ctx)
}

func (r *FacetRepository) GetFilteredFacets(ctx context.Context, filters ProductListFilters) ([]catalog.GetFilteredFacetsRow, error) {
	var categoryID int64
	var brandID int64
	var handlingClass catalog.CatalogHandlingClassType
	var supplierID int64
	var query string

	if filters.CategoryID != nil {
		categoryID = *filters.CategoryID
	}
	if filters.BrandID != nil {
		brandID = *filters.BrandID
	}
	if filters.HandlingClass != nil {
		handlingClass = *filters.HandlingClass
	}
	if filters.SupplierID != nil {
		supplierID = *filters.SupplierID
	}
	if filters.Query != nil && *filters.Query != "" {
		query = *filters.Query
	}

	params := catalog.GetFilteredFacetsParams{
		Column1: categoryID,
		Column2: brandID,
		Column3: string(handlingClass),
		Column4: supplierID,
		Column5: query,
	}

	return r.q.GetFilteredFacets(ctx, params)
}

func (r *FacetRepository) GetAttributeFacets(ctx context.Context, attributeID *int64) ([]catalog.GetAttributeFacetsRow, error) {
	if attributeID != nil {
		return r.q.GetAttributeFacets(ctx, *attributeID)
	}
	return r.q.GetAttributeFacets(ctx, 0)
}
