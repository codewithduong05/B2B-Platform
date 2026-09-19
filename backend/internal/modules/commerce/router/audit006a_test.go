package router_test

// TASK-006A concurrency probes: state-machine races, cancel compensation
// races, overshipment races. Each asserts exactly-once effects under
// -race -count=3. These FAIL on the pre-audit implementation.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/atlas-platform/backend/internal/modules/commerce/schema"
)

// advanceToProcessing moves a fresh order placed -> confirmed -> processing.
func advanceToProcessing(t *testing.T, senv *testEnv, oid int64) {
	t.Helper()
	for _, next := range []string{"confirmed", "processing"} {
		resp, body := adminTransition(t, senv, oid, next, "setup")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("setup %s: %d: %s", next, resp.StatusCode, string(body))
		}
	}
}

func historyCount(t *testing.T, env *testEnv, oid int64, toStatus string) int {
	t.Helper()
	var n int
	if err := env.db.Pool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM commerce.order_history WHERE order_id=$1 AND to_status=$2
	`, oid, toStatus).Scan(&n); err != nil {
		t.Fatalf("history count: %v", err)
	}
	return n
}

func orderStatus(t *testing.T, env *testEnv, oid int64) (string, bool) {
	t.Helper()
	var st string
	var hold bool
	if err := env.db.Pool.QueryRow(context.Background(),
		`SELECT status, on_hold FROM commerce."order" WHERE id=$1`, oid).Scan(&st, &hold); err != nil {
		t.Fatalf("order status: %v", err)
	}
	return st, hold
}

// T1: concurrent identical transitions — exactly one succeeds.
func TestAudit006A_ConcurrentSameTransition(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	staff := map[int64]bool{}
	staffID := createTestBuyer(t, env)
	staff[staffID] = true
	_, _, codes := checkoutFixture(t, env, staff, 1, 10)
	senv := staffEnv(t, env, staffID, staff)
	oid := orderIDByCode(t, env, codes[0])

	const n = 8
	var statuses [n]int
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			resp, _ := adminTransition(t, senv, oid, "confirmed", "race")
			statuses[i] = resp.StatusCode
		}(i)
	}
	close(start)
	wg.Wait()

	ok, conflict := 0, 0
	for i, st := range statuses {
		switch st {
		case http.StatusOK:
			ok++
		case http.StatusConflict:
			// Raced the winner: read placed, lost the lock.
			conflict++
		case http.StatusUnprocessableEntity:
			// Arrived after the winner committed: read confirmed,
			// confirmed->confirmed is sequentially invalid. Also correct.
			conflict++
		default:
			t.Errorf("req %d: unexpected status %d", i, st)
		}
	}
	if ok != 1 {
		t.Errorf("expected exactly 1 successful transition, got %d (%v)", ok, statuses)
	}
	if conflict != n-1 {
		t.Errorf("expected %d losers (409/422), got %d (%v)", n-1, conflict, statuses)
	}
	if c := historyCount(t, env, oid, "confirmed"); c != 1 {
		t.Errorf("expected exactly 1 confirmed history row, got %d", c)
	}
}

// T2: ship on an already-held order is rejected for every contender.
func TestAudit006A_ShipWhileHeld(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	staff := map[int64]bool{}
	staffID := createTestBuyer(t, env)
	staff[staffID] = true
	_, _, codes := checkoutFixture(t, env, staff, 1, 10)
	senv := staffEnv(t, env, staffID, staff)
	oid := orderIDByCode(t, env, codes[0])
	advanceToProcessing(t, senv, oid)

	// Full coverage so only the hold blocks the ship.
	var lineID int64
	_ = env.db.Pool.QueryRow(context.Background(),
		`SELECT id FROM commerce.order_line WHERE order_id=$1`, oid).Scan(&lineID)
	resp, body := doJSON(t, senv, http.MethodPost, fmt.Sprintf("/api/v1/admin/orders/%d/shipments", oid),
		fmt.Sprintf(`{"lines":[{"order_line_id":%d,"quantity":1}]}`, lineID), nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("coverage shipment: %d: %s", resp.StatusCode, string(body))
	}

	resp, _ = doJSON(t, senv, http.MethodPost, fmt.Sprintf("/api/v1/admin/orders/%d/hold", oid), `{"reason":"review"}`, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("hold: %d", resp.StatusCode)
	}

	const n = 6
	var statuses [n]int
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			resp, _ := adminTransition(t, senv, oid, "shipped", "race")
			statuses[i] = resp.StatusCode
		}(i)
	}
	close(start)
	wg.Wait()
	for i, st := range statuses {
		if st != http.StatusConflict {
			t.Errorf("req %d: held ship must be 409, got %d", i, st)
		}
	}
	if st, _ := orderStatus(t, env, oid); st != "processing" {
		t.Errorf("held order must stay processing, got %q", st)
	}
	if c := historyCount(t, env, oid, "shipped"); c != 0 {
		t.Errorf("no shipped history expected, got %d", c)
	}
}

// T3: concurrent cancels — exactly one succeeds, stock restored once.
func TestAudit006A_ConcurrentCancel(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	staff := map[int64]bool{}
	staffID := createTestBuyer(t, env)
	staff[staffID] = true
	_, _, codes := checkoutFixture(t, env, staff, 2, 10)
	senv := staffEnv(t, env, staffID, staff)
	oid := orderIDByCode(t, env, codes[0])

	var productID int64
	_ = env.db.Pool.QueryRow(context.Background(),
		`SELECT product_id FROM commerce.order_line WHERE order_id=$1`, oid).Scan(&productID)

	const n = 5
	var statuses [n]int
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			resp, _ := adminTransition(t, senv, oid, "cancelled", "race")
			statuses[i] = resp.StatusCode
		}(i)
	}
	close(start)
	wg.Wait()

	ok := 0
	for i, st := range statuses {
		switch st {
		case http.StatusOK:
			ok++
		case http.StatusConflict, http.StatusUnprocessableEntity:
		default:
			t.Errorf("req %d: unexpected status %d", i, st)
		}
	}
	if ok != 1 {
		t.Errorf("expected exactly 1 successful cancel, got %d (%v)", ok, statuses)
	}
	if st, _ := orderStatus(t, env, oid); st != "cancelled" {
		t.Errorf("expected cancelled, got %q", st)
	}
	if c := historyCount(t, env, oid, "cancelled"); c != 1 {
		t.Errorf("expected exactly 1 cancelled history row, got %d", c)
	}
	if r := countRows(t, senv, `SELECT COALESCE(SUM(reserved_quantity),0) FROM inventory.stock_level WHERE product_id=$1`, productID); r != 0 {
		t.Errorf("expected stock fully released, reserved=%d", r)
	}
	if a := countRows(t, senv, `SELECT COALESCE(SUM(available_quantity),0) FROM inventory.stock_level WHERE product_id=$1`, productID); a != 10 {
		t.Errorf("expected available restored to 10, got %d", a)
	}
}

// T4: cancel racing a ship — mutual exclusion between released stock and
// shipped status, regardless of who wins.
func TestAudit006A_CancelVsShip(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	staff := map[int64]bool{}
	staffID := createTestBuyer(t, env)
	staff[staffID] = true
	_, _, codes := checkoutFixture(t, env, staff, 2, 10)
	senv := staffEnv(t, env, staffID, staff)
	oid := orderIDByCode(t, env, codes[0])
	advanceToProcessing(t, senv, oid)

	var productID, lineID int64
	_ = env.db.Pool.QueryRow(context.Background(),
		`SELECT product_id FROM commerce.order_line WHERE order_id=$1`, oid).Scan(&productID)
	_ = env.db.Pool.QueryRow(context.Background(),
		`SELECT id FROM commerce.order_line WHERE order_id=$1`, oid).Scan(&lineID)

	// Coverage shipment first so ship is eligible.
	resp, body := doJSON(t, senv, http.MethodPost, fmt.Sprintf("/api/v1/admin/orders/%d/shipments", oid),
		fmt.Sprintf(`{"lines":[{"order_line_id":%d,"quantity":2}]}`, lineID), nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("coverage: %d: %s", resp.StatusCode, string(body))
	}

	var cancelCode, shipCode int
	var wg sync.WaitGroup
	start := make(chan struct{})
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		resp, _ := adminTransition(t, senv, oid, "cancelled", "race")
		cancelCode = resp.StatusCode
	}()
	go func() {
		defer wg.Done()
		<-start
		resp, _ := adminTransition(t, senv, oid, "shipped", "race")
		shipCode = resp.StatusCode
	}()
	close(start)
	wg.Wait()

	st, _ := orderStatus(t, env, oid)
	reserved := countRows(t, senv, `SELECT COALESCE(SUM(reserved_quantity),0) FROM inventory.stock_level WHERE product_id=$1`, productID)
	t.Logf("outcome: cancel=%d ship=%d final=%q reserved=%d", cancelCode, shipCode, st, reserved)
	switch st {
	case "shipped":
		if shipCode != http.StatusOK {
			t.Errorf("shipped final state but ship call got %d", shipCode)
		}
		if reserved != 2 {
			t.Errorf("shipped order must keep its 2-unit hold, reserved=%d", reserved)
		}
	case "cancelled":
		if cancelCode != http.StatusOK {
			t.Errorf("cancelled final state but cancel call got %d", cancelCode)
		}
		if reserved != 0 {
			t.Errorf("cancelled order must release stock, reserved=%d", reserved)
		}
	default:
		t.Errorf("unexpected final status %q (cancel=%d ship=%d)", st, cancelCode, shipCode)
	}
	if c := historyCount(t, env, oid, "cancelled") + historyCount(t, env, oid, "shipped"); c != 1 {
		t.Errorf("expected exactly one terminal history row, got %d", c)
	}
}

// T5: cancel with active shipments is rejected with 409; stock stays held.
func TestAudit006A_CancelWithShipmentsRejected(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	staff := map[int64]bool{}
	staffID := createTestBuyer(t, env)
	staff[staffID] = true
	_, _, codes := checkoutFixture(t, env, staff, 2, 10)
	senv := staffEnv(t, env, staffID, staff)
	oid := orderIDByCode(t, env, codes[0])

	var productID, lineID int64
	_ = env.db.Pool.QueryRow(context.Background(),
		`SELECT product_id FROM commerce.order_line WHERE order_id=$1`, oid).Scan(&productID)
	_ = env.db.Pool.QueryRow(context.Background(),
		`SELECT id FROM commerce.order_line WHERE order_id=$1`, oid).Scan(&lineID)
	resp, body := doJSON(t, senv, http.MethodPost, fmt.Sprintf("/api/v1/admin/orders/%d/shipments", oid),
		fmt.Sprintf(`{"lines":[{"order_line_id":%d,"quantity":1}]}`, lineID), nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("shipment: %d: %s", resp.StatusCode, string(body))
	}

	resp, body = adminTransition(t, senv, oid, "cancelled", "with shipments")
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("cancel with shipments: expected 409, got %d: %s", resp.StatusCode, string(body))
	}
	if st, _ := orderStatus(t, env, oid); st != "placed" {
		t.Errorf("order must stay placed, got %q", st)
	}
	if r := countRows(t, senv, `SELECT COALESCE(SUM(reserved_quantity),0) FROM inventory.stock_level WHERE product_id=$1`, productID); r != 2 {
		t.Errorf("stock must stay held, reserved=%d", r)
	}
}

// T6: concurrent partial shipments summing over the ordered qty — exactly
// one wins fully, the other is rejected; never over-shipped.
func TestAudit006A_OvershipRace(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	staff := map[int64]bool{}
	staffID := createTestBuyer(t, env)
	staff[staffID] = true
	_, _, codes := checkoutFixture(t, env, staff, 4, 10)
	senv := staffEnv(t, env, staffID, staff)
	oid := orderIDByCode(t, env, codes[0])
	advanceToProcessing(t, senv, oid)

	var lineID int64
	_ = env.db.Pool.QueryRow(context.Background(),
		`SELECT id FROM commerce.order_line WHERE order_id=$1`, oid).Scan(&lineID)

	const n = 2
	var statuses [n]int
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			resp, _ := doJSON(t, senv, http.MethodPost, fmt.Sprintf("/api/v1/admin/orders/%d/shipments", oid),
				fmt.Sprintf(`{"lines":[{"order_line_id":%d,"quantity":3}]}`, lineID), nil)
			statuses[i] = resp.StatusCode
		}(i)
	}
	close(start)
	wg.Wait()

	created := 0
	for i, st := range statuses {
		if st == http.StatusCreated {
			created++
		} else if st != http.StatusBadRequest && st != http.StatusConflict {
			// 400: sequential-shape over-ship caught by pre-check;
			// 409: raced over-ship caught by locked re-validation.
			t.Errorf("req %d: unexpected status %d", i, st)
		}
	}
	if created != 1 {
		t.Errorf("expected exactly 1 winning shipment, got %d (%v)", created, statuses)
	}
	var total int
	_ = env.db.Pool.QueryRow(context.Background(), `
		SELECT COALESCE(SUM(sl.quantity),0) FROM commerce.shipment_line sl
		JOIN commerce.shipment s ON s.id = sl.shipment_id
		WHERE s.order_id=$1 AND s.status != 'cancelled' AND s.deleted_at IS NULL
	`, oid).Scan(&total)
	if total != 3 {
		t.Errorf("expected exactly 3 shipped units, got %d", total)
	}
}

func TestAudit006A_OrderDetailShape(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	staff := map[int64]bool{}
	staffID := createTestBuyer(t, env)
	staff[staffID] = true
	_, _, codes := checkoutFixture(t, env, staff, 1, 10)
	senv := staffEnv(t, env, staffID, staff)
	oid := orderIDByCode(t, env, codes[0])

	resp, body := doJSON(t, senv, http.MethodGet, fmt.Sprintf("/api/v1/admin/orders/%d", oid), "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admin detail: %d: %s", resp.StatusCode, string(body))
	}
	var detail schema.AdminOrderDetailResponse
	if err := json.Unmarshal(body, &detail); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if detail.Code != codes[0] || len(detail.Lines) != 1 || len(detail.History) != 1 {
		t.Errorf("unexpected detail: %+v", detail)
	}
}
