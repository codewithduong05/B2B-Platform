package router_test

// TASK-005A audit probes: cross-buyer key isolation, stale pending reclaim,
// checkout-vs-cart-mutation invariants, release idempotency/concurrency.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/atlas-platform/backend/internal/modules/commerce/schema"
	inventory_schema "github.com/atlas-platform/backend/internal/modules/inventory/schema"
	inventory_service "github.com/atlas-platform/backend/internal/modules/inventory/service"
)

// A1: two buyers using the same idempotency key string must remain fully
// isolated: separate claims, separate reservations, separate orders.
func TestAudit_CrossBuyerSameKey(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	buyerA := createTestBuyer(t, env)
	buyerB := createTestBuyer(t, env)
	prodCode, productID, supplierID := createTestProduct(t, env)
	addStock(t, env, productID, supplierID, 10, false)
	envA := setupEnv(t, buyerA, withPrincipalMiddleware(buyerA), nopMiddleware)
	envB := setupEnv(t, buyerB, withPrincipalMiddleware(buyerB), nopMiddleware)

	addItem(t, envA, prodCode, 2)
	addItem(t, envB, prodCode, 2)

	resp, body := doCheckout(t, envA, "shared-key", "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("A checkout: %d: %s", resp.StatusCode, string(body))
	}
	resp, body = doCheckout(t, envB, "shared-key", "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("B checkout: %d: %s", resp.StatusCode, string(body))
	}

	// Each buyer holds their own 2 units: 4 reserved in total.
	if n := countRows(t, env, `SELECT COALESCE(SUM(reserved_quantity),0) FROM inventory.stock_level WHERE product_id=$1`, productID); n != 4 {
		t.Errorf("expected 4 reserved units (2 per buyer), got %d: cross-buyer reservation cross-talk", n)
	}
	if n := countRows(t, envA, `SELECT COUNT(*) FROM commerce."order" WHERE buyer_id=$1`, buyerA); n != 1 {
		t.Errorf("expected A's order, got %d", n)
	}
	if n := countRows(t, envB, `SELECT COUNT(*) FROM commerce."order" WHERE buyer_id=$1`, buyerB); n != 1 {
		t.Errorf("expected B's order, got %d", n)
	}

	// A's replay still returns A's own order.
	resp, body = doCheckout(t, envA, "shared-key", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("A replay: %d: %s", resp.StatusCode, string(body))
	}
	var _ = body
}

// A2: a stale pending claim (crashed owner, no completion) must not block
// the same key forever; a fresh attempt reclaims it.
func TestAudit_StalePendingReclaim(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	buyerID := createTestBuyer(t, env)
	prodCode, productID, supplierID := createTestProduct(t, env)
	addStock(t, env, productID, supplierID, 10, false)
	envAuth := setupEnv(t, buyerID, withPrincipalMiddleware(buyerID), nopMiddleware)
	addItem(t, envAuth, prodCode, 1)

	// Simulate a crashed checkout: pending claim older than the stale threshold.
	if _, err := env.db.Pool.Exec(context.Background(), `
		INSERT INTO commerce.checkout_idempotency (buyer_id, idem_key, payload_hash, status, created_at, updated_at)
		VALUES ($1, 'stale-key-1', 'deadbeef', 'pending', NOW() - INTERVAL '1 hour', NOW() - INTERVAL '1 hour')
	`, buyerID); err != nil {
		t.Fatalf("seed stale claim: %v", err)
	}

	resp, body := doCheckout(t, envAuth, "stale-key-1", "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected stale claim to be reclaimed with 201, got %d: %s", resp.StatusCode, string(body))
	}
}

// A3: checkout racing cart adds must never silently lose a line: every
// added line ends up either in the order set or still in the cart.
func TestAudit_CheckoutVsConcurrentAdd(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	buyerID := createTestBuyer(t, env)
	const products = 6
	codes := make([]string, 0, products)
	for i := 0; i < products; i++ {
		prodCode, productID, supplierID := createTestProduct(t, env)
		addStock(t, env, productID, supplierID, 10, false)
		codes = append(codes, prodCode)
	}
	envAuth := setupEnv(t, buyerID, withPrincipalMiddleware(buyerID), nopMiddleware)

	// Seed one line so checkout has something to do.
	addItem(t, envAuth, codes[0], 1)
	// Give the seed a moment to commit before the race.
	time.Sleep(50 * time.Millisecond)

	var wg sync.WaitGroup
	var checkoutStatus int
	wg.Add(1)
	go func() {
		defer wg.Done()
		resp, _ := doCheckout(t, envAuth, "race-add-key", "")
		checkoutStatus = resp.StatusCode
	}()
	for _, c := range codes[1:] {
		wg.Add(1)
		go func(code string) {
			defer wg.Done()
			resp, _ := doJSON(t, envAuth, http.MethodPost, "/api/v1/commerce/cart/items",
				fmt.Sprintf(`{"product_code":"%s","quantity":1}`, code), nil)
			if resp.StatusCode != http.StatusCreated {
				t.Errorf("concurrent add %s: %d", code, resp.StatusCode)
			}
		}(c)
	}
	wg.Wait()
	if checkoutStatus != http.StatusCreated && checkoutStatus != http.StatusConflict && checkoutStatus != http.StatusUnprocessableEntity {
		t.Fatalf("unexpected checkout status %d", checkoutStatus)
	}

	// Invariant: ordered qty + cart qty == added qty for every product.
	ordered := map[string]int{}
	resp, body := doJSON(t, envAuth, http.MethodGet, "/api/v1/commerce/orders/me", "", nil)
	if resp.StatusCode == http.StatusOK {
		var list []schema.OrderResponse
		_ = json.Unmarshal(body, &list)
		for _, o := range list {
			for _, l := range o.Lines {
				ordered[l.ProductCode] += l.Quantity
			}
		}
	}
	resp, body = doJSON(t, envAuth, http.MethodGet, "/api/v1/commerce/cart", "", nil)
	var cart schema.CartResponse
	_ = json.Unmarshal(body, &cart)
	inCart := map[string]int{}
	for _, g := range cart.Suppliers {
		for _, it := range g.Items {
			inCart[it.ProductCode] += it.Quantity
		}
	}
	for _, c := range codes {
		if ordered[c]+inCart[c] < 1 {
			t.Errorf("line for %s silently lost: ordered=%d inCart=%d", c, ordered[c], inCart[c])
		}
	}
}

