package router_test

// TASK-005 checkout tests: basic + multi-supplier split, pricing snapshot,
// inventory matrix, idempotency, cart protection, isolation, errors,
// concurrency. Uses the existing advisory-lock TestMain and helpers.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/atlas-platform/backend/internal/modules/commerce/schema"
	commerce_service "github.com/atlas-platform/backend/internal/modules/commerce/service"
	inventory_service "github.com/atlas-platform/backend/internal/modules/inventory/service"
	pricing_service "github.com/atlas-platform/backend/internal/modules/pricing/service"
)

func addStock(t *testing.T, env *testEnv, productID, supplierID int64, qty int32, quarantined bool) {
	t.Helper()
	ctx := context.Background()
	suffix := uniqueSuffix()
	var stockLevelID int64
	if err := env.db.Pool.QueryRow(ctx, `
		INSERT INTO inventory.stock_level (code, product_id, supplier_id, available_quantity, total_quantity)
		VALUES ($1, $2, $3, $4, $4)
		ON CONFLICT (product_id, supplier_id) DO UPDATE
		SET available_quantity = inventory.stock_level.available_quantity + $4,
		    total_quantity = inventory.stock_level.total_quantity + $4
		RETURNING id
	`, "sl_"+suffix, productID, supplierID, qty).Scan(&stockLevelID); err != nil {
		t.Fatalf("upsert stock level: %v", err)
	}
	status := "active"
	avail := qty
	if quarantined {
		status = "quarantined"
		avail = 0
	}
	if _, err := env.db.Pool.Exec(ctx, `
		INSERT INTO inventory.lot (code, stock_level_id, lot_number, initial_quantity, available_quantity, status, is_quarantined, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, "lot_"+suffix, stockLevelID, "LOT-"+suffix, qty, avail, status,
		quarantined, time.Now().Add(30*24*time.Hour)); err != nil {
		t.Fatalf("create lot: %v", err)
	}
}

func addItem(t *testing.T, env *testEnv, prodCode string, qty int) {
	t.Helper()
	resp, body := doJSON(t, env, http.MethodPost, "/api/v1/commerce/cart/items",
		fmt.Sprintf(`{"product_code":"%s","quantity":%d}`, prodCode, qty), nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("add item %s x%d: expected 201, got %d: %s", prodCode, qty, resp.StatusCode, string(body))
	}
}

func doCheckout(t *testing.T, env *testEnv, key, body string) (*http.Response, []byte) {
	t.Helper()
	headers := map[string]string{}
	if key != "" {
		headers["Idempotency-Key"] = key
	}
	return doJSON(t, env, http.MethodPost, "/api/v1/commerce/checkout", body, headers)
}

func decodeCheckout(t *testing.T, body []byte) schema.CheckoutResponse {
	t.Helper()
	var resp schema.CheckoutResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode checkout response: %v (%s)", err, string(body))
	}
	return resp
}

func countRows(t *testing.T, env *testEnv, query string, args ...interface{}) int {
	t.Helper()
	var n int
	if err := env.db.Pool.QueryRow(context.Background(), query, args...).Scan(&n); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	return n
}

// countReservedForProduct counts active reservations for one product only.
// Tests share one database, so global reservation counts are meaningless.
func countReservedForProduct(t *testing.T, env *testEnv, productID int64) (rows, units int) {
	t.Helper()
	err := env.db.Pool.QueryRow(context.Background(), `
		SELECT COUNT(*), COALESCE(SUM(r.quantity), 0)
		FROM inventory.reservation r
		JOIN inventory.lot l ON l.id = r.lot_id
		JOIN inventory.stock_level sl ON sl.id = l.stock_level_id
		WHERE sl.product_id = $1 AND r.status = 'reserved' AND r.deleted_at IS NULL
	`, productID).Scan(&rows, &units)
	if err != nil {
		t.Fatalf("count reservations: %v", err)
	}
	return rows, units
}

func TestCheckout_Basic(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	buyerID := createTestBuyer(t, env)
	prodCode, productID, supplierID := createTestProduct(t, env)
	addStock(t, env, productID, supplierID, 10, false)
	envAuth := setupEnv(t, buyerID, withPrincipalMiddleware(buyerID), nopMiddleware)

	addItem(t, envAuth, prodCode, 2)

	resp, body := doCheckout(t, envAuth, "basic-key-1", "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("checkout: expected 201, got %d: %s", resp.StatusCode, string(body))
	}
	if loc := resp.Header.Get("Location"); loc == "" {
		t.Errorf("expected Location header on 201")
	}
	out := decodeCheckout(t, body)
	if len(out.Orders) != 1 {
		t.Fatalf("expected 1 order, got %d: %s", len(out.Orders), string(body))
	}
	ord := out.Orders[0]
	if ord.Status != "placed" {
		t.Errorf("expected status placed, got %q", ord.Status)
	}
	if len(ord.Lines) != 1 || ord.Lines[0].Quantity != 2 {
		t.Errorf("unexpected lines: %+v", ord.Lines)
	}
	if ord.Lines[0].UnitPriceMinor != 1500 || ord.Lines[0].TotalPriceMinor != 3000 {
		t.Errorf("expected 1500/3000 pricing, got %+v", ord.Lines[0])
	}
	if ord.TotalMinor != 3000 {
		t.Errorf("expected order total 3000, got %d", ord.TotalMinor)
	}

	// Inventory reserved exactly 2.
	if n := countRows(t, envAuth, `SELECT COALESCE(SUM(available_quantity),0) FROM inventory.stock_level WHERE product_id=$1`, productID); n != 8 {
		t.Errorf("expected available 8 after reservation, got %d", n)
	}
	if n := countRows(t, envAuth, `SELECT COALESCE(SUM(reserved_quantity),0) FROM inventory.stock_level WHERE product_id=$1`, productID); n != 2 {
		t.Errorf("expected reserved 2 after reservation, got %d", n)
	}

	// Cart consumed.
	resp, body = doJSON(t, envAuth, http.MethodGet, "/api/v1/commerce/cart", "", nil)
	var cart schema.CartResponse
	_ = json.Unmarshal(body, &cart)
	if len(cart.Suppliers) != 0 {
		t.Errorf("expected cleared cart, got %+v", cart)
	}

	// Order readable with history.
	resp, body = doJSON(t, envAuth, http.MethodGet, "/api/v1/commerce/orders/me/"+ord.Code, "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get order: %d: %s", resp.StatusCode, string(body))
	}
	var detail schema.OrderDetailResponse
	if err := json.Unmarshal(body, &detail); err != nil {
		t.Fatalf("decode order detail: %v", err)
	}
	if len(detail.History) != 1 || detail.History[0].ToStatus != "placed" {
		t.Errorf("expected single placed history row, got %+v", detail.History)
	}
	if detail.History[0].Actor == nil || *detail.History[0].Actor != buyerID {
		t.Errorf("expected history actor %d, got %+v", buyerID, detail.History[0])
	}
}

func TestCheckout_MultiSupplierSplit(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	buyerID := createTestBuyer(t, env)
	var codes []string
	var totals int64
	for i := 0; i < 3; i++ {
		prodCode, productID, supplierID := createTestProduct(t, env)
		addStock(t, env, productID, supplierID, 20, false)
		codes = append(codes, prodCode)
		totals += 1500
	}
	envAuth := setupEnv(t, buyerID, withPrincipalMiddleware(buyerID), nopMiddleware)
	for _, c := range codes {
		addItem(t, envAuth, c, 1)
	}

	resp, body := doCheckout(t, envAuth, "split-key-1", "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("checkout: %d: %s", resp.StatusCode, string(body))
	}
	out := decodeCheckout(t, body)
	if len(out.Orders) != 3 {
		t.Fatalf("expected 3 orders (one per supplier), got %d", len(out.Orders))
	}
	seenSuppliers := map[string]bool{}
	var grandTotal int64
	for _, o := range out.Orders {
		if len(o.Lines) != 1 {
			t.Errorf("expected 1 line per split order, got %+v", o)
		}
		if seenSuppliers[o.SupplierCode] {
			t.Errorf("duplicate supplier order: %q", o.SupplierCode)
		}
		seenSuppliers[o.SupplierCode] = true
		grandTotal += o.TotalMinor
	}
	if grandTotal != totals {
		t.Errorf("expected grand total %d, got %d", totals, grandTotal)
	}

	// Buyer history lists all three.
	resp, body = doJSON(t, envAuth, http.MethodGet, "/api/v1/commerce/orders/me", "", nil)
	var list []schema.OrderResponse
	_ = json.Unmarshal(body, &list)
	if len(list) != 3 {
		t.Errorf("expected 3 orders in history, got %d", len(list))
	}
}

func TestCheckout_IdempotentReplay(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	buyerID := createTestBuyer(t, env)
	prodCode, productID, supplierID := createTestProduct(t, env)
	addStock(t, env, productID, supplierID, 10, false)
	envAuth := setupEnv(t, buyerID, withPrincipalMiddleware(buyerID), nopMiddleware)
	addItem(t, envAuth, prodCode, 2)

	resp, body := doCheckout(t, envAuth, "replay-key-1", "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("first checkout: %d: %s", resp.StatusCode, string(body))
	}
	first := decodeCheckout(t, body)

	// Same key, cart now empty → 200 replay of the identical result.
	resp, body = doCheckout(t, envAuth, "replay-key-1", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("replay: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	second := decodeCheckout(t, body)
	if !second.Replayed {
		t.Errorf("expected replayed=true on retry")
	}
	if len(first.Orders) != len(second.Orders) || first.Orders[0].Code != second.Orders[0].Code {
		t.Errorf("replay returned different orders: %+v vs %+v", first.Orders, second.Orders)
	}

	// No duplicate orders or reservations.
	if n := countRows(t, envAuth, `SELECT COUNT(*) FROM commerce."order" WHERE buyer_id=$1 AND deleted_at IS NULL`, buyerID); n != 1 {
		t.Errorf("expected 1 order after replay, got %d", n)
	}
	if rows, _ := countReservedForProduct(t, envAuth, productID); rows != 1 {
		t.Errorf("expected 1 reservation after replay, got %d", rows)
	}
}

func TestCheckout_SameKeyDifferentPayload(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	buyerID := createTestBuyer(t, env)
	prodA, idA, supA := createTestProduct(t, env)
	prodB, idB, supB := createTestProduct(t, env)
	addStock(t, env, idA, supA, 10, false)
	addStock(t, env, idB, supB, 10, false)
	envAuth := setupEnv(t, buyerID, withPrincipalMiddleware(buyerID), nopMiddleware)

	addItem(t, envAuth, prodA, 1)
	resp, body := doCheckout(t, envAuth, "conflict-key-1", "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("first checkout: %d: %s", resp.StatusCode, string(body))
	}

	// Cart changed under the same key → 409, no new order set.
	addItem(t, envAuth, prodB, 1)
	resp, body = doCheckout(t, envAuth, "conflict-key-1", "")
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 for changed payload, got %d: %s", resp.StatusCode, string(body))
	}
	if code := decodeErr(t, body)["code"]; code != "idempotency_conflict" {
		t.Errorf("expected idempotency_conflict, got %q", code)
	}
	if n := countRows(t, envAuth, `SELECT COUNT(*) FROM commerce."order" WHERE buyer_id=$1 AND deleted_at IS NULL`, buyerID); n != 1 {
		t.Errorf("expected still 1 order, got %d", n)
	}
}

func TestCheckout_InsufficientStock(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	buyerID := createTestBuyer(t, env)
	prodCode, productID, supplierID := createTestProduct(t, env)
	addStock(t, env, productID, supplierID, 2, false)
	envAuth := setupEnv(t, buyerID, withPrincipalMiddleware(buyerID), nopMiddleware)
	addItem(t, envAuth, prodCode, 5)

	resp, body := doCheckout(t, envAuth, "short-key-1", "")
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", resp.StatusCode, string(body))
	}
	if code := decodeErr(t, body)["code"]; code != "insufficient_stock" {
		t.Errorf("expected insufficient_stock, got %q", code)
	}
	// No order set, no leftover reservation.
	if n := countRows(t, envAuth, `SELECT COUNT(*) FROM commerce."order" WHERE buyer_id=$1`, buyerID); n != 0 {
		t.Errorf("expected 0 orders, got %d", n)
	}
	if n := countRows(t, envAuth, `SELECT COALESCE(SUM(reserved_quantity),0) FROM inventory.stock_level WHERE product_id=$1`, productID); n != 0 {
		t.Errorf("expected 0 reserved stock, got %d", n)
	}
	if n := countRows(t, envAuth, `SELECT COALESCE(SUM(available_quantity),0) FROM inventory.stock_level WHERE product_id=$1`, productID); n != 2 {
		t.Errorf("expected available restored to 2, got %d", n)
	}
}

func TestCheckout_PartialSupplierFailureReleases(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	buyerID := createTestBuyer(t, env)
	prodA, idA, supA := createTestProduct(t, env)
	prodB, idB, supB := createTestProduct(t, env)
	addStock(t, env, idA, supA, 10, false)
	addStock(t, env, idB, supB, 1, false) // too little for qty 4
	envAuth := setupEnv(t, buyerID, withPrincipalMiddleware(buyerID), nopMiddleware)
	addItem(t, envAuth, prodA, 3)
	addItem(t, envAuth, prodB, 4)

	resp, body := doCheckout(t, envAuth, "partial-key-1", "")
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", resp.StatusCode, string(body))
	}
	if n := countRows(t, envAuth, `SELECT COUNT(*) FROM commerce."order" WHERE buyer_id=$1`, buyerID); n != 0 {
		t.Errorf("no partial order set allowed, found %d orders", n)
	}
	// Supplier A's reservation must have been rolled back.
	if n := countRows(t, envAuth, `SELECT COALESCE(SUM(reserved_quantity),0) FROM inventory.stock_level WHERE product_id=$1`, idA); n != 0 {
		t.Errorf("expected A's reservation released, reserved=%d", n)
	}
	if rows, _ := countReservedForProduct(t, envAuth, idA); rows != 0 {
		t.Errorf("expected no lingering A reservations, got %d", rows)
	}
	if rows, _ := countReservedForProduct(t, envAuth, idB); rows != 0 {
		t.Errorf("expected no lingering B reservations, got %d", rows)
	}
}

func TestCheckout_Errors(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	buyerID := createTestBuyer(t, env)
	envAuth := setupEnv(t, buyerID, withPrincipalMiddleware(buyerID), nopMiddleware)

	// Missing key → 400.
	resp, body := doCheckout(t, envAuth, "", "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("missing key: expected 400, got %d: %s", resp.StatusCode, string(body))
	}

	// Empty cart → 422.
	resp, body = doCheckout(t, envAuth, "empty-key-1", "")
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("empty cart: expected 422, got %d: %s", resp.StatusCode, string(body))
	}
	if code := decodeErr(t, body)["code"]; code != "empty_cart" {
		t.Errorf("expected empty_cart, got %q", code)
	}

	// Unavailable (archived) product → 404, no order.
	prodCode, _, _ := createTestProduct(t, env)
	addItem(t, envAuth, prodCode, 1)
	if _, err := env.db.Pool.Exec(context.Background(),
		`UPDATE catalog.product SET status='archived' WHERE code=$1`, prodCode); err != nil {
		t.Fatalf("archive product: %v", err)
	}
	resp, body = doCheckout(t, envAuth, "gone-key-1", "")
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("archived product: expected 404, got %d: %s", resp.StatusCode, string(body))
	}
	if n := countRows(t, envAuth, `SELECT COUNT(*) FROM commerce."order" WHERE buyer_id=$1`, buyerID); n != 0 {
		t.Errorf("expected 0 orders, got %d", n)
	}
}

func TestCheckout_BuyerIsolation(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	buyerA := createTestBuyer(t, env)
	buyerB := createTestBuyer(t, env)
	prodCode, productID, supplierID := createTestProduct(t, env)
	addStock(t, env, productID, supplierID, 10, false)
	envA := setupEnv(t, buyerA, withPrincipalMiddleware(buyerA), nopMiddleware)
	envB := setupEnv(t, buyerB, withPrincipalMiddleware(buyerB), nopMiddleware)

	addItem(t, envA, prodCode, 1)
	resp, body := doCheckout(t, envA, "iso-key-1", "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("A checkout: %d: %s", resp.StatusCode, string(body))
	}
	orderCode := decodeCheckout(t, body).Orders[0].Code

	// B sees no orders and cannot read A's order.
	resp, body = doJSON(t, envB, http.MethodGet, "/api/v1/commerce/orders/me", "", nil)
	var list []schema.OrderResponse
	_ = json.Unmarshal(body, &list)
	if len(list) != 0 {
		t.Errorf("B sees A's orders: %+v", list)
	}
	resp, body = doJSON(t, envB, http.MethodGet, "/api/v1/commerce/orders/me/"+orderCode, "", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("B read A order: expected 404, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestCheckout_NoDoubleCheckoutNewKey(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	buyerID := createTestBuyer(t, env)
	prodCode, productID, supplierID := createTestProduct(t, env)
	addStock(t, env, productID, supplierID, 10, false)
	envAuth := setupEnv(t, buyerID, withPrincipalMiddleware(buyerID), nopMiddleware)
	addItem(t, envAuth, prodCode, 1)

	resp, body := doCheckout(t, envAuth, "once-key-1", "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("first: %d: %s", resp.StatusCode, string(body))
	}
	// Same cart, different key → 422, no second order set.
	resp, body = doCheckout(t, envAuth, "once-key-2", "")
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("second key: expected 422, got %d: %s", resp.StatusCode, string(body))
	}
	if n := countRows(t, envAuth, `SELECT COUNT(*) FROM commerce."order" WHERE buyer_id=$1`, buyerID); n != 1 {
		t.Errorf("expected 1 order, got %d", n)
	}
}

func TestCheckout_ConcurrentSameKey(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	buyerID := createTestBuyer(t, env)
	prodCode, productID, supplierID := createTestProduct(t, env)
	addStock(t, env, productID, supplierID, 10, false)
	envAuth := setupEnv(t, buyerID, withPrincipalMiddleware(buyerID), nopMiddleware)
	addItem(t, envAuth, prodCode, 2)

	const n = 10
	var statuses [n]int
	var bodies [n]string
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			resp, body := doCheckout(t, envAuth, "conc-same-key", "")
			statuses[i] = resp.StatusCode
			bodies[i] = string(body)
		}(i)
	}
	wg.Wait()

	created, okCount := 0, 0
	firstCode := ""
	for i := 0; i < n; i++ {
		switch statuses[i] {
		case http.StatusCreated:
			created++
			out := decodeCheckout(t, []byte(bodies[i]))
			if firstCode == "" {
				firstCode = out.Orders[0].Code
			} else if out.Orders[0].Code != firstCode {
				t.Errorf("concurrent same-key created different orders: %q vs %q", firstCode, out.Orders[0].Code)
			}
		case http.StatusOK, http.StatusConflict:
			okCount++
		default:
			t.Errorf("request %d: unexpected status %d: %s", i, statuses[i], bodies[i])
		}
	}
	if created != 1 {
		t.Errorf("expected exactly 1 created order set, got %d", created)
	}
	if n := countRows(t, envAuth, `SELECT COUNT(*) FROM commerce."order" WHERE buyer_id=$1`, buyerID); n != 1 {
		t.Errorf("expected 1 order row, got %d", n)
	}
}

func TestCheckout_ConcurrentDifferentKeys(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	buyerID := createTestBuyer(t, env)
	prodCode, productID, supplierID := createTestProduct(t, env)
	addStock(t, env, productID, supplierID, 10, false)
	envAuth := setupEnv(t, buyerID, withPrincipalMiddleware(buyerID), nopMiddleware)
	addItem(t, envAuth, prodCode, 1)

	const n = 5
	var statuses [n]int
	var bodies [n]string
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			resp, body := doCheckout(t, envAuth, fmt.Sprintf("conc-diff-key-%d", i), "")
			statuses[i] = resp.StatusCode
			bodies[i] = string(body)
		}(i)
	}
	wg.Wait()

	created := 0
	for i, st := range statuses {
		if st == http.StatusCreated {
			created++
		} else if st != http.StatusUnprocessableEntity && st != http.StatusConflict {
			t.Errorf("request %d: unexpected status %d: %s", i, st, bodies[i])
		}
	}
	if created != 1 {
		t.Errorf("expected exactly 1 successful checkout, got %d", created)
	}
	if c := countRows(t, envAuth, `SELECT COUNT(*) FROM commerce."order" WHERE buyer_id=$1`, buyerID); c != 1 {
		t.Errorf("expected 1 order, got %d", c)
	}
}

func TestCheckout_InventoryContention(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	buyerA := createTestBuyer(t, env)
	buyerB := createTestBuyer(t, env)
	prodCode, productID, supplierID := createTestProduct(t, env)
	addStock(t, env, productID, supplierID, 3, false)
	envA := setupEnv(t, buyerA, withPrincipalMiddleware(buyerA), nopMiddleware)
	envB := setupEnv(t, buyerB, withPrincipalMiddleware(buyerB), nopMiddleware)
	addItem(t, envA, prodCode, 3)
	addItem(t, envB, prodCode, 3)

	var stA, stB int
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); r, _ := doCheckout(t, envA, "cont-key-a", ""); stA = r.StatusCode }()
	go func() { defer wg.Done(); r, _ := doCheckout(t, envB, "cont-key-b", ""); stB = r.StatusCode }()
	wg.Wait()

	successes := 0
	for _, st := range []int{stA, stB} {
		if st == http.StatusCreated {
			successes++
		} else if st != http.StatusUnprocessableEntity {
			t.Errorf("unexpected status %d", st)
		}
	}
	if successes != 1 {
		t.Errorf("expected exactly 1 winner for 3 units, got %d (%d/%d)", successes, stA, stB)
	}
	// No overselling: reserved across the product never exceeds 3.
	if n := countRows(t, env, `SELECT COALESCE(SUM(reserved_quantity),0) FROM inventory.stock_level WHERE product_id=$1`, productID); n > 3 {
		t.Errorf("oversold: reserved=%d for 3 units", n)
	}
	if n := countRows(t, env, `SELECT COALESCE(SUM(available_quantity),0) FROM inventory.stock_level WHERE product_id=$1`, productID); n < 0 {
		t.Errorf("negative availability: %d", n)
	}
}

func TestCheckout_QuarantinedLotExcluded(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	buyerID := createTestBuyer(t, env)
	prodCode, productID, supplierID := createTestProduct(t, env)
	addStock(t, env, productID, supplierID, 10, true) // quarantined only
	envAuth := setupEnv(t, buyerID, withPrincipalMiddleware(buyerID), nopMiddleware)
	addItem(t, envAuth, prodCode, 2)

	resp, body := doCheckout(t, envAuth, "quar-key-1", "")
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for quarantined-only stock, got %d: %s", resp.StatusCode, string(body))
	}
	if n := countRows(t, envAuth, `SELECT COUNT(*) FROM commerce."order" WHERE buyer_id=$1`, buyerID); n != 0 {
		t.Errorf("expected 0 orders, got %d", n)
	}
}

func TestCheckout_MultiLotReservation(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	buyerID := createTestBuyer(t, env)
	prodCode, productID, supplierID := createTestProduct(t, env)
	addStock(t, env, productID, supplierID, 2, false)
	addStock(t, env, productID, supplierID, 3, false)
	envAuth := setupEnv(t, buyerID, withPrincipalMiddleware(buyerID), nopMiddleware)
	addItem(t, envAuth, prodCode, 4)

	resp, body := doCheckout(t, envAuth, "multilot-key-1", "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("checkout: %d: %s", resp.StatusCode, string(body))
	}
	// Reservation must span both lots (FEFO, no partial).
	if rows, _ := countReservedForProduct(t, envAuth, productID); rows != 2 {
		t.Errorf("expected 2 lot allocations, got %d", rows)
	}
	if _, units := countReservedForProduct(t, envAuth, productID); units != 4 {
		t.Errorf("expected 4 reserved units, got %d", units)
	}
}

func TestCheckout_PriceSnapshotStable(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	buyerID := createTestBuyer(t, env)
	prodCode, productID, supplierID := createTestProduct(t, env)
	addStock(t, env, productID, supplierID, 10, false)
	envAuth := setupEnv(t, buyerID, withPrincipalMiddleware(buyerID), nopMiddleware)
	addItem(t, envAuth, prodCode, 2)

	resp, body := doCheckout(t, envAuth, "snap-key-1", "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("checkout: %d: %s", resp.StatusCode, string(body))
	}
	orderCode := decodeCheckout(t, body).Orders[0].Code

	// Change the live product price afterwards.
	if _, err := env.db.Pool.Exec(context.Background(),
		`UPDATE catalog.product SET base_price_minor=9999 WHERE id=$1`, productID); err != nil {
		t.Fatalf("bump price: %v", err)
	}

	resp, body = doJSON(t, envAuth, http.MethodGet, "/api/v1/commerce/orders/me/"+orderCode, "", nil)
	var detail schema.OrderDetailResponse
	_ = json.Unmarshal(body, &detail)
	if detail.Lines[0].UnitPriceMinor != 1500 || detail.TotalMinor != 3000 {
		t.Errorf("order snapshot changed with live price: %+v", detail)
	}
}

type recordingCheckoutPublisher struct {
	mu   sync.Mutex
	keys []string
}

func (p *recordingCheckoutPublisher) Publish(ctx context.Context, routingKey string, payload interface{}) error {
	p.mu.Lock()
	p.keys = append(p.keys, routingKey)
	p.mu.Unlock()
	return nil
}

func (p *recordingCheckoutPublisher) count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.keys)
}

func TestCheckout_EventPublished(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	buyerID := createTestBuyer(t, env)
	prodCode, productID, supplierID := createTestProduct(t, env)
	addStock(t, env, productID, supplierID, 10, false)

	pub := &recordingCheckoutPublisher{}
	svc := commerce_service.NewCommerceService(env.db,
		pricing_service.NewServices(env.db).PriceList,
		inventory_service.NewInventoryService(env.db, nil),
		pub)
	ctx := context.Background()

	// Seed the cart through the service (same path as the router).
	if _, err := svc.AddCartItem(ctx, buyerID, schema.AddCartItemRequest{ProductCode: prodCode, Quantity: 2}); err != nil {
		t.Fatalf("add item: %v", err)
	}
	result, err := svc.Checkout(ctx, buyerID, "evt-key-1")
	if err != nil {
		t.Fatalf("checkout: %v", err)
	}
	if got := pub.count(); got != len(result.Response.Orders) || got != 1 {
		t.Errorf("expected 1 order.placed event, got %d", got)
	}

	// Replay must not publish again.
	if _, err := svc.Checkout(ctx, buyerID, "evt-key-1"); err != nil {
		t.Fatalf("replay: %v", err)
	}
	if got := pub.count(); got != 1 {
		t.Errorf("replay published duplicate events: %d", got)
	}
}
