package router_test

// TASK-006 fulfillment tests: partial shipments, coverage gating,
// over-ship rejection, shipment status flow, invoice lifecycle, envelopes.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/atlas-platform/backend/internal/modules/commerce/schema"
)

func TestFulfillment_PartialAndCoverageGating(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	staff := map[int64]bool{}
	staffID := createTestBuyer(t, env)
	staff[staffID] = true
	_, _, codes := checkoutFixture(t, env, staff, 4, 10)
	senv := staffEnv(t, env, staffID, staff)
	oid := orderIDByCode(t, env, codes[0])

	// Advance to processing (shippable state).
	for _, next := range []string{"confirmed", "processing"} {
		if resp, body := adminTransition(t, senv, oid, next, "x"); resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: %d: %s", next, resp.StatusCode, string(body))
		}
	}

	// Partial shipment of 1 of 4 units.
	qtys := lineIDQtys(t, env, oid)
	_ = qtys
	var lineID int64
	_ = env.db.Pool.QueryRow(context.Background(),
		`SELECT id FROM commerce.order_line WHERE order_id=$1`, oid).Scan(&lineID)
	resp, body := doJSON(t, senv, http.MethodPost, fmt.Sprintf("/api/v1/admin/orders/%d/shipments", oid),
		fmt.Sprintf(`{"lines":[{"order_line_id":%d,"quantity":1}],"carrier":"DHL","tracking_code":"TRK-1"}`, lineID), nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("partial shipment: %d: %s", resp.StatusCode, string(body))
	}
	var shp schema.ShipmentResponse
	_ = json.Unmarshal(body, &shp)
	if shp.Status != "preparing" || shp.Carrier != "DHL" || shp.TrackingCode != "TRK-1" {
		t.Errorf("unexpected shipment: %+v", shp)
	}

	// Order ship while partially covered → 422.
	if resp, _ := adminTransition(t, senv, oid, "shipped", "x"); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("partial ship: expected 422, got %d", resp.StatusCode)
	}

	// Ship the remaining 3 units, then the order ships.
	resp, body = doJSON(t, senv, http.MethodPost, fmt.Sprintf("/api/v1/admin/orders/%d/shipments", oid),
		fmt.Sprintf(`{"lines":[{"order_line_id":%d,"quantity":3}]}`, lineID), nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("remainder shipment: %d: %s", resp.StatusCode, string(body))
	}
	if resp, body := adminTransition(t, senv, oid, "shipped", "full"); resp.StatusCode != http.StatusOK {
		t.Fatalf("ship: %d: %s", resp.StatusCode, string(body))
	}
}

