package router_test

// TASK-006 lifecycle tests: state machine matrix, hold, cancel+release,
// shipment coverage gating, invoices, RBAC. Staff vs buyer enforced by an
// injected staff-set middleware (production wiring remains a global stub).

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/atlas-platform/backend/internal/modules/commerce/router"
	"github.com/atlas-platform/backend/internal/modules/commerce/schema"
)

func staffMiddleware(staff map[int64]bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !staff[router.PrincipalIDFromContext(r.Context())] {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				fmt.Fprint(w, `{"detail":"staff only","code":"forbidden"}`)
				return
			}
			next.ServeHTTP(w, r.WithContext(r.Context()))
		})
	}
}

// checkoutFixture builds buyer + product + stock + cart item and checks out.
// Returns the buyer env (principal-injected), order codes, and product IDs.
func checkoutFixture(t *testing.T, env *testEnv, staff map[int64]bool, qty int32, stockQty int32) (*testEnv, int64, []string) {
	t.Helper()
	buyerID := createTestBuyer(t, env)
	prodCode, productID, supplierID := createTestProduct(t, env)
	addStock(t, env, productID, supplierID, stockQty, false)
	buyerEnv := setupEnv(t, buyerID, withPrincipalMiddleware(buyerID), staffMiddleware(staff))
	addItem(t, buyerEnv, prodCode, int(qty))

	resp, body := doCheckout(t, buyerEnv, fmt.Sprintf("lc-key-%d-%s", buyerID, uniqueSuffix()), "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("fixture checkout: %d: %s", resp.StatusCode, string(body))
	}
	out := decodeCheckout(t, body)
	var codes []string
	for _, o := range out.Orders {
		codes = append(codes, o.Code)
	}
	return buyerEnv, buyerID, codes
}

func orderIDByCode(t *testing.T, env *testEnv, code string) int64 {
	t.Helper()
	var id int64
	if err := env.db.Pool.QueryRow(context.Background(),
		`SELECT id FROM commerce."order" WHERE code = $1`, code).Scan(&id); err != nil {
		t.Fatalf("resolve order id: %v", err)
	}
	return id
}

func staffEnv(t *testing.T, env *testEnv, staffID int64, staff map[int64]bool) *testEnv {
	t.Helper()
	return setupEnv(t, staffID, withPrincipalMiddleware(staffID), staffMiddleware(staff))
}

func adminTransition(t *testing.T, env *testEnv, orderID int64, toStatus, reason string) (*http.Response, []byte) {
	t.Helper()
	return doJSON(t, env, http.MethodPost, fmt.Sprintf("/api/v1/admin/orders/%d/transition", orderID),
		fmt.Sprintf(`{"to_status":%q,"reason":%q}`, toStatus, reason), nil)
}

func TestLifecycle_FullMatrix(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	staff := map[int64]bool{}
	staffID := createTestBuyer(t, env) // staff principal (any id in the set)
	staff[staffID] = true
	_, _, codes := checkoutFixture(t, env, staff, 1, 10)
	senv := staffEnv(t, env, staffID, staff)
	oid := orderIDByCode(t, env, codes[0])

	// placed -> shipped is invalid (must confirm first).
	if resp, body := adminTransition(t, senv, oid, "shipped", "skip"); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("placed->shipped: expected 422, got %d: %s", resp.StatusCode, string(body))
	}

	// placed -> confirmed -> processing.
	for _, next := range []string{"confirmed", "processing"} {
		resp, body := adminTransition(t, senv, oid, next, "advance")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: %d: %s", next, resp.StatusCode, string(body))
		}
	}

	// processing -> shipped requires full shipment coverage.
	if resp, body := adminTransition(t, senv, oid, "shipped", "no coverage"); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("ship without coverage: expected 422, got %d: %s", resp.StatusCode, string(body))
	}

	// Ship full coverage via a shipment, then ship the order.
	var detail schema.AdminOrderDetailResponse
	resp, body := doJSON(t, senv, http.MethodGet, fmt.Sprintf("/api/v1/admin/orders/%d", oid), "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admin get order: %d: %s", resp.StatusCode, string(body))
	}
	_ = json.Unmarshal(body, &detail)
	shipBody := fmt.Sprintf(`{"lines":[%s]}`, joinStrings(lineIDQtys(t, env, oid), ","))
	resp, body = doJSON(t, senv, http.MethodPost, fmt.Sprintf("/api/v1/admin/orders/%d/shipments", oid), shipBody, nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create shipment: %d: %s", resp.StatusCode, string(body))
	}

	if resp, body := adminTransition(t, senv, oid, "shipped", "covered"); resp.StatusCode != http.StatusOK {
		t.Fatalf("ship: %d: %s", resp.StatusCode, string(body))
	}

	// shipped -> cancelled is forbidden; shipped -> delivered ok.
	if resp, _ := adminTransition(t, senv, oid, "cancelled", "too late"); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("shipped->cancelled: expected 422, got %d", resp.StatusCode)
	}
	if resp, body := adminTransition(t, senv, oid, "delivered", "done"); resp.StatusCode != http.StatusOK {
		t.Fatalf("deliver: %d: %s", resp.StatusCode, string(body))
	}

	// delivered is terminal.
	if resp, _ := adminTransition(t, senv, oid, "cancelled", "x"); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("delivered->cancelled: expected 422, got %d", resp.StatusCode)
	}

	// History: placed + confirmed + processing + shipment note + shipped + delivered.
	resp, body = doJSON(t, senv, http.MethodGet, fmt.Sprintf("/api/v1/admin/orders/%d", oid), "", nil)
	_ = json.Unmarshal(body, &detail)
	if len(detail.History) < 6 {
		t.Errorf("expected >=6 history rows, got %d: %+v", len(detail.History), detail.History)
	}
	statuses := map[string]bool{}
	for _, h := range detail.History {
		statuses[h.ToStatus] = true
	}
	for _, want := range []string{"placed", "confirmed", "processing", "shipped", "delivered"} {
		if !statuses[want] {
			t.Errorf("history missing to_status %q", want)
		}
	}
}

