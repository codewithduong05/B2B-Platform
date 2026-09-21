package router

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/atlas-platform/backend/internal/modules/catalog/schema"
	"github.com/atlas-platform/backend/internal/modules/catalog/service"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Router struct {
	router  chi.Router
	service *service.Services
}

func New(svc *service.Services) *Router {
	r := chi.NewRouter()
	return &Router{router: r, service: svc}
}

func (rt *Router) ChiRouter() chi.Router {
	return rt.router
}

func (rt *Router) RegisterRoutes() {
	// Public routes
	rt.router.Route("/catalog", func(r chi.Router) {
		r.Get("/products", rt.handleListProducts)
		r.Get("/products/search", rt.handleSearchProducts)
		r.Get("/products/{slug}", rt.handleGetProduct)
		r.Get("/categories", rt.handleListCategories)
		r.Get("/categories/{slug}", rt.handleGetCategory)
		r.Get("/brands", rt.handleListBrands)
		r.Get("/brands/{slug}", rt.handleGetBrand)
		r.Get("/units", rt.handleListUnits)
		r.Get("/handling-classes", rt.handleListHandlingClasses)
		r.Get("/suppliers", rt.handleListSuppliers)
		r.Get("/facets", rt.handleGetFacets)
	})

	// Admin routes
	rt.router.Route("/admin/catalog", func(r chi.Router) {
		r.Post("/products", rt.handleCreateProduct)
		r.Get("/products", rt.handleAdminListProducts)
		r.Get("/products/{id}", rt.handleAdminGetProduct)
		r.Patch("/products/{id}", rt.handleAdminUpdateProduct)
		r.Post("/products/{id}/publish", rt.handleAdminPublishProduct)
		r.Post("/products/{id}/archive", rt.handleAdminArchiveProduct)

		r.Post("/categories", rt.handleAdminCreateCategory)
		r.Get("/categories", rt.handleAdminListCategories)
		r.Get("/categories/{id}", rt.handleAdminGetCategory)
		r.Patch("/categories/{id}", rt.handleAdminUpdateCategory)
		r.Delete("/categories/{id}", rt.handleAdminDeleteCategory)

		r.Post("/brands", rt.handleAdminCreateBrand)
		r.Get("/brands", rt.handleAdminListBrands)
		r.Get("/brands/{id}", rt.handleAdminGetBrand)
		r.Patch("/brands/{id}", rt.handleAdminUpdateBrand)
		r.Delete("/brands/{id}", rt.handleAdminDeleteBrand)

		r.Post("/units", rt.handleAdminCreateUnit)
		r.Get("/units", rt.handleAdminListUnits)
		r.Get("/units/{id}", rt.handleAdminGetUnit)
		r.Patch("/units/{id}", rt.handleAdminUpdateUnit)
		r.Delete("/units/{id}", rt.handleAdminDeleteUnit)

		r.Post("/suppliers", rt.handleAdminCreateSupplier)
		r.Get("/suppliers", rt.handleAdminListSuppliers)
		r.Get("/suppliers/{id}", rt.handleAdminGetSupplier)
		r.Patch("/suppliers/{id}", rt.handleAdminUpdateSupplier)
		r.Delete("/suppliers/{id}", rt.handleAdminDeleteSupplier)
	})
}

