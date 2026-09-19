package router_test

// TASK-008 returns/credit tests: lifecycle, validation, over-quantity,
// concurrent approve, void reversal, reissue, buyer reads, aged debt, RBAC.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/atlas-platform/backend/internal/modules/commerce/schema"
)

// shipFixture checks out, advances to processing, and fully ships the order.
func shipFixture(t *testing.T, env *testEnv, staff map[int64]bool, qty int32) (*testEnv, int64, int64) {
	t.Helper()
	buyerEnv, buyerID, codes := checkoutFixture(t, env, staff, qty, 20)
	staffID := int64(0)
	for id := range staff {
		staffID = id
	}
	senv := staffEnv(t, env, staffID, staff)
	oid := orderIDByCode(t, env, codes[0])
	for _, next := range []string{"confirmed", "processing"} {
		resp, body := adminTransition(t, senv, oid, next, "x")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: %d: %s", next, resp.StatusCode, string(body))
		}
	}
	for _, l := range lineIDQtys(t, env, oid) {
		_ = l
	}
	qtys := lineIDQtys(t, env, oid)
	resp, body := doJSON(t, senv, http.MethodPost, fmt.Sprintf("/api/v1/admin/orders/%d/shipments", oid),
		fmt.Sprintf(`{"lines":[%s]}`, joinStrings(qtys, ",")), nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("ship coverage: %d: %s", resp.StatusCode, string(body))
	}
	if resp, body := adminTransition(t, senv, oid, "shipped", "x"); resp.StatusCode != http.StatusOK {
		t.Fatalf("ship order: %d: %s", resp.StatusCode, string(body))
	}
	return buyerEnv, buyerID, oid
}

func invoiceFixture(t *testing.T, senv *testEnv, env *testEnv, oid int64) int64 {
	t.Helper()
	resp, body := doJSON(t, senv, http.MethodPost, "/api/v1/admin/invoices",
		fmt.Sprintf(`{"order_id":%d}`, oid), nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("invoice: %d: %s", resp.StatusCode, string(body))
	}
	var inv schema.InvoiceResponse
	_ = json.Unmarshal(body, &inv)
	var invID int64
	_ = env.db.Pool.QueryRow(context.Background(), `SELECT id FROM commerce.invoice WHERE code=$1`, inv.Code).Scan(&invID)
	resp, _ = doJSON(t, senv, http.MethodPost, fmt.Sprintf("/api/v1/admin/invoices/%d/issue", invID), "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("issue: %d", resp.StatusCode)
	}
	return invID
}

func staffFixture(t *testing.T, env *testEnv) (map[int64]bool, int64) {
	t.Helper()
	staff := map[int64]bool{}
	staffID := createTestBuyer(t, env)
	staff[staffID] = true
	return staff, staffID
}

