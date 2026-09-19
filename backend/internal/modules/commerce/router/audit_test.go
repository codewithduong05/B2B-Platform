package router_test

// TASK-004A audit tests: buyer isolation, concurrency invariants,
// product-availability gating, quantity overflow, delete/re-add,
// update-after-delete mapping, and 201 Location header.
//
// Each test targets a real finding from the cart/quote audit.
// They use the existing advisory-lock TestMain and helper pattern.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/atlas-platform/backend/internal/modules/commerce/schema"
	commerce_service "github.com/atlas-platform/backend/internal/modules/commerce/service"
)

// A1: buyer A cannot read, modify, delete, or quote buyer B's cart state.
func TestAudit_BuyerIsolation(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	buyerA := createTestBuyer(t, env)
	buyerB := createTestBuyer(t, env)
	prodCode, _, _ := createTestProduct(t, env)

	envA := setupEnv(t, buyerA, withPrincipalMiddleware(buyerA), nopMiddleware)
	envB := setupEnv(t, buyerB, withPrincipalMiddleware(buyerB), nopMiddleware)

	// A adds an item.
	resp, body := doJSON(t, envA, http.MethodPost, "/api/v1/commerce/cart/items",
		fmt.Sprintf(`{"product_code":"%s","quantity":2}`, prodCode), nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("A add item: expected 201, got %d: %s", resp.StatusCode, string(body))
	}
	var cartA schema.CartResponse
	if err := json.Unmarshal(body, &cartA); err != nil {
		t.Fatalf("decode A cart: %v", err)
	}
	if len(cartA.Suppliers) != 1 || len(cartA.Suppliers[0].Items) != 1 {
		t.Fatalf("expected A's line, got %+v", cartA)
	}
	lineCode := cartA.Suppliers[0].Items[0].Code

	// B's cart must not contain A's items.
	resp, body = doJSON(t, envB, http.MethodGet, "/api/v1/commerce/cart", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("B get cart: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	var cartB schema.CartResponse
	if err := json.Unmarshal(body, &cartB); err != nil {
		t.Fatalf("decode B cart: %v", err)
	}
	if len(cartB.Suppliers) != 0 {
		t.Errorf("buyer B sees buyer A's items: %+v", cartB)
	}

	// B cannot modify A's line.
	resp, body = doJSON(t, envB, http.MethodPatch,
		fmt.Sprintf("/api/v1/commerce/cart/items/%s", lineCode), `{"quantity":9}`, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("B patch A line: expected 404, got %d: %s", resp.StatusCode, string(body))
	}

	// B cannot delete A's line.
	resp, body = doJSON(t, envB, http.MethodDelete,
		fmt.Sprintf("/api/v1/commerce/cart/items/%s", lineCode), "", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("B delete A line: expected 404, got %d: %s", resp.StatusCode, string(body))
	}
	if code := decodeErr(t, body)["code"]; code != "cart_line_not_found" {
		t.Errorf("expected cart_line_not_found, got %q", code)
	}

	// B's quote must not include A's lines.
	resp, body = doJSON(t, envB, http.MethodPost, "/api/v1/commerce/cart/quote", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("B quote: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	var quoteB schema.CartQuoteResponse
	if err := json.Unmarshal(body, &quoteB); err != nil {
		t.Fatalf("decode B quote: %v", err)
	}
	if quoteB.TotalMinor != 0 || len(quoteB.Suppliers) != 0 {
		t.Errorf("buyer B quote includes buyer A state: %+v", quoteB)
	}

	// A's line is untouched by B's attempts.
	resp, body = doJSON(t, envA, http.MethodGet, "/api/v1/commerce/cart", "", nil)
	var cartA2 schema.CartResponse
	if err := json.Unmarshal(body, &cartA2); err != nil {
		t.Fatalf("decode A cart reread: %v", err)
	}
	if len(cartA2.Suppliers) != 1 || cartA2.Suppliers[0].Items[0].Quantity != 2 {
		t.Errorf("A's cart changed by B: %+v", cartA2)
	}
}

// A2: concurrent cart creation for one buyer yields exactly one cart row.
func TestAudit_ConcurrentCartCreation(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	buyerID := createTestBuyer(t, env)
	envAuth := setupEnv(t, buyerID, withPrincipalMiddleware(buyerID), nopMiddleware)

	const n = 20
	codes := make([]string, n)
	statuses := make([]int, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			resp, body := doJSON(t, envAuth, http.MethodGet, "/api/v1/commerce/cart", "", nil)
			statuses[i] = resp.StatusCode
			if resp.StatusCode != http.StatusOK {
				return
			}
			var cart schema.CartResponse
			if err := json.Unmarshal(body, &cart); err == nil {
				codes[i] = cart.Code
			}
		}(i)
	}
	wg.Wait()

	for i, st := range statuses {
		if st != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i, st)
		}
	}
	for i := 1; i < n; i++ {
		if codes[i] != codes[0] {
			t.Fatalf("concurrent callers got different carts: %q vs %q", codes[0], codes[i])
		}
	}

	var count int
	err := envAuth.db.Pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM commerce.cart WHERE buyer_id = $1 AND deleted_at IS NULL`, buyerID).Scan(&count)
	if err != nil {
		t.Fatalf("count carts: %v", err)
	}
	if count != 1 {
		t.Errorf("expected exactly 1 cart row for buyer, got %d", count)
	}
}

// A3: concurrent adds of the same product yield one line with summed quantity.
func TestAudit_ConcurrentSameProductAdd(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	buyerID := createTestBuyer(t, env)
	prodCode, _, _ := createTestProduct(t, env)
	envAuth := setupEnv(t, buyerID, withPrincipalMiddleware(buyerID), nopMiddleware)

	const n = 10
	statuses := make([]int, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			resp, _ := doJSON(t, envAuth, http.MethodPost, "/api/v1/commerce/cart/items",
				fmt.Sprintf(`{"product_code":"%s","quantity":1}`, prodCode), nil)
			statuses[i] = resp.StatusCode
		}(i)
	}
	wg.Wait()

	for i, st := range statuses {
		if st != http.StatusCreated {
			t.Fatalf("add %d: expected 201, got %d", i, st)
		}
	}

	resp, body := doJSON(t, envAuth, http.MethodGet, "/api/v1/commerce/cart", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get cart: %d: %s", resp.StatusCode, string(body))
	}
	var cart schema.CartResponse
	if err := json.Unmarshal(body, &cart); err != nil {
		t.Fatalf("decode cart: %v", err)
	}
	totalLines, totalQty := 0, 0
	for _, g := range cart.Suppliers {
		for _, it := range g.Items {
			totalLines++
			totalQty += it.Quantity
		}
	}
	if totalLines != 1 {
		t.Errorf("expected 1 logical cart line, got %d", totalLines)
	}
	if totalQty != n {
		t.Errorf("expected summed quantity %d, got %d", n, totalQty)
	}
}

// A4: delete then re-add of the same product must be visible again.
func TestAudit_DeleteThenReAdd(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	buyerID := createTestBuyer(t, env)
	prodCode, _, _ := createTestProduct(t, env)
	envAuth := setupEnv(t, buyerID, withPrincipalMiddleware(buyerID), nopMiddleware)

	resp, body := doJSON(t, envAuth, http.MethodPost, "/api/v1/commerce/cart/items",
		fmt.Sprintf(`{"product_code":"%s","quantity":2}`, prodCode), nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("add: %d: %s", resp.StatusCode, string(body))
	}
	var cart schema.CartResponse
	_ = json.Unmarshal(body, &cart)
	lineCode := cart.Suppliers[0].Items[0].Code

	resp, body = doJSON(t, envAuth, http.MethodDelete,
		fmt.Sprintf("/api/v1/commerce/cart/items/%s", lineCode), "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete: %d: %s", resp.StatusCode, string(body))
	}

	resp, body = doJSON(t, envAuth, http.MethodPost, "/api/v1/commerce/cart/items",
		fmt.Sprintf(`{"product_code":"%s","quantity":3}`, prodCode), nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("re-add: %d: %s", resp.StatusCode, string(body))
	}
	_ = json.Unmarshal(body, &cart)
	found := false
	for _, g := range cart.Suppliers {
		for _, it := range g.Items {
			if it.ProductCode == prodCode {
				found = true
				if it.Quantity != 3 {
					t.Errorf("expected re-added quantity 3, got %d", it.Quantity)
				}
			}
		}
	}
	if !found {
		t.Errorf("re-added product is invisible in cart (ghost line): %s", string(body))
	}
}

// A5: draft/archived/inactive products must not be addable (404, not 201).
func TestAudit_UnpublishedProductRejected(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	buyerID := createTestBuyer(t, env)
	envAuth := setupEnv(t, buyerID, withPrincipalMiddleware(buyerID), nopMiddleware)
	ctx := context.Background()

	cases := []struct {
		name string
		set  string
	}{
		{"draft", "status = 'draft'"},
		{"archived", "status = 'archived'"},
		{"inactive", "is_active = false"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prodCode, prodID, _ := createTestProduct(t, env)
			if _, err := env.db.Pool.Exec(ctx,
				fmt.Sprintf(`UPDATE catalog.product SET %s WHERE id = $1`, tc.set), prodID); err != nil {
				t.Fatalf("flip product state: %v", err)
			}

			resp, body := doJSON(t, envAuth, http.MethodPost, "/api/v1/commerce/cart/items",
				fmt.Sprintf(`{"product_code":"%s","quantity":1}`, prodCode), nil)
			if resp.StatusCode != http.StatusNotFound {
				t.Errorf("expected 404 for %s product, got %d: %s", tc.name, resp.StatusCode, string(body))
			}
			if code := decodeErr(t, body)["code"]; code != "product_not_found" {
				t.Errorf("expected product_not_found, got %q", code)
			}
		})
	}
}

// A6: product without a supplier reference must not 500.
func TestAudit_SupplierlessProductRejected(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	buyerID := createTestBuyer(t, env)
	prodCode, prodID, _ := createTestProduct(t, env)
	envAuth := setupEnv(t, buyerID, withPrincipalMiddleware(buyerID), nopMiddleware)

	if _, err := env.db.Pool.Exec(context.Background(),
		`UPDATE catalog.product SET supplier_id = NULL WHERE id = $1`, prodID); err != nil {
		t.Fatalf("clear supplier: %v", err)
	}

	resp, body := doJSON(t, envAuth, http.MethodPost, "/api/v1/commerce/cart/items",
		fmt.Sprintf(`{"product_code":"%s","quantity":1}`, prodCode), nil)
	if resp.StatusCode == http.StatusInternalServerError {
		t.Errorf("supplier-less product caused 500: %s", string(body))
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for supplier-less product, got %d: %s", resp.StatusCode, string(body))
	}
}

// A7: excessively large quantity is rejected with 400, never 500.
func TestAudit_QuantityOverflowRejected(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	buyerID := createTestBuyer(t, env)
	prodCode, _, _ := createTestProduct(t, env)
	envAuth := setupEnv(t, buyerID, withPrincipalMiddleware(buyerID), nopMiddleware)

	// 2^40 exceeds the commerce.cart_line.quantity INT column range.
	resp, body := doJSON(t, envAuth, http.MethodPost, "/api/v1/commerce/cart/items",
		fmt.Sprintf(`{"product_code":"%s","quantity":1099511627776}`, prodCode), nil)
	if resp.StatusCode == http.StatusInternalServerError {
		t.Fatalf("overflow quantity caused 500: %s", string(body))
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for overflow quantity, got %d: %s", resp.StatusCode, string(body))
	}
	if code := decodeErr(t, body)["code"]; code != "invalid_quantity" {
		t.Errorf("expected invalid_quantity, got %q", code)
	}
}

// A8: updating a line deleted out-of-band maps to 404, never 500.
func TestAudit_UpdateAfterDeleteMapsToNotFound(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	buyerID := createTestBuyer(t, env)
	prodCode, _, _ := createTestProduct(t, env)
	envAuth := setupEnv(t, buyerID, withPrincipalMiddleware(buyerID), nopMiddleware)
	ctx := context.Background()

	resp, body := doJSON(t, envAuth, http.MethodPost, "/api/v1/commerce/cart/items",
		fmt.Sprintf(`{"product_code":"%s","quantity":2}`, prodCode), nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("add: %d: %s", resp.StatusCode, string(body))
	}
	var cart schema.CartResponse
	_ = json.Unmarshal(body, &cart)
	lineCode := cart.Suppliers[0].Items[0].Code

	// Delete out-of-band so the service update hits a deleted row.
	if _, err := env.db.Pool.Exec(ctx,
		`UPDATE commerce.cart_line SET deleted_at = NOW() WHERE code = $1`, lineCode); err != nil {
		t.Fatalf("out-of-band delete: %v", err)
	}

	svc := commerce_service.NewCommerceService(envAuth.db, nil, nil, nil, nil)
	_, err := svc.UpdateCartItem(ctx, buyerID, lineCode, schema.UpdateCartItemRequest{Quantity: 5})
	if err == nil {
		t.Fatalf("expected error updating a deleted line, got nil")
	}
	// Must be the domain not-found error, not a raw pgx no-rows error.
	if err != commerce_service.ErrCartLineNotFound {
		t.Errorf("expected ErrCartLineNotFound, got %v", err)
	}
}

// A9: creating a line returns 201 with a Location header (contract §Status code usage).
func TestAudit_CreateLineSetsLocation(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	buyerID := createTestBuyer(t, env)
	prodCode, _, _ := createTestProduct(t, env)
	envAuth := setupEnv(t, buyerID, withPrincipalMiddleware(buyerID), nopMiddleware)

	resp, body := doJSON(t, envAuth, http.MethodPost, "/api/v1/commerce/cart/items",
		fmt.Sprintf(`{"product_code":"%s","quantity":1}`, prodCode), nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("add: %d: %s", resp.StatusCode, string(body))
	}
	if loc := resp.Header.Get("Location"); loc == "" {
		t.Errorf("expected Location header on 201, body: %s", string(body))
	}
}