func TestFulfillment_OverShipRejected(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	staff := map[int64]bool{}
	staffID := createTestBuyer(t, env)
	staff[staffID] = true
	_, _, codes := checkoutFixture(t, env, staff, 2, 10)
	senv := staffEnv(t, env, staffID, staff)
	oid := orderIDByCode(t, env, codes[0])

	var lineID int64
	_ = env.db.Pool.QueryRow(context.Background(),
		`SELECT id FROM commerce.order_line WHERE order_id=$1`, oid).Scan(&lineID)
	resp, body := doJSON(t, senv, http.MethodPost, fmt.Sprintf("/api/v1/admin/orders/%d/shipments", oid),
		fmt.Sprintf(`{"lines":[{"order_line_id":%d,"quantity":5}]}`, lineID), nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("over-ship: expected 400, got %d: %s", resp.StatusCode, string(body))
	}

	// Unknown order line → 404.
	resp, body = doJSON(t, senv, http.MethodPost, fmt.Sprintf("/api/v1/admin/orders/%d/shipments", oid),
		`{"lines":[{"order_line_id":999999999,"quantity":1}]}`, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("unknown line: expected 404, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestFulfillment_ShipmentStatusFlow(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	staff := map[int64]bool{}
	staffID := createTestBuyer(t, env)
	staff[staffID] = true
	_, _, codes := checkoutFixture(t, env, staff, 1, 10)
	senv := staffEnv(t, env, staffID, staff)
	oid := orderIDByCode(t, env, codes[0])

	var lineID int64
	_ = env.db.Pool.QueryRow(context.Background(),
		`SELECT id FROM commerce.order_line WHERE order_id=$1`, oid).Scan(&lineID)
	resp, body := doJSON(t, senv, http.MethodPost, fmt.Sprintf("/api/v1/admin/orders/%d/shipments", oid),
		fmt.Sprintf(`{"lines":[{"order_line_id":%d,"quantity":1}]}`, lineID), nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("shipment: %d: %s", resp.StatusCode, string(body))
	}
	var shp schema.ShipmentResponse
	_ = json.Unmarshal(body, &shp)
	var shipID int64
	_ = env.db.Pool.QueryRow(context.Background(),
		`SELECT id FROM commerce.shipment WHERE code=$1`, shp.Code).Scan(&shipID)

	// preparing -> shipped -> delivered.
	for _, next := range []string{"shipped", "delivered"} {
		resp, body = doJSON(t, senv, http.MethodPatch, fmt.Sprintf("/api/v1/admin/shipments/%d", shipID),
			fmt.Sprintf(`{"tracking_code":"T-9","status":%q}`, next), nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: %d: %s", next, resp.StatusCode, string(body))
		}
	}
	// delivered is terminal.
	resp, _ = doJSON(t, senv, http.MethodPatch, fmt.Sprintf("/api/v1/admin/shipments/%d", shipID),
		`{"status":"shipped"}`, nil)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("terminal shipment: expected 422, got %d", resp.StatusCode)
	}
	// Unknown shipment → 404.
	resp, _ = doJSON(t, senv, http.MethodPatch, "/api/v1/admin/shipments/999999999", `{"status":"shipped"}`, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("missing shipment: expected 404, got %d", resp.StatusCode)
	}
}

func TestFulfillment_InvoiceFlow(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	staff := map[int64]bool{}
	staffID := createTestBuyer(t, env)
	staff[staffID] = true
	_, _, codes := checkoutFixture(t, env, staff, 2, 10)
	senv := staffEnv(t, env, staffID, staff)
	oid := orderIDByCode(t, env, codes[0])

	// Draft from order totals (2 x 1500).
	resp, body := doJSON(t, senv, http.MethodPost, "/api/v1/admin/invoices",
		fmt.Sprintf(`{"order_id":%d}`, oid), nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("draft: %d: %s", resp.StatusCode, string(body))
	}
	var inv schema.InvoiceResponse
	_ = json.Unmarshal(body, &inv)
	if inv.Status != "draft" || inv.TotalMinor != 3000 || inv.BalanceMinor != 3000 {
		t.Errorf("unexpected draft: %+v", inv)
	}
	var invID int64
	_ = env.db.Pool.QueryRow(context.Background(),
		`SELECT id FROM commerce.invoice WHERE code=$1`, inv.Code).Scan(&invID)

	// Issue, then re-issue is invalid.
	resp, body = doJSON(t, senv, http.MethodPost, fmt.Sprintf("/api/v1/admin/invoices/%d/issue", invID), "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("issue: %d: %s", resp.StatusCode, string(body))
	}
	_ = json.Unmarshal(body, &inv)
	if inv.Status != "issued" || inv.IssuedAt == nil {
		t.Errorf("expected issued with timestamp: %+v", inv)
	}
	resp, _ = doJSON(t, senv, http.MethodPost, fmt.Sprintf("/api/v1/admin/invoices/%d/issue", invID), "", nil)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("re-issue: expected 422, got %d", resp.StatusCode)
	}

	// Void keeps the row.
	resp, body = doJSON(t, senv, http.MethodPost, fmt.Sprintf("/api/v1/admin/invoices/%d/void", invID), "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("void: %d: %s", resp.StatusCode, string(body))
	}
	_ = json.Unmarshal(body, &inv)
	if inv.Status != "void" {
		t.Errorf("expected void: %+v", inv)
	}

	// Invoice for a cancelled order → 422.
	_, _, codes2 := checkoutFixture(t, env, staff, 1, 10)
	oid2 := orderIDByCode(t, env, codes2[0])
	if resp, _ := adminTransition(t, senv, oid2, "cancelled", "x"); resp.StatusCode != http.StatusOK {
		t.Fatalf("cancel: %d", resp.StatusCode)
	}
	resp, _ = doJSON(t, senv, http.MethodPost, "/api/v1/admin/invoices",
		fmt.Sprintf(`{"order_id":%d}`, oid2), nil)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("cancelled invoice: expected 422, got %d", resp.StatusCode)
	}
}

func TestFulfillment_AdminEnvelopes(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	staff := map[int64]bool{}
	staffID := createTestBuyer(t, env)
	staff[staffID] = true
	_, _, _ = checkoutFixture(t, env, staff, 1, 10)
	senv := staffEnv(t, env, staffID, staff)

	for path := range map[string]bool{
		"/api/v1/admin/orders":    true,
		"/api/v1/admin/shipments": true,
		"/api/v1/admin/invoices":  true,
	} {
		resp, body := doJSON(t, senv, http.MethodGet, path+"?page=1&page_size=20", "", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: %d: %s", path, resp.StatusCode, string(body))
		}
		var env2 struct {
			Items    []json.RawMessage `json:"items"`
			Page     int               `json:"page"`
			PageSize int               `json:"page_size"`
			Total    int               `json:"total"`
			HasNext  bool              `json:"has_next"`
		}
		if err := json.Unmarshal(body, &env2); err != nil {
			t.Fatalf("%s envelope: %v", path, err)
		}
		wantItems := env2.Total
		if wantItems > env2.PageSize {
			wantItems = env2.PageSize
		}
		if env2.Page != 1 || env2.PageSize != 20 || len(env2.Items) != wantItems || env2.HasNext != (env2.Total > env2.Page*env2.PageSize) {
			t.Errorf("%s: unexpected envelope page=%d size=%d total=%d items=%d has_next=%v",
				path, env2.Page, env2.PageSize, env2.Total, len(env2.Items), env2.HasNext)
		}
	}

	// Status filter: cancel a fresh order, then it must appear under cancelled.
	_, _, cancelCodes := checkoutFixture(t, env, staff, 1, 10)
	cancelOID := orderIDByCode(t, env, cancelCodes[0])
	if resp, _ := adminTransition(t, senv, cancelOID, "cancelled", "x"); resp.StatusCode != http.StatusOK {
		t.Fatalf("cancel: %d", resp.StatusCode)
	}
	resp, body := doJSON(t, senv, http.MethodGet, "/api/v1/admin/orders?status=cancelled", "", nil)
	var filtered struct {
		Items []json.RawMessage `json:"items"`
		Total int               `json:"total"`
	}
	_ = json.Unmarshal(body, &filtered)
	if filtered.Total < 1 || len(filtered.Items) != filtered.Total {
		t.Errorf("status filter: %+v", string(body))
	}
	_ = resp
}