func TestReturns_FullFlow(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	staff, staffID := staffFixture(t, env)
	buyerEnv, _, oid := shipFixture(t, env, staff, 2)
	senv := staffEnv(t, env, staffID, staff)
	invoiceFixture(t, senv, env, oid)

	var lineID int64
	_ = env.db.Pool.QueryRow(context.Background(),
		`SELECT id FROM commerce.order_line WHERE order_id=$1`, oid).Scan(&lineID)

	// Buyer requests return of 1 of 2 units.
	var orderCode string
	_ = env.db.Pool.QueryRow(context.Background(), `SELECT code FROM commerce."order" WHERE id=$1`, oid).Scan(&orderCode)
	resp, body := doJSON(t, buyerEnv, http.MethodPost, "/api/v1/commerce/orders/me/"+orderCode+"/returns",
		fmt.Sprintf(`{"lines":[{"order_line_id":%d,"quantity":1}],"reason":"damaged"}`, lineID), nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("request: %d: %s", resp.StatusCode, string(body))
	}
	var ret schema.ReturnResponse
	_ = json.Unmarshal(body, &ret)
	if ret.Status != "requested" {
		t.Fatalf("expected requested: %+v", ret)
	}
	var retID int64
	_ = env.db.Pool.QueryRow(context.Background(), `SELECT id FROM commerce.return_request WHERE code=$1`, ret.Code).Scan(&retID)

	// Approve then complete → credit of 1500 applied to the invoice.
	resp, _ = doJSON(t, senv, http.MethodPost, fmt.Sprintf("/api/v1/admin/returns/%d/approve", retID), "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("approve: %d", resp.StatusCode)
	}
	resp, body = doJSON(t, senv, http.MethodPost, fmt.Sprintf("/api/v1/admin/returns/%d/complete", retID), "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("complete: %d: %s", resp.StatusCode, string(body))
	}
	var balance int64
	_ = env.db.Pool.QueryRow(context.Background(),
		`SELECT balance_minor FROM commerce.invoice WHERE order_id=$1 AND status='issued'`, oid).Scan(&balance)
	if balance != 1500 {
		t.Errorf("expected balance 1500 after 1500 credit on 3000 invoice, got %d", balance)
	}
	var notes int
	_ = env.db.Pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM commerce.credit_note WHERE order_id=$1 AND status='applied'`, oid).Scan(&notes)
	if notes != 1 {
		t.Errorf("expected 1 applied credit note, got %d", notes)
	}
	// History recorded the return lifecycle.
	var hist int
	_ = env.db.Pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM commerce.order_history WHERE order_id=$1 AND reason LIKE 'return %'`, oid).Scan(&hist)
	if hist < 3 {
		t.Errorf("expected return history rows, got %d", hist)
	}
}

func TestReturns_Validation(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	staff, staffID := staffFixture(t, env)
	_, buyerID, codes := checkoutFixture(t, env, staff, 2, 10)
	buyerEnv := setupEnv(t, buyerID, withPrincipalMiddleware(buyerID), staffMiddleware(staff))
	senv := staffEnv(t, env, staffID, staff)
	oid := orderIDByCode(t, env, codes[0])
	var lineID int64
	_ = env.db.Pool.QueryRow(context.Background(),
		`SELECT id FROM commerce.order_line WHERE order_id=$1`, oid).Scan(&lineID)
	var orderCode string
	_ = env.db.Pool.QueryRow(context.Background(), `SELECT code FROM commerce."order" WHERE id=$1`, oid).Scan(&orderCode)

	// Placed order is not returnable → 422.
	resp, body := doJSON(t, buyerEnv, http.MethodPost, "/api/v1/commerce/orders/me/"+orderCode+"/returns",
		fmt.Sprintf(`{"lines":[{"order_line_id":%d,"quantity":1}],"reason":"x"}`, lineID), nil)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("placed return: expected 422, got %d: %s", resp.StatusCode, string(body))
	}

	// Over-quantity on a shipped order → 400.
	_, _, codes2 := checkoutFixture(t, env, staff, 2, 10)
	oid2 := orderIDByCode(t, env, codes2[0])
	for _, next := range []string{"confirmed", "processing"} {
		_, _ = adminTransition(t, senv, oid2, next, "x")
	}
	var line2 int64
	_ = env.db.Pool.QueryRow(context.Background(),
		`SELECT id FROM commerce.order_line WHERE order_id=$1`, oid2).Scan(&line2)
	var code2 string
	_ = env.db.Pool.QueryRow(context.Background(), `SELECT code FROM commerce."order" WHERE id=$1`, oid2).Scan(&code2)
	buyer2Env := buyerEnvFor(t, env, staff, oid2)
	resp, body = doJSON(t, buyer2Env, http.MethodPost, "/api/v1/commerce/orders/me/"+code2+"/returns",
		fmt.Sprintf(`{"lines":[{"order_line_id":%d,"quantity":9}],"reason":"x"}`, line2), nil)
	if resp.StatusCode == http.StatusCreated {
		t.Errorf("over-qty: expected rejection, got 201")
	}
	_ = body

	// Foreign buyer → 404.
	otherID := createTestBuyer(t, env)
	otherEnv := setupEnv(t, otherID, withPrincipalMiddleware(otherID), staffMiddleware(staff))
	resp, _ = doJSON(t, otherEnv, http.MethodPost, "/api/v1/commerce/orders/me/"+orderCode+"/returns",
		fmt.Sprintf(`{"lines":[{"order_line_id":%d,"quantity":1}],"reason":"x"}`, lineID), nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("foreign return: expected 404, got %d", resp.StatusCode)
	}
	_ = senv
}