func lineIDByCode(t *testing.T, env *testEnv, orderID int64, productCode string) int64 {
	t.Helper()
	var id int64
	if err := env.db.Pool.QueryRow(context.Background(), `
		SELECT ol.id FROM commerce.order_line ol
		JOIN commerce."order" o ON o.id = ol.order_id
		WHERE o.id = $1 AND ol.product_code = $2
	`, orderID, productCode).Scan(&id); err != nil {
		t.Fatalf("resolve line id: %v", err)
	}
	return id
}

func lineIDQtys(t *testing.T, env *testEnv, orderID int64) []string {
	t.Helper()
	rows, err := env.db.Pool.Query(context.Background(), `
		SELECT id, quantity FROM commerce.order_line WHERE order_id = $1 ORDER BY id
	`, orderID)
	if err != nil {
		t.Fatalf("lines: %v", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id int64
		var qty int
		_ = rows.Scan(&id, &qty)
		out = append(out, fmt.Sprintf(`{"order_line_id":%d,"quantity":%d}`, id, qty))
	}
	return out
}

func joinStrings(in []string, sep string) string {
	out := ""
	for i, s := range in {
		if i > 0 {
			out += sep
		}
		out += s
	}
	return out
}

func TestLifecycle_HoldBlocks(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	staff := map[int64]bool{}
	staffID := createTestBuyer(t, env)
	staff[staffID] = true
	_, _, codes := checkoutFixture(t, env, staff, 1, 10)
	senv := staffEnv(t, env, staffID, staff)
	oid := orderIDByCode(t, env, codes[0])

	// Hold without reason → error (service-level 500? router passes reason).
	resp, body := doJSON(t, senv, http.MethodPost, fmt.Sprintf("/api/v1/admin/orders/%d/hold", oid), `{"reason":"fraud review"}`, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("hold: %d: %s", resp.StatusCode, string(body))
	}

	// Transitions blocked while held.
	if resp, _ := adminTransition(t, senv, oid, "confirmed", "x"); resp.StatusCode != http.StatusConflict {
		t.Errorf("held transition: expected 409, got %d", resp.StatusCode)
	}
	if resp, _ := adminTransition(t, senv, oid, "cancelled", "x"); resp.StatusCode != http.StatusConflict {
		t.Errorf("held cancel: expected 409, got %d", resp.StatusCode)
	}

	// Release and proceed.
	resp, body = doJSON(t, senv, http.MethodPost, fmt.Sprintf("/api/v1/admin/orders/%d/release", oid), `{"reason":"cleared"}`, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("release: %d: %s", resp.StatusCode, string(body))
	}
	if resp, body := adminTransition(t, senv, oid, "confirmed", "ok"); resp.StatusCode != http.StatusOK {
		t.Fatalf("confirm after release: %d: %s", resp.StatusCode, string(body))
	}
}

func TestLifecycle_CancelReleasesStock(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	staff := map[int64]bool{}
	staffID := createTestBuyer(t, env)
	staff[staffID] = true
	_, buyerID, codes := checkoutFixture(t, env, staff, 2, 10)
	senv := staffEnv(t, env, staffID, staff)
	oid := orderIDByCode(t, env, codes[0])

	// Stock held by checkout: find product, assert reserved == 2.
	var productID int64
	_ = env.db.Pool.QueryRow(context.Background(),
		`SELECT product_id FROM commerce.order_line WHERE order_id=$1`, oid).Scan(&productID)
	if n := countRows(t, senv, `SELECT COALESCE(SUM(reserved_quantity),0) FROM inventory.stock_level WHERE product_id=$1`, productID); n != 2 {
		t.Fatalf("precondition reserved=2, got %d", n)
	}

	if resp, body := adminTransition(t, senv, oid, "cancelled", "buyer asked"); resp.StatusCode != http.StatusOK {
		t.Fatalf("cancel: %d: %s", resp.StatusCode, string(body))
	}
	if n := countRows(t, senv, `SELECT COALESCE(SUM(reserved_quantity),0) FROM inventory.stock_level WHERE product_id=$1`, productID); n != 0 {
		t.Errorf("expected stock released, reserved=%d", n)
	}
	if n := countRows(t, senv, `SELECT COALESCE(SUM(available_quantity),0) FROM inventory.stock_level WHERE product_id=$1`, productID); n != 10 {
		t.Errorf("expected available restored to 10, got %d", n)
	}

	// Cancelled is terminal.
	if resp, _ := adminTransition(t, senv, oid, "confirmed", "x"); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("cancelled->confirmed: expected 422, got %d", resp.StatusCode)
	}
	_ = buyerID
}