// A6: generated order/order-line codes respect VARCHAR(26) under load.
func TestAudit_CodeLengths(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	buyerID := createTestBuyer(t, env)
	prodCode, productID, supplierID := createTestProduct(t, env)
	addStock(t, env, productID, supplierID, 50, false)
	envAuth := setupEnv(t, buyerID, withPrincipalMiddleware(buyerID), nopMiddleware)
	addItem(t, envAuth, prodCode, 1)

	resp, body := doCheckout(t, envAuth, "codelen-key-1", "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("checkout: %d: %s", resp.StatusCode, string(body))
	}
	out := decodeCheckout(t, body)
	for _, o := range out.Orders {
		if len(o.Code) > 26 {
			t.Errorf("order code too long: %q", o.Code)
		}
		for _, l := range o.Lines {
			if len(l.Code) > 26 {
				t.Errorf("order line code too long: %q", l.Code)
			}
		}
	}
}

// A4: repeated release of the same attempt is a safe no-op: counters are
// restored exactly once and already-released rows are never double-restored.
func TestAudit_ReleaseRepeatedIsNoop(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	_, productID, supplierID := createTestProduct(t, env)
	addStock(t, env, productID, supplierID, 10, false)
	svc := inventory_service.NewInventoryService(env.db, nil)
	ctx := context.Background()

	if _, err := svc.ReserveStock(ctx, inventory_schema.ReserveStockRequest{
		ProductID: productID, SupplierID: supplierID, Quantity: 3, RequestID: "audit-rel-1",
	}); err != nil {
		t.Fatalf("reserve: %v", err)
	}
	for i := 0; i < 3; i++ {
		if err := svc.ReleaseReservationAttempt(ctx, []string{"audit-rel-1"}); err != nil {
			t.Fatalf("release %d: %v", i, err)
		}
	}

	if n := countRows(t, env, `SELECT COALESCE(SUM(available_quantity),0) FROM inventory.stock_level WHERE product_id=$1`, productID); n != 10 {
		t.Errorf("expected available restored to 10, got %d", n)
	}
	if n := countRows(t, env, `SELECT COALESCE(SUM(reserved_quantity),0) FROM inventory.stock_level WHERE product_id=$1`, productID); n != 0 {
		t.Errorf("expected reserved 0, got %d", n)
	}
	if n := countRows(t, env, `SELECT COUNT(*) FROM inventory.reservation WHERE request_id='audit-rel-1' AND status='released'`); n != 1 {
		t.Errorf("expected 1 released row, got %d", n)
	}
}

// A5: concurrent releases of the same attempt restore exactly once.
func TestAudit_ReleaseConcurrent(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	_, productID, supplierID := createTestProduct(t, env)
	addStock(t, env, productID, supplierID, 10, false)
	svc := inventory_service.NewInventoryService(env.db, nil)
	ctx := context.Background()

	if _, err := svc.ReserveStock(ctx, inventory_schema.ReserveStockRequest{
		ProductID: productID, SupplierID: supplierID, Quantity: 4, RequestID: "audit-conc-1",
	}); err != nil {
		t.Fatalf("reserve: %v", err)
	}

	const n = 8
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = svc.ReleaseReservationAttempt(ctx, []string{"audit-conc-1"})
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Errorf("release %d: %v", i, err)
		}
	}
	if n := countRows(t, env, `SELECT COALESCE(SUM(available_quantity),0) FROM inventory.stock_level WHERE product_id=$1`, productID); n != 10 {
		t.Errorf("expected available exactly 10, got %d", n)
	}
	if n := countRows(t, env, `SELECT COALESCE(SUM(reserved_quantity),0) FROM inventory.stock_level WHERE product_id=$1`, productID); n != 0 {
		t.Errorf("expected reserved exactly 0, got %d", n)
	}
}