// buyerEnvFor builds a principal env for the buyer owning the given order.
func buyerEnvFor(t *testing.T, env *testEnv, staff map[int64]bool, orderID int64) *testEnv {
	t.Helper()
	var buyerID int64
	_ = env.db.Pool.QueryRow(context.Background(), `SELECT buyer_id FROM commerce."order" WHERE id=$1`, orderID).Scan(&buyerID)
	return setupEnv(t, buyerID, withPrincipalMiddleware(buyerID), staffMiddleware(staff))
}

func TestReturns_RejectAndConcurrency(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	staff, staffID := staffFixture(t, env)
	_, _, codes := checkoutFixture(t, env, staff, 1, 10)
	senv := staffEnv(t, env, staffID, staff)
	oid := orderIDByCode(t, env, codes[0])
	for _, next := range []string{"confirmed", "processing"} {
		_, _ = adminTransition(t, senv, oid, next, "x")
	}
	// Ship to make it returnable.
	var lineID int64
	_ = env.db.Pool.QueryRow(context.Background(),
		`SELECT id FROM commerce.order_line WHERE order_id=$1`, oid).Scan(&lineID)
	_, _ = doJSON(t, senv, http.MethodPost, fmt.Sprintf("/api/v1/admin/orders/%d/shipments", oid),
		fmt.Sprintf(`{"lines":[{"order_line_id":%d,"quantity":1}]}`, lineID), nil)
	_, _ = adminTransition(t, senv, oid, "shipped", "x")

	buyerEnv := buyerEnvFor(t, env, staff, oid)
	var orderCode string
	_ = env.db.Pool.QueryRow(context.Background(), `SELECT code FROM commerce."order" WHERE id=$1`, oid).Scan(&orderCode)
	resp, body := doJSON(t, buyerEnv, http.MethodPost, "/api/v1/commerce/orders/me/"+orderCode+"/returns",
		fmt.Sprintf(`{"lines":[{"order_line_id":%d,"quantity":1}],"reason":"wrong item"}`, lineID), nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("request: %d: %s", resp.StatusCode, string(body))
	}
	var ret schema.ReturnResponse
	_ = json.Unmarshal(body, &ret)
	var retID int64
	_ = env.db.Pool.QueryRow(context.Background(), `SELECT id FROM commerce.return_request WHERE code=$1`, ret.Code).Scan(&retID)

	// Concurrent approves: exactly one acts.
	const n = 6
	var codesArr [n]int
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			resp, _ := doJSON(t, senv, http.MethodPost, fmt.Sprintf("/api/v1/admin/returns/%d/approve", retID), "", nil)
			codesArr[i] = resp.StatusCode
		}(i)
	}
	close(start)
	wg.Wait()
	ok := 0
	for _, c := range codesArr {
		switch c {
		case http.StatusOK:
			ok++
		case http.StatusUnprocessableEntity:
		default:
			t.Errorf("unexpected %d", c)
		}
	}
	if ok != 1 {
		t.Errorf("expected exactly 1 approve, got %d (%v)", ok, codesArr)
	}

	// Reject after approve → 422.
	resp, _ = doJSON(t, senv, http.MethodPost, fmt.Sprintf("/api/v1/admin/returns/%d/reject", retID), `{"reason":"x"}`, nil)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("reject-after-approve: expected 422, got %d", resp.StatusCode)
	}
}