func (rt *Router) handleListProducts(w http.ResponseWriter, r *http.Request) {
	var req schema.ProductListRequest
	if err := rt.decodeQuery(r, &req); err != nil {
		rt.writeError(w, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	resp, err := rt.service.Product.ListProducts(r.Context(), req)
	if err != nil {
		rt.writeError(w, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusOK, resp)
}

func (rt *Router) handleSearchProducts(w http.ResponseWriter, r *http.Request) {
	var req schema.ProductListRequest
	if err := rt.decodeQuery(r, &req); err != nil {
		rt.writeError(w, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	// Search uses the same request structure but calls search service
	// For now, we'll reuse ListProducts with search query
	resp, err := rt.service.Product.ListProducts(r.Context(), req)
	if err != nil {
		rt.writeError(w, http.StatusInternalServerError, "search_failed", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusOK, resp)
}

func (rt *Router) handleGetProduct(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		rt.writeError(w, http.StatusBadRequest, "missing_slug", "product slug is required")
		return
	}

	product, err := rt.service.Product.GetProduct(r.Context(), slug)
	if err != nil {
		if err == service.ErrProductNotFound || err == service.ErrProductNotPublished {
			rt.writeError(w, http.StatusNotFound, "not_found", "product not found")
			return
		}
		rt.writeError(w, http.StatusInternalServerError, "get_failed", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusOK, schema.ProductDetailResponse{Product: *product})
}

func (rt *Router) handleListCategories(w http.ResponseWriter, r *http.Request) {
	var req schema.CategoryListRequest
	if err := rt.decodeQuery(r, &req); err != nil {
		rt.writeError(w, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	resp, err := rt.service.Category.ListCategories(r.Context(), req)
	if err != nil {
		rt.writeError(w, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusOK, resp)
}

func (rt *Router) handleGetCategory(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		rt.writeError(w, http.StatusBadRequest, "missing_slug", "category slug is required")
		return
	}

	category, err := rt.service.Category.GetCategory(r.Context(), slug)
	if err != nil {
		if err == service.ErrCategoryNotFound {
			rt.writeError(w, http.StatusNotFound, "not_found", "category not found")
			return
		}
		rt.writeError(w, http.StatusInternalServerError, "get_failed", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusOK, category)
}

func (rt *Router) handleListBrands(w http.ResponseWriter, r *http.Request) {
	var req schema.BrandListRequest
	if err := rt.decodeQuery(r, &req); err != nil {
		rt.writeError(w, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	resp, err := rt.service.Brand.ListBrands(r.Context(), req)
	if err != nil {
		rt.writeError(w, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusOK, resp)
}

func (rt *Router) handleGetBrand(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		rt.writeError(w, http.StatusBadRequest, "missing_slug", "brand slug is required")
		return
	}

	brand, err := rt.service.Brand.GetBrand(r.Context(), slug)
	if err != nil {
		if err == service.ErrBrandNotFound {
			rt.writeError(w, http.StatusNotFound, "not_found", "brand not found")
			return
		}
		rt.writeError(w, http.StatusInternalServerError, "get_failed", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusOK, brand)
}

func (rt *Router) handleListUnits(w http.ResponseWriter, r *http.Request) {
	var req schema.BrandListRequest // Reuse pagination params
	if err := rt.decodeQuery(r, &req); err != nil {
		rt.writeError(w, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	// Units don't have a dedicated list request, use default pagination
	limit := int32(req.PageSize)
	offset := int32((req.Page - 1) * req.PageSize)

	units, err := rt.service.Unit.ListUnits(r.Context(), limit, offset)
	if err != nil {
		rt.writeError(w, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}

	// Convert to schema.UnitSummary
	var resp []schema.UnitSummary
	for _, u := range units {
		resp = append(resp, schema.UnitSummary{
			Code:             u.Code,
			Name:             u.Name,
			Symbol:           u.Symbol,
			UnitType:         string(u.UnitType),
			BaseUnitCode:     "", // Would need lookup
			ConversionFactor: conversionFactorToFloat64(u.ConversionFactor),
			IsActive:         u.IsActive,
		})
	}

	total, _ := rt.service.Unit.CountUnits(r.Context())
	rt.writeJSON(w, http.StatusOK, map[string]interface{}{
		"items":     resp,
		"page":      req.Page,
		"page_size": req.PageSize,
		"total":     total,
		"has_next":  int64(req.Page*req.PageSize) < total,
	})
}

func (rt *Router) handleListHandlingClasses(w http.ResponseWriter, r *http.Request) {
	classes, err := rt.service.HandlingClass.ListHandlingClasses(r.Context())
	if err != nil {
		rt.writeError(w, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}

	var resp []schema.HandlingClassSummary
	for _, c := range classes {
		var minC, maxC *int
		if c.TemperatureMinC.Valid {
			v := int(c.TemperatureMinC.Int32)
			minC = &v
		}
		if c.TemperatureMaxC.Valid {
			v := int(c.TemperatureMaxC.Int32)
			maxC = &v
		}
		resp = append(resp, schema.HandlingClassSummary{
			Code:            c.Code,
			Name:            string(c.Name),
			Description:     c.Description.String,
			TemperatureMinC: minC,
			TemperatureMaxC: maxC,
		})
	}

	rt.writeJSON(w, http.StatusOK, resp)
}

func (rt *Router) handleListSuppliers(w http.ResponseWriter, r *http.Request) {
	var req schema.BrandListRequest // Reuse pagination params
	if err := rt.decodeQuery(r, &req); err != nil {
		rt.writeError(w, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	limit := int32(req.PageSize)
	offset := int32((req.Page - 1) * req.PageSize)

	suppliers, err := rt.service.Supplier.ListSuppliers(r.Context(), limit, offset)
	if err != nil {
		rt.writeError(w, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}

	var resp []schema.SupplierSummary
	for _, s := range suppliers {
		resp = append(resp, schema.SupplierSummary{
			Code:        s.Code,
			SupplierID:  s.SupplierID,
			Name:        s.Name,
			Slug:        s.Slug,
			Description: s.Description.String,
			LogoURL:     s.LogoUrl.String,
			IsActive:    s.IsActive,
		})
	}

	total, _ := rt.service.Supplier.CountSuppliers(r.Context())
	rt.writeJSON(w, http.StatusOK, map[string]interface{}{
		"items":     resp,
		"page":      req.Page,
		"page_size": req.PageSize,
		"total":     total,
		"has_next":  int64(req.Page*req.PageSize) < total,
	})
}

func (rt *Router) handleGetFacets(w http.ResponseWriter, r *http.Request) {
	facets, err := rt.service.Facet.GetFacets(r.Context())
	if err != nil {
		rt.writeError(w, http.StatusInternalServerError, "facets_failed", err.Error())
		return
	}

	var resp []schema.Facet
	for _, f := range facets {
		resp = append(resp, schema.Facet{
			FacetType: f.FacetType,
			ValueID:   f.ValueID,
			ValueName: f.ValueName,
			Count:     f.Count,
		})
	}

	rt.writeJSON(w, http.StatusOK, schema.FacetResponse{Facets: resp})
}

// Admin handlers
func (rt *Router) handleCreateProduct(w http.ResponseWriter, r *http.Request) {
	var req schema.CreateProductRequest
	if err := rt.decodeBody(r, &req); err != nil {
		rt.writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}

	product, err := rt.service.Product.CreateProduct(r.Context(), req)
	if err != nil {
		if err == service.ErrInvalidCategory {
			rt.writeError(w, http.StatusBadRequest, "invalid_category", "invalid category")
			return
		}
		if err == service.ErrInvalidBrand {
			rt.writeError(w, http.StatusBadRequest, "invalid_brand", "invalid brand")
			return
		}
		if err == service.ErrInvalidUnit {
			rt.writeError(w, http.StatusBadRequest, "invalid_unit", "invalid unit")
			return
		}
		if err == service.ErrInvalidSupplier {
			rt.writeError(w, http.StatusBadRequest, "invalid_supplier", "invalid supplier")
			return
		}
		if err == service.ErrInvalidHandlingClass {
			rt.writeError(w, http.StatusBadRequest, "invalid_handling_class", "invalid handling class")
			return
		}
		if err == service.ErrInvalidSlug {
			rt.writeError(w, http.StatusConflict, "duplicate_slug", "slug already exists")
			return
		}
		rt.writeError(w, http.StatusInternalServerError, "create_failed", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusCreated, schema.ProductDetailResponse{Product: *product})
}

func (rt *Router) handleAdminListProducts(w http.ResponseWriter, r *http.Request) {
	var req schema.ProductListRequest
	if err := rt.decodeQuery(r, &req); err != nil {
		rt.writeError(w, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	resp, err := rt.service.Product.ListProducts(r.Context(), req)
	if err != nil {
		rt.writeError(w, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusOK, resp)
}

func (rt *Router) handleAdminGetProduct(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		rt.writeError(w, http.StatusBadRequest, "invalid_id", "invalid product ID")
		return
	}

	_ = id
	rt.writeError(w, http.StatusNotImplemented, "not_implemented", "admin get product by ID not implemented")
}

func (rt *Router) handleAdminUpdateProduct(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		rt.writeError(w, http.StatusBadRequest, "invalid_id", "invalid product ID")
		return
	}

	var req schema.UpdateProductRequest
	if err := rt.decodeBody(r, &req); err != nil {
		rt.writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}

	product, err := rt.service.Product.UpdateProduct(r.Context(), id, req)
	if err != nil {
		rt.writeError(w, http.StatusInternalServerError, "update_failed", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusOK, schema.ProductDetailResponse{Product: *product})
}

func (rt *Router) handleAdminPublishProduct(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		rt.writeError(w, http.StatusBadRequest, "invalid_id", "invalid product ID")
		return
	}

	err = rt.service.Product.PublishProduct(r.Context(), id)
	if err != nil {
		rt.writeError(w, http.StatusInternalServerError, "publish_failed", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusOK, map[string]string{"status": "published"})
}

func (rt *Router) handleAdminArchiveProduct(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		rt.writeError(w, http.StatusBadRequest, "invalid_id", "invalid product ID")
		return
	}

	err = rt.service.Product.ArchiveProduct(r.Context(), id)
	if err != nil {
		rt.writeError(w, http.StatusInternalServerError, "archive_failed", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusOK, map[string]string{"status": "archived"})
}

func (rt *Router) handleAdminCreateCategory(w http.ResponseWriter, r *http.Request) {
	var req schema.CreateCategoryRequest
	if err := rt.decodeBody(r, &req); err != nil {
		rt.writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}

	category, err := rt.service.Category.CreateCategory(r.Context(), req)
	if err != nil {
		rt.writeError(w, http.StatusInternalServerError, "create_failed", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusCreated, category)
}

func (rt *Router) handleAdminListCategories(w http.ResponseWriter, r *http.Request) {
	var req schema.CategoryListRequest
	if err := rt.decodeQuery(r, &req); err != nil {
		rt.writeError(w, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	resp, err := rt.service.Category.ListCategories(r.Context(), req)
	if err != nil {
		rt.writeError(w, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusOK, resp)
}

func (rt *Router) handleAdminGetCategory(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		rt.writeError(w, http.StatusBadRequest, "invalid_id", "invalid category ID")
		return
	}

	// Would need GetCategoryByID in service
	_ = id
	rt.writeError(w, http.StatusNotImplemented, "not_implemented", "admin get category by ID not implemented")
}

func (rt *Router) handleAdminUpdateCategory(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		rt.writeError(w, http.StatusBadRequest, "invalid_id", "invalid category ID")
		return
	}

	var req schema.UpdateCategoryRequest
	if err := rt.decodeBody(r, &req); err != nil {
		rt.writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}

	category, err := rt.service.Category.UpdateCategory(r.Context(), id, req)
	if err != nil {
		rt.writeError(w, http.StatusInternalServerError, "update_failed", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusOK, category)
}

func (rt *Router) handleAdminDeleteCategory(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		rt.writeError(w, http.StatusBadRequest, "invalid_id", "invalid category ID")
		return
	}

	err = rt.service.Category.DeleteCategory(r.Context(), id)
	if err != nil {
		rt.writeError(w, http.StatusInternalServerError, "delete_failed", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (rt *Router) handleAdminCreateBrand(w http.ResponseWriter, r *http.Request) {
	var req schema.CreateBrandRequest
	if err := rt.decodeBody(r, &req); err != nil {
		rt.writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}

	brand, err := rt.service.Brand.CreateBrand(r.Context(), req)
	if err != nil {
		rt.writeError(w, http.StatusInternalServerError, "create_failed", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusCreated, brand)
}

func (rt *Router) handleAdminListBrands(w http.ResponseWriter, r *http.Request) {
	var req schema.BrandListRequest
	if err := rt.decodeQuery(r, &req); err != nil {
		rt.writeError(w, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	resp, err := rt.service.Brand.ListBrands(r.Context(), req)
	if err != nil {
		rt.writeError(w, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusOK, resp)
}

func (rt *Router) handleAdminGetBrand(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		rt.writeError(w, http.StatusBadRequest, "invalid_id", "invalid brand ID")
		return
	}

	_ = id
	rt.writeError(w, http.StatusNotImplemented, "not_implemented", "admin get brand by ID not implemented")
}

func (rt *Router) handleAdminUpdateBrand(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		rt.writeError(w, http.StatusBadRequest, "invalid_id", "invalid brand ID")
		return
	}

	var req schema.UpdateBrandRequest
	if err := rt.decodeBody(r, &req); err != nil {
		rt.writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}

	brand, err := rt.service.Brand.UpdateBrand(r.Context(), id, req)
	if err != nil {
		rt.writeError(w, http.StatusInternalServerError, "update_failed", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusOK, brand)
}

func (rt *Router) handleAdminDeleteBrand(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		rt.writeError(w, http.StatusBadRequest, "invalid_id", "invalid brand ID")
		return
	}

	err = rt.service.Brand.DeleteBrand(r.Context(), id)
	if err != nil {
		rt.writeError(w, http.StatusInternalServerError, "delete_failed", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (rt *Router) handleAdminCreateUnit(w http.ResponseWriter, r *http.Request) {
	var req schema.CreateUnitRequest
	if err := rt.decodeBody(r, &req); err != nil {
		rt.writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}

	// Unit service doesn't have CreateUnit implemented yet
	_ = req
	rt.writeError(w, http.StatusNotImplemented, "not_implemented", "create unit not implemented")
}

func (rt *Router) handleAdminListUnits(w http.ResponseWriter, r *http.Request) {
	var req schema.BrandListRequest
	if err := rt.decodeQuery(r, &req); err != nil {
		rt.writeError(w, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	limit := int32(req.PageSize)
	offset := int32((req.Page - 1) * req.PageSize)

	units, err := rt.service.Unit.ListUnits(r.Context(), limit, offset)
	if err != nil {
		rt.writeError(w, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}

	var resp []schema.UnitSummary
	for _, u := range units {
		resp = append(resp, schema.UnitSummary{
			Code:             u.Code,
			Name:             u.Name,
			Symbol:           u.Symbol,
			UnitType:         string(u.UnitType),
			BaseUnitCode:     "",
			ConversionFactor: 1.0,
			IsActive:         u.IsActive,
		})
	}

	total, _ := rt.service.Unit.CountUnits(r.Context())
	rt.writeJSON(w, http.StatusOK, map[string]interface{}{
		"items":     resp,
		"page":      req.Page,
		"page_size": req.PageSize,
		"total":     total,
		"has_next":  int64(req.Page*req.PageSize) < total,
	})
}

func (rt *Router) handleAdminGetUnit(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		rt.writeError(w, http.StatusBadRequest, "invalid_id", "invalid unit ID")
		return
	}

	_ = id
	rt.writeError(w, http.StatusNotImplemented, "not_implemented", "admin get unit by ID not implemented")
}

func (rt *Router) handleAdminUpdateUnit(w http.ResponseWriter, r *http.Request) {
	rt.writeError(w, http.StatusNotImplemented, "not_implemented", "update unit not implemented")
}

func (rt *Router) handleAdminDeleteUnit(w http.ResponseWriter, r *http.Request) {
	rt.writeError(w, http.StatusNotImplemented, "not_implemented", "delete unit not implemented")
}

func (rt *Router) handleAdminCreateSupplier(w http.ResponseWriter, r *http.Request) {
	var req schema.CreateSupplierRequest
	if err := rt.decodeBody(r, &req); err != nil {
		rt.writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}

	_ = req
	rt.writeError(w, http.StatusNotImplemented, "not_implemented", "create supplier not implemented")
}

func (rt *Router) handleAdminListSuppliers(w http.ResponseWriter, r *http.Request) {
	var req schema.BrandListRequest
	if err := rt.decodeQuery(r, &req); err != nil {
		rt.writeError(w, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	limit := int32(req.PageSize)
	offset := int32((req.Page - 1) * req.PageSize)

	suppliers, err := rt.service.Supplier.ListSuppliersAdmin(r.Context(), limit, offset)
	if err != nil {
		rt.writeError(w, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}

	var resp []schema.SupplierSummary
	for _, s := range suppliers {
		resp = append(resp, schema.SupplierSummary{
			Code:        s.Code,
			SupplierID:  s.SupplierID,
			Name:        s.Name,
			Slug:        s.Slug,
			Description: s.Description.String,
			LogoURL:     s.LogoUrl.String,
			IsActive:    s.IsActive,
		})
	}

	total, _ := rt.service.Supplier.CountSuppliers(r.Context())
	rt.writeJSON(w, http.StatusOK, map[string]interface{}{
		"items":     resp,
		"page":      req.Page,
		"page_size": req.PageSize,
		"total":     total,
		"has_next":  int64(req.Page*req.PageSize) < total,
	})
}

func (rt *Router) handleAdminGetSupplier(w http.ResponseWriter, r *http.Request) {
	rt.writeError(w, http.StatusNotImplemented, "not_implemented", "admin get supplier by ID not implemented")
}

func (rt *Router) handleAdminUpdateSupplier(w http.ResponseWriter, r *http.Request) {
	rt.writeError(w, http.StatusNotImplemented, "not_implemented", "update supplier not implemented")
}

func (rt *Router) handleAdminDeleteSupplier(w http.ResponseWriter, r *http.Request) {
	rt.writeError(w, http.StatusNotImplemented, "not_implemented", "delete supplier not implemented")
}

// Helper methods
func (rt *Router) decodeQuery(r *http.Request, dest interface{}) error {
	values := r.URL.Query()

	// Parse pagination for any request type
	parsePage := func() int {
		if v := values.Get("page"); v != "" {
			n := 0
			for _, c := range v {
				if c >= '0' && c <= '9' {
					n = n*10 + int(c-'0')
				}
			}
			if n > 0 {
				return n
			}
		}
		return 1
	}
	parsePageSize := func(defaultSize int) int {
		if v := values.Get("page_size"); v != "" {
			n := 0
			for _, c := range v {
				if c >= '0' && c <= '9' {
					n = n*10 + int(c-'0')
				}
			}
			if n > 0 {
				return n
			}
		}
		return defaultSize
	}

	switch d := dest.(type) {
	case *schema.ProductListRequest:
		d.Page = parsePage()
		d.PageSize = parsePageSize(24)
		if v := values.Get("category"); v != "" {
			d.Category = &v
		}
		if v := values.Get("brand"); v != "" {
			d.Brand = &v
		}
		if v := values.Get("handling_class"); v != "" {
			d.HandlingClass = &v
		}
		if v := values.Get("supplier"); v != "" {
			d.Supplier = &v
		}
		if v := values.Get("q"); v != "" {
			d.Query = &v
		}
		if v := values.Get("sort"); v != "" {
			d.Sort = &v
		}
	case *schema.CategoryListRequest:
		d.Page = parsePage()
		d.PageSize = parsePageSize(50)
		if v := values.Get("parent"); v != "" {
			d.Parent = &v
		}
	case *schema.BrandListRequest:
		d.Page = parsePage()
		d.PageSize = parsePageSize(50)
	}

	return nil
}

func (rt *Router) decodeBody(r *http.Request, dest interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(dest)
}

func (rt *Router) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (rt *Router) writeError(w http.ResponseWriter, status int, code, detail string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"detail": detail,
		"code":   code,
	})
}

func conversionFactorToFloat64(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f, _ := n.Float64Value()
	return f.Float64
}