func TestLifecycle_RBAC(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	staff := map[int64]bool{}
	staffID := createTestBuyer(t, env)
	staff[staffID] = true
	buyerEnv, _, codes := checkoutFixture(t, env, staff, 1, 10)
	oid := orderIDByCode(t, env, codes[0])

	// Buyer principal against admin paths → 403 from staff middleware.
	adminPaths := []struct{ method, path, body string }{
		{http.MethodGet, "/api/v1/admin/orders", ""},
		{http.MethodGet, fmt.Sprintf("/api/v1/admin/orders/%d", oid), ""},
		{http.MethodPost, fmt.Sprintf("/api/v1/admin/orders/%d/transition", oid), `{"to_status":"confirmed"}`},
		{http.MethodPost, fmt.Sprintf("/api/v1/admin/orders/%d/hold", oid), `{"reason":"x"}`},
		{http.MethodPost, fmt.Sprintf("/api/v1/admin/orders/%d/shipments", oid), `{"lines":[]}`},
		{http.MethodGet, "/api/v1/admin/shipments", ""},
		{http.MethodGet, "/api/v1/admin/invoices", ""},
		{http.MethodPost, "/api/v1/admin/invoices", fmt.Sprintf(`{"order_id":%d}`, oid)},
	}
	for _, p := range adminPaths {
		resp, body := doJSON(t, buyerEnv, p.method, p.path, p.body, nil)
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("%s %s: expected 403, got %d: %s", p.method, p.path, resp.StatusCode, string(body))
		}
	}

	// Malformed staff id → 400.
	senv := staffEnv(t, env, staffID, staff)
	resp, body := doJSON(t, senv, http.MethodGet, "/api/v1/admin/orders/abc", "", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("bad id: expected 400, got %d: %s", resp.StatusCode, string(body))
	}
	// Missing order → 404.
	resp, body = doJSON(t, senv, http.MethodGet, "/api/v1/admin/orders/999999999", "", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("missing order: expected 404, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestLifecycle_Notes(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	staff := map[int64]bool{}
	staffID := createTestBuyer(t, env)
	staff[staffID] = true
	_, _, codes := checkoutFixture(t, env, staff, 1, 10)
	senv := staffEnv(t, env, staffID, staff)
	oid := orderIDByCode(t, env, codes[0])

	resp, body := doJSON(t, senv, http.MethodPost, fmt.Sprintf("/api/v1/admin/orders/%d/notes", oid), `{"note":"called buyer"}`, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("note: %d: %s", resp.StatusCode, string(body))
	}

	resp, body = doJSON(t, senv, http.MethodGet, fmt.Sprintf("/api/v1/admin/orders/%d", oid), "", nil)
	var detail schema.AdminOrderDetailResponse
	_ = json.Unmarshal(body, &detail)
	found := false
	for _, h := range detail.History {
		if h.ToStatus == "placed" && len(h.Reason) >= 4 && h.Reason[:4] == "note" {
			found = true
			if h.Actor == nil || *h.Actor != staffID {
				t.Errorf("note actor should be staff %d, got %+v", staffID, h)
			}
		}
	}
	if !found {
		t.Errorf("note missing from history: %+v", detail.History)
	}
}