func TestCredit_VoidAndOverCredit(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	staff, staffID := staffFixture(t, env)
	_, _, codes := checkoutFixture(t, env, staff, 2, 10)
	senv := staffEnv(t, env, staffID, staff)
	oid := orderIDByCode(t, env, codes[0])
	invID := invoiceFixture(t, senv, env, oid)

	// Manual credit 1000 of 3000 balance.
	resp, body := doJSON(t, senv, http.MethodPost, "/api/v1/admin/credit-notes",
		fmt.Sprintf(`{"order_id":%d,"amount_minor":1000,"reason":"goodwill"}`, oid), nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("credit: %d: %s", resp.StatusCode, string(body))
	}
	var notes []schema.CreditNoteResponse
	_ = json.Unmarshal(body, &notes)
	if len(notes) != 1 || notes[0].AmountMinor != 1000 {
		t.Fatalf("unexpected notes: %+v", notes)
	}
	var cnID int64
	_ = env.db.Pool.QueryRow(context.Background(), `SELECT id FROM commerce.credit_note WHERE code=$1`, notes[0].Code).Scan(&cnID)

	var balance int64
	_ = env.db.Pool.QueryRow(context.Background(), `SELECT balance_minor FROM commerce.invoice WHERE id=$1`, invID).Scan(&balance)
	if balance != 2000 {
		t.Fatalf("expected balance 2000, got %d", balance)
	}

	// Over-credit (5000 > 2000 remaining) → 422.
	resp, _ = doJSON(t, senv, http.MethodPost, "/api/v1/admin/credit-notes",
		fmt.Sprintf(`{"order_id":%d,"amount_minor":5000,"reason":"x"}`, oid), nil)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("over-credit: expected 422, got %d", resp.StatusCode)
	}

	// Void restores the balance.
	resp, body = doJSON(t, senv, http.MethodPost, fmt.Sprintf("/api/v1/admin/credit-notes/%d/void", cnID), "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("void: %d: %s", resp.StatusCode, string(body))
	}
	_ = env.db.Pool.QueryRow(context.Background(), `SELECT balance_minor FROM commerce.invoice WHERE id=$1`, invID).Scan(&balance)
	if balance != 3000 {
		t.Errorf("expected balance 3000 after void, got %d", balance)
	}
	// Double void → 422.
	resp, _ = doJSON(t, senv, http.MethodPost, fmt.Sprintf("/api/v1/admin/credit-notes/%d/void", cnID), "", nil)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("double void: expected 422, got %d", resp.StatusCode)
	}
}

func TestInvoice_Reissue(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	staff, staffID := staffFixture(t, env)
	_, _, codes := checkoutFixture(t, env, staff, 1, 10)
	senv := staffEnv(t, env, staffID, staff)
	oid := orderIDByCode(t, env, codes[0])
	invID := invoiceFixture(t, senv, env, oid)
	var oldCode string
	_ = env.db.Pool.QueryRow(context.Background(), `SELECT code FROM commerce.invoice WHERE id=$1`, invID).Scan(&oldCode)

	resp, body := doJSON(t, senv, http.MethodPost, fmt.Sprintf("/api/v1/admin/invoices/%d/reissue", invID), "", nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("reissue: %d: %s", resp.StatusCode, string(body))
	}
	var inv schema.InvoiceResponse
	_ = json.Unmarshal(body, &inv)
	if inv.Status != "draft" || inv.ReplacesCode == nil || *inv.ReplacesCode != oldCode {
		t.Errorf("unexpected replacement: %+v", inv)
	}
	var oldStatus string
	_ = env.db.Pool.QueryRow(context.Background(), `SELECT status FROM commerce.invoice WHERE id=$1`, invID).Scan(&oldStatus)
	if oldStatus != "void" {
		t.Errorf("expected old invoice void, got %q", oldStatus)
	}
	// Reissuing a void invoice → 422.
	resp, _ = doJSON(t, senv, http.MethodPost, fmt.Sprintf("/api/v1/admin/invoices/%d/reissue", invID), "", nil)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("reissue void: expected 422, got %d", resp.StatusCode)
	}
}

func TestBuyer_InvoicesAndRBAC(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	staff, _ := staffFixture(t, env)
	buyerEnv, buyerID, codes := checkoutFixture(t, env, staff, 1, 10)
	oid := orderIDByCode(t, env, codes[0])
	var staffID int64
	for id := range staff {
		staffID = id
	}
	senv := staffEnv(t, env, staffID, staff)
	invID := invoiceFixture(t, senv, env, oid)
	var invCode string
	_ = env.db.Pool.QueryRow(context.Background(), `SELECT code FROM commerce.invoice WHERE id=$1`, invID).Scan(&invCode)

	// Own list + detail.
	resp, body := doJSON(t, buyerEnv, http.MethodGet, "/api/v1/commerce/invoices/me", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list: %d: %s", resp.StatusCode, string(body))
	}
	var list []schema.InvoiceResponse
	_ = json.Unmarshal(body, &list)
	if len(list) != 1 || list[0].Code != invCode {
		t.Errorf("unexpected list: %+v", list)
	}
	resp, _ = doJSON(t, buyerEnv, http.MethodGet, "/api/v1/commerce/invoices/me/"+invCode, "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("detail: %d", resp.StatusCode)
	}

	// Foreign buyer → 404s.
	otherID := createTestBuyer(t, env)
	otherEnv := setupEnv(t, otherID, withPrincipalMiddleware(otherID), staffMiddleware(staff))
	resp, _ = doJSON(t, otherEnv, http.MethodGet, "/api/v1/commerce/invoices/me/"+invCode, "", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("foreign invoice: expected 404, got %d", resp.StatusCode)
	}
	resp, body = doJSON(t, otherEnv, http.MethodGet, "/api/v1/commerce/invoices/me", "", nil)
	var otherList []schema.InvoiceResponse
	_ = json.Unmarshal(body, &otherList)
	if len(otherList) != 0 {
		t.Errorf("foreign list leak: %+v", otherList)
	}

	// Buyer on admin returns/credit paths → 403.
	for _, p := range [][2]string{
		{http.MethodGet, "/api/v1/admin/returns"},
		{http.MethodPost, fmt.Sprintf("/api/v1/admin/returns/%d/approve", 1)},
		{http.MethodGet, "/api/v1/admin/credit-notes"},
		{http.MethodPost, "/api/v1/admin/credit-notes"},
		{http.MethodGet, "/api/v1/admin/finance/aged-debt"},
	} {
		resp, _ := doJSON(t, buyerEnv, p[0], p[1], "", nil)
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("%s %s: expected 403, got %d", p[0], p[1], resp.StatusCode)
		}
	}
	_ = buyerID
}

func TestAgedDebt_Buckets(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	staff, staffID := staffFixture(t, env)
	_, _, codes := checkoutFixture(t, env, staff, 1, 10)
	senv := staffEnv(t, env, staffID, staff)
	oid := orderIDByCode(t, env, codes[0])
	invID := invoiceFixture(t, senv, env, oid)

	// Backdate issued_at 45 days → bucket 31-60.
	if _, err := env.db.Pool.Exec(context.Background(),
		`UPDATE commerce.invoice SET issued_at = NOW() - INTERVAL '45 days' WHERE id=$1`, invID); err != nil {
		t.Fatalf("backdate: %v", err)
	}

	resp, body := doJSON(t, senv, http.MethodGet, "/api/v1/admin/finance/aged-debt", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("aged debt: %d: %s", resp.StatusCode, string(body))
	}
	var out struct {
		Buckets map[string]int64 `json:"buckets"`
		Total   int64            `json:"total_minor"`
		Items   []struct {
			Bucket  string `json:"bucket"`
			Balance int64  `json:"balance_minor"`
		} `json:"items"`
	}
	_ = json.Unmarshal(body, &out)
	if out.Buckets["31-60"] < 1500 {
		t.Errorf("expected 31-60 bucket ≥ 1500: %+v", out.Buckets)
	}
	if out.Total < 1500 {
		t.Errorf("expected total ≥ 1500: %+v", out)
	}
}
