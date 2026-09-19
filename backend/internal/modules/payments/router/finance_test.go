package router_test

// TASK-007 tests: signed webhooks (matrix + duplicates + out-of-order),
// refunds with approval threshold, reconciliation buckets, credit accounts
// and the checkout credit gate.

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	payments_schema "github.com/atlas-platform/backend/internal/modules/payments/schema"
)

const webhookTestSecret = "test-webhook-secret"

// signWebhook implements the provider side: HMAC(timestamp + "." + body).
func signWebhook(t *testing.T, body string, skew time.Duration) (ts, sig string) {
	t.Helper()
	ts = fmt.Sprintf("%d", time.Now().Add(skew).Unix())
	mac := hmac.New(sha256.New, []byte(webhookTestSecret))
	mac.Write([]byte(ts + "." + body))
	return ts, hex.EncodeToString(mac.Sum(nil))
}

func postWebhook(t *testing.T, env *payEnv, provider, body, ts, sig string) (*http.Response, []byte) {
	t.Helper()
	return payDo(t, env, http.MethodPost, "/webhooks/payments/"+provider, body,
		map[string]string{"X-Webhook-Timestamp": ts, "X-Webhook-Signature": sig})
}

func webhookEnv(t *testing.T, buyerID int64, staff map[int64]bool) *payEnv {
	t.Helper()
	env := setupPayEnv(t, buyerID, staff)
	env.paymentSvc.SetWebhookSecrets(map[string]string{"sim": webhookTestSecret})
	return env
}

func webhookIntentFixture(t *testing.T, env *payEnv, buyerEnv *payEnv, staffEnv *payEnv, key, checkoutKey string, qty int) (orderCode, intentCode string, orderID int64) {
	t.Helper()
	prodCode, productID, supplierID := payProduct(t, env)
	payStock(t, env, productID, supplierID, 20)
	orderCode = payOrderFixture(t, env, buyerEnv, prodCode, checkoutKey)
	_ = qty
	resp, body := payDo(t, buyerEnv, http.MethodPost, "/api/v1/payments/intents",
		fmt.Sprintf(`{"order_code":%q,"idem_key":%q}`, orderCode, key), nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("intent: %d: %s", resp.StatusCode, string(body))
	}
	var out payments_schema.IntentResponse
	_ = json.Unmarshal(body, &out)
	return orderCode, out.Code, payOrderID(t, env, orderCode)
}

func TestWebhook_Matrix(t *testing.T) {
	env := webhookEnv(t, 0, map[int64]bool{})
	buyerID := payBuyer(t, env)
	staff := map[int64]bool{}
	buyerEnv := setupPayEnv(t, buyerID, staff)
	orderCode, intentCode, _ := webhookIntentFixture(t, env, buyerEnv, buyerEnv, "wh-1", "wh-co-1", 1)

	// Valid succeeded callback.
	evt := "evt-1-" + paySuffix()
	body := fmt.Sprintf(`{"event_id":%q,"type":"intent.succeeded","intent_code":%q}`, evt, intentCode)
	ts, sig := signWebhook(t, body, 0)
	resp, rbody := postWebhook(t, env, "sim", body, ts, sig)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("valid webhook: %d: %s", resp.StatusCode, string(rbody))
	}
	var out struct {
		Applied bool   `json:"applied"`
		Status  string `json:"status"`
	}
	_ = json.Unmarshal(rbody, &out)
	if !out.Applied || out.Status != "succeeded" {
		t.Errorf("unexpected outcome: %+v", out)
	}

	// Exact duplicate delivery → same outcome, no double-apply.
	resp, rbody = postWebhook(t, env, "sim", body, ts, sig)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("duplicate: %d: %s", resp.StatusCode, string(rbody))
	}
	var attempts int
	_ = env.db.Pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM payments.payment_attempt WHERE intent_id=(SELECT id FROM payments.payment_intent WHERE code=$1)`, intentCode).Scan(&attempts)
	if attempts != 1 {
		t.Errorf("duplicate delivery double-applied: %d attempts", attempts)
	}

	// Bad signature → 401.
	resp, _ = postWebhook(t, env, "sim", body, ts, "deadbeef")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("bad sig: expected 401, got %d", resp.StatusCode)
	}

	// Stale timestamp → 401.
	sts, ssig := signWebhook(t, body, -time.Hour)
	resp, _ = postWebhook(t, env, "sim", body, sts, ssig)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("stale: expected 401, got %d", resp.StatusCode)
	}

	// Unknown provider → 401.
	resp, _ = postWebhook(t, env, "nope", body, ts, sig)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("unknown provider: expected 401, got %d", resp.StatusCode)
	}

	// Malformed body with valid signature → 400.
	badBody := `{"event_id":`
	bts, bsig := signWebhook(t, badBody, 0)
	resp, _ = postWebhook(t, env, "sim", badBody, bts, bsig)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("malformed: expected 400, got %d", resp.StatusCode)
	}

	// Unknown intent → 202 unmatched (and replays as unmatched).
	ubody := fmt.Sprintf(`{"event_id":"evt-unknown-1-%s","type":"intent.succeeded","intent_code":"pi_missing"}`, paySuffix())
	uts, usig := signWebhook(t, ubody, 0)
	resp, rbody = postWebhook(t, env, "sim", ubody, uts, usig)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("unknown intent: expected 202, got %d: %s", resp.StatusCode, string(rbody))
	}
	resp, _ = postWebhook(t, env, "sim", ubody, uts, usig)
	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("unknown replay: expected 202, got %d", resp.StatusCode)
	}
	_ = orderCode
}

func TestWebhook_OutOfOrderAndDelayed(t *testing.T) {
	env := webhookEnv(t, 0, map[int64]bool{})
	buyerID := payBuyer(t, env)
	staff := map[int64]bool{}
	buyerEnv := setupPayEnv(t, buyerID, staff)
	_, intentCode, _ := webhookIntentFixture(t, env, buyerEnv, buyerEnv, "wh-oo-1", "wh-oo-co-1", 1)

	// Failed delivery arrives first, succeeded (older business outcome)
	// arrives delayed afterwards.
	fbody := fmt.Sprintf(`{"event_id":"evt-oo-fail-%s","type":"intent.failed","intent_code":%q}`, paySuffix(), intentCode)
	fts, fsig := signWebhook(t, fbody, 0)
	resp, _ := postWebhook(t, env, "sim", fbody, fts, fsig)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("fail first: %d", resp.StatusCode)
	}
	sbody := fmt.Sprintf(`{"event_id":"evt-oo-succ-%s","type":"intent.succeeded","intent_code":%q}`, paySuffix(), intentCode)
	sts, ssig := signWebhook(t, sbody, 0)
	resp, rbody := postWebhook(t, env, "sim", sbody, sts, ssig)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("late success on terminal intent: expected 202, got %d: %s", resp.StatusCode, string(rbody))
	}

	var status string
	_ = env.db.Pool.QueryRow(context.Background(),
		`SELECT status FROM payments.payment_intent WHERE code=$1`, intentCode).Scan(&status)
	if status != "failed" {
		t.Errorf("terminal intent must not flip: %q", status)
	}
}

func TestWebhook_ConcurrentDuplicates(t *testing.T) {
	env := webhookEnv(t, 0, map[int64]bool{})
	buyerID := payBuyer(t, env)
	staff := map[int64]bool{}
	buyerEnv := setupPayEnv(t, buyerID, staff)
	_, intentCode, _ := webhookIntentFixture(t, env, buyerEnv, buyerEnv, "wh-cc-1", "wh-cc-co-1", 1)

	body := fmt.Sprintf(`{"event_id":"evt-cc-1-%s","type":"intent.succeeded","intent_code":%q}`, paySuffix(), intentCode)
	ts, sig := signWebhook(t, body, 0)

	const n = 10
	var codes [n]int
	var firstBody string
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			resp, body := postWebhook(t, env, "sim", body, ts, sig)
			codes[i] = resp.StatusCode
			if i == 0 {
				firstBody = string(body)
			}
		}(i)
	}
	close(start)
	wg.Wait()
	for i, c := range codes {
		if c != http.StatusOK {
			t.Errorf("req %d: expected 200, got %d (%s)", i, c, firstBody)
		}
	}
	var attempts int
	_ = env.db.Pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM payments.payment_attempt WHERE intent_id=(SELECT id FROM payments.payment_intent WHERE code=$1)`, intentCode).Scan(&attempts)
	if attempts != 1 {
		t.Errorf("concurrent duplicates applied %d times", attempts)
	}
}

func TestRefund_ThresholdAndFlow(t *testing.T) {	env := webhookEnv(t, 0, map[int64]bool{})
	buyerID := payBuyer(t, env)
	staffA, staffB := payBuyer(t, env), payBuyer(t, env)
	staff := map[int64]bool{staffA: true, staffB: true}
	buyerEnv := setupPayEnv(t, buyerID, staff)
	staffEnvA := setupPayEnv(t, staffA, staff)
	staffEnvB := setupPayEnv(t, staffB, staff)

	// Small order: qty 2 x 1500 = 3000.
	orderCode, intentCode, oid := webhookIntentFixture(t, env, buyerEnv, buyerEnv, "rf-1", "rf-co-1", 2)
	payIssueInvoiceForRefund(t, env, staffEnvA, oid)
	completeIntent(t, staffEnvA, intentCode)

	// Small refund (500 < threshold) → auto-approved on request.
	refundCode := requestRefund(t, staffEnvA, intentCode, 500, "short-ship")
	var balance int64
	_ = env.db.Pool.QueryRow(context.Background(),
		`SELECT balance_minor FROM commerce.invoice WHERE order_id=$1 AND status='issued'`, oid).Scan(&balance)
	if balance != 0 {
		t.Fatalf("balance should still be 0 before apply, got %d", balance)
	}
	// Approve (applies): balance restored to 500.
	approveRefund(t, env, staffEnvA, refundCode, http.StatusOK)
	_ = env.db.Pool.QueryRow(context.Background(),
		`SELECT balance_minor FROM commerce.invoice WHERE order_id=$1 AND status='issued'`, oid).Scan(&balance)
	if balance != 500 {
		t.Errorf("expected balance 500 after refund apply, got %d", balance)
	}
	// Re-approve applied refund → 422.
	resp, _ := payDo(t, staffEnvA, http.MethodPost, "/api/v1/admin/payments/refunds/"+refundID(t, env, refundCode)+"/approve", `{}`, nil)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("re-approve: expected 422, got %d", resp.StatusCode)
	}

	// Over-refund (5000 > captured 3000) → 422.
	resp, body := payDo(t, staffEnvA, http.MethodPost, "/api/v1/admin/payments/intents/"+intentCode+"/refund",
		`{"amount_minor":5000,"reason":"too much"}`, nil)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("over-refund: expected 422, got %d: %s", resp.StatusCode, string(body))
	}

	// Large order for threshold testing: qty 700 x 1500 = 1,050,000.
	_, bigIntent, bigOID := webhookBigOrder(t, env, buyerEnv, staffEnvA, "rf-big", "rf-co-big", 700)
	payIssueInvoiceForRefund(t, env, staffEnvA, bigOID)
	completeIntent(t, staffEnvA, bigIntent)

	// Large refund → pending_approval; same-approver forbidden.
	resp, body = payDo(t, staffEnvA, http.MethodPost, "/api/v1/admin/payments/intents/"+bigIntent+"/refund",
		`{"amount_minor":1050000,"reason":"cancelled pallet"}`, nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("large refund request: %d: %s", resp.StatusCode, string(body))
	}
	var rf struct {
		Code   string `json:"code"`
		Status string `json:"status"`
	}
	_ = json.Unmarshal(body, &rf)
	if rf.Status != "pending_approval" {
		t.Fatalf("expected pending_approval, got %+v", rf)
	}
	bigRefundID := refundID(t, env, rf.Code)
	resp, _ = payDo(t, staffEnvA, http.MethodPost, "/api/v1/admin/payments/refunds/"+bigRefundID+"/approve", `{}`, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("self-approve: expected 403, got %d", resp.StatusCode)
	}
	// Second staffer approves → applied, balance restored.
	resp, body = payDo(t, staffEnvB, http.MethodPost, "/api/v1/admin/payments/refunds/"+bigRefundID+"/approve", `{}`, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("approve: %d: %s", resp.StatusCode, string(body))
	}
	_ = env.db.Pool.QueryRow(context.Background(),
		`SELECT balance_minor FROM commerce.invoice WHERE order_id=$1 AND status='issued'`, bigOID).Scan(&balance)
	if balance != 1050000 {
		t.Errorf("expected balance 1050000, got %d", balance)
	}

	// Reject flow needs its own pending refund: fresh big order, full refund.
	big2Code, big2Intent, big2OID := webhookBigOrder(t, env, buyerEnv, staffEnvA, "rf-big2", "rf-co-big2", 700)
	_ = big2Code
	payIssueInvoiceForRefund(t, env, staffEnvA, big2OID)
	completeIntent(t, staffEnvA, big2Intent)
	resp, body = payDo(t, staffEnvA, http.MethodPost, "/api/v1/admin/payments/intents/"+big2Intent+"/refund",
		`{"amount_minor":1000000,"reason":"goodwill"}`, nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("reject-flow refund: %d: %s", resp.StatusCode, string(body))
	}
	_ = json.Unmarshal(body, &rf)
	rejectID := refundID(t, env, rf.Code)
	resp, _ = payDo(t, staffEnvB, http.MethodPost, "/api/v1/admin/payments/refunds/"+rejectID+"/reject", `{}`, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("reject: %d", resp.StatusCode)
	}
	resp, _ = payDo(t, staffEnvB, http.MethodPost, "/api/v1/admin/payments/refunds/"+rejectID+"/approve", `{}`, nil)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("approve-after-reject: expected 422, got %d", resp.StatusCode)
	}
	_ = orderCode
	_ = oid
}

func completeIntent(t *testing.T, staffEnv *payEnv, intentCode string) {
	t.Helper()
	resp, body := payDo(t, staffEnv, http.MethodPost, "/api/v1/admin/payments/intents/"+intentCode+"/complete", `{}`, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("complete: %d: %s", resp.StatusCode, string(body))
	}
}

func requestRefund(t *testing.T, staffEnv *payEnv, intentCode string, amount int64, reason string) string {
	t.Helper()
	resp, body := payDo(t, staffEnv, http.MethodPost, "/api/v1/admin/payments/intents/"+intentCode+"/refund",
		fmt.Sprintf(`{"amount_minor":%d,"reason":%q}`, amount, reason), nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("request refund: %d: %s", resp.StatusCode, string(body))
	}
	var rf struct {
		Code   string `json:"code"`
		Status string `json:"status"`
	}
	_ = json.Unmarshal(body, &rf)
	return rf.Code
}

func approveRefund(t *testing.T, env *payEnv, staffEnv *payEnv, refundCode string, want int) {
	t.Helper()
	resp, body := payDo(t, staffEnv, http.MethodPost, "/api/v1/admin/payments/refunds/"+refundID(t, env, refundCode)+"/approve", `{}`, nil)
	if resp.StatusCode != want {
		t.Fatalf("approve: expected %d, got %d: %s", want, resp.StatusCode, string(body))
	}
}

func refundID(t *testing.T, env *payEnv, code string) string {
	t.Helper()
	var id int64
	_ = env.db.Pool.QueryRow(context.Background(), `SELECT id FROM payments.payment_refund WHERE code=$1`, code).Scan(&id)
	return fmt.Sprintf("%d", id)
}

// webhookBigOrder checks out qty units with ample stock.
func webhookBigOrder(t *testing.T, env *payEnv, buyerEnv *payEnv, staffEnv *payEnv, key, checkoutKey string, qty int) (string, string, int64) {
	t.Helper()
	prodCode, productID, supplierID := payProduct(t, env)
	payBigStock(t, env, productID, supplierID, int32(qty)+10)
	resp, body := payDoCommerce(t, buyerEnv, http.MethodPost, "/api/v1/commerce/cart/items",
		fmt.Sprintf(`{"product_code":%q,"quantity":%d}`, prodCode, qty), nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("add: %d: %s", resp.StatusCode, string(body))
	}
	resp, body = payDoCommerce(t, buyerEnv, http.MethodPost, "/api/v1/commerce/checkout", "",
		map[string]string{"Idempotency-Key": checkoutKey})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("checkout: %d: %s", resp.StatusCode, string(body))
	}
	var out struct {
		Orders []struct {
			Code string `json:"code"`
		} `json:"orders"`
	}
	_ = json.Unmarshal(body, &out)
	orderCode := out.Orders[0].Code
	resp, body = payDo(t, buyerEnv, http.MethodPost, "/api/v1/payments/intents",
		fmt.Sprintf(`{"order_code":%q,"idem_key":%q}`, orderCode, key), nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("intent: %d: %s", resp.StatusCode, string(body))
	}
	var intent struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(body, &intent)
	_ = staffEnv
	return orderCode, intent.Code, payOrderID(t, env, orderCode)
}

func payBigStock(t *testing.T, env *payEnv, productID, supplierID int64, qty int32) {
	t.Helper()
	payStock(t, env, productID, supplierID, qty)
}

func payIssueInvoiceForRefund(t *testing.T, env *payEnv, staffEnv *payEnv, orderID int64) int64 {
	t.Helper()
	resp, body := payDoCommerce(t, staffEnv, http.MethodPost, "/api/v1/admin/invoices",
		fmt.Sprintf(`{"order_id":%d}`, orderID), nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("invoice: %d: %s", resp.StatusCode, string(body))
	}
	var inv struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(body, &inv)
	var invID int64
	_ = env.db.Pool.QueryRow(context.Background(), `SELECT id FROM commerce.invoice WHERE code=$1`, inv.Code).Scan(&invID)
	resp, _ = payDoCommerce(t, staffEnv, http.MethodPost, fmt.Sprintf("/api/v1/admin/invoices/%d/issue", invID), "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("issue: %d", resp.StatusCode)
	}
	return invID
}

func TestReconciliation_Buckets(t *testing.T) {
	env := webhookEnv(t, 0, map[int64]bool{})
	buyerID := payBuyer(t, env)
	staff := map[int64]bool{888001: true}
	buyerEnv := setupPayEnv(t, buyerID, staff)
	staffEnv := setupPayEnv(t, 888001, staff)

	// Matched order: invoiced, intent succeeded, balance cleared.
	orderCode, intentCode, oid := webhookIntentFixture(t, env, buyerEnv, buyerEnv, "rc-1", "rc-co-1", 2)
	payIssueInvoiceForRefund(t, env, staffEnv, oid)
	completeIntent(t, staffEnv, intentCode)

	// Discrepancy order: two intents for one invoice total; the second
	// succeeds at intent level but cannot allocate (overpayment) → the
	// succeeded-but-unallocated shape reconciliation must flag.
	orderCode2, _, oid2 := webhookIntentFixture(t, env, buyerEnv, buyerEnv, "rc-2", "rc-co-2", 1)
	payIssueInvoiceForRefund(t, env, staffEnv, oid2)
	mkRCIntent := func(key string) string {
		t.Helper()
		resp, body := payDo(t, buyerEnv, http.MethodPost, "/api/v1/payments/intents",
			fmt.Sprintf(`{"order_code":%q,"idem_key":%q}`, orderCode2, key), nil)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("intent %s: %d: %s", key, resp.StatusCode, string(body))
		}
		var out payments_schema.IntentResponse
		_ = json.Unmarshal(body, &out)
		return out.Code
	}
	codeA, codeB := mkRCIntent("rc-intent-2a"), mkRCIntent("rc-intent-2b")
	resp, _ := payDo(t, staffEnv, http.MethodPost, "/api/v1/admin/payments/intents/"+codeA+"/complete", `{}`, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("complete A: %d", resp.StatusCode)
	}
	resp, _ = payDo(t, staffEnv, http.MethodPost, "/api/v1/admin/payments/intents/"+codeB+"/complete", `{}`, nil)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("complete B overpayment: expected 422, got %d", resp.StatusCode)
	}

	// Unmatched webhook: callback for an unknown intent.
	ubody := fmt.Sprintf(`{"event_id":"rc-unknown-1-%s","type":"intent.succeeded","intent_code":"pi_missing"}`, paySuffix())
	uts, usig := signWebhook(t, ubody, 0)
	resp, _ = postWebhook(t, env, "sim", ubody, uts, usig)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("unmatched: %d", resp.StatusCode)
	}

	resp, rbody := payDo(t, staffEnv, http.MethodGet, "/api/v1/admin/payments/reconciliation", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("reconcile: %d: %s", resp.StatusCode, string(rbody))
	}
	var out struct {
		Orders []struct {
			OrderCode      string `json:"order_code"`
			InvoicedTotal  int64  `json:"invoiced_total"`
			PaidApplied    int64  `json:"paid_applied"`
			SucceededTotal int64  `json:"succeeded_total"`
			Status         string `json:"status"`
		} `json:"orders"`
		Unmatched []struct {
			EventID string `json:"event_id"`
		} `json:"unmatched_webhooks"`
	}
	_ = json.Unmarshal(rbody, &out)
	byCode := map[string]string{}
	for _, r := range out.Orders {
		byCode[r.OrderCode] = r.Status
		if r.OrderCode == orderCode && (r.InvoicedTotal != 3000 || r.PaidApplied != 3000 || r.SucceededTotal != 3000) {
			t.Errorf("matched row wrong: %+v", r)
		}
	}
	if byCode[orderCode] != "matched" {
		t.Errorf("expected matched for %s: %+v", orderCode, byCode)
	}
	if byCode[orderCode2] != "discrepancy" {
		t.Errorf("expected discrepancy for %s: %+v", orderCode2, byCode)
	}
	found := false
	for _, u := range out.Unmatched {
		if strings.HasPrefix(u.EventID, "rc-unknown-1-") {
			found = true
		}
	}
	if !found {
		t.Errorf("unmatched webhook missing: %+v", out.Unmatched)
	}
	_ = orderCode
}

func TestCredit_AccountAndCheckoutGate(t *testing.T) {	env := webhookEnv(t, 0, map[int64]bool{})
	buyerID := payBuyer(t, env)
	staff := map[int64]bool{888002: true}
	buyerEnv := setupPayEnv(t, buyerID, staff)
	staffEnv := setupPayEnv(t, 888002, staff)
	// Wire the credit gate for the commerce service used by buyerEnv.
	buyerEnv.commerceSvc.SetCreditChecker(buyerEnv.paymentSvc)

	// No account → gate open.
	prodCode, productID, supplierID := payProduct(t, env)
	payStock(t, env, productID, supplierID, 20)
	resp, body := payDoCommerce(t, buyerEnv, http.MethodPost, "/api/v1/commerce/cart/items",
		fmt.Sprintf(`{"product_code":%q,"quantity":1}`, prodCode), nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("add: %d: %s", resp.StatusCode, string(body))
	}

	// Create a tight credit account: limit 1000, exposure 0.
	resp, body = payDo(t, staffEnv, http.MethodPut, fmt.Sprintf("/api/v1/admin/payments/credit/%d", buyerID),
		`{"credit_limit_minor":1000,"terms":"net_30"}`, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("set credit: %d: %s", resp.StatusCode, string(body))
	}
	var credit struct {
		ExposureMinor  int64 `json:"exposure_minor"`
		AvailableMinor int64 `json:"available_minor"`
	}
	_ = json.Unmarshal(body, &credit)
	if credit.ExposureMinor != 0 || credit.AvailableMinor != 1000 {
		t.Errorf("unexpected credit: %+v", credit)
	}

	// Checkout of 1500 exceeds available 1000 → 422, no order, no reservation.
	resp, body = payDoCommerce(t, buyerEnv, http.MethodPost, "/api/v1/commerce/checkout", "",
		map[string]string{"Idempotency-Key": "credit-gate-1"})
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("gate: expected 422, got %d: %s", resp.StatusCode, string(body))
	}
	var errBody map[string]string
	_ = json.Unmarshal(body, &errBody)
	if errBody["code"] != "credit_limit_exceeded" {
		t.Errorf("expected credit_limit_exceeded, got %q", errBody["code"])
	}
	var orders int
	_ = env.db.Pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM commerce."order" WHERE buyer_id=$1`, buyerID).Scan(&orders)
	if orders != 0 {
		t.Errorf("blocked checkout created %d orders", orders)
	}

	// Raise the limit → checkout succeeds.
	resp, _ = payDo(t, staffEnv, http.MethodPut, fmt.Sprintf("/api/v1/admin/payments/credit/%d", buyerID),
		`{"credit_limit_minor":100000,"terms":"net_30"}`, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("raise: %d", resp.StatusCode)
	}
	resp, body = payDoCommerce(t, buyerEnv, http.MethodPost, "/api/v1/commerce/checkout", "",
		map[string]string{"Idempotency-Key": "credit-gate-2"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("checkout after raise: %d: %s", resp.StatusCode, string(body))
	}

	// Put the account on hold → checkout blocked even with limit.
	resp, _ = payDo(t, staffEnv, http.MethodPut, fmt.Sprintf("/api/v1/admin/payments/credit/%d", buyerID),
		`{"credit_limit_minor":100000,"terms":"net_30","on_hold":true,"hold_reason":"review"}`, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("hold: %d", resp.StatusCode)
	}
	resp, body = payDoCommerce(t, buyerEnv, http.MethodPost, "/api/v1/commerce/cart/items",
		fmt.Sprintf(`{"product_code":%q,"quantity":1}`, prodCode), nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("re-add: %d: %s", resp.StatusCode, string(body))
	}
	resp, body = payDoCommerce(t, buyerEnv, http.MethodPost, "/api/v1/commerce/checkout", "",
		map[string]string{"Idempotency-Key": "credit-gate-3"})
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("hold gate: expected 422, got %d: %s", resp.StatusCode, string(body))
	}
	_ = json.Unmarshal(body, &errBody)
	if errBody["code"] != "credit_hold" {
		t.Errorf("expected credit_hold, got %q", errBody["code"])
	}

	// Missing account reads 404.
	resp, _ = payDo(t, staffEnv, http.MethodGet, "/api/v1/admin/payments/credit/999999999", "", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("missing credit: expected 404, got %d", resp.StatusCode)
	}
}

func TestStatements_BuyerAndAdmin(t *testing.T) {
	env := webhookEnv(t, 0, map[int64]bool{})
	buyerID := payBuyer(t, env)
	staff := map[int64]bool{888004: true}
	buyerEnv := setupPayEnv(t, buyerID, staff)
	staffEnv := setupPayEnv(t, 888004, staff)
	orderCode, intentCode, oid := webhookIntentFixture(t, env, buyerEnv, buyerEnv, "st-1", "st-co-1", 2)
	payIssueInvoiceForRefund(t, env, staffEnv, oid)
	completeIntent(t, staffEnv, intentCode)

	period := time.Now().UTC().Format("2006-01")

	// Buyer statement: invoiced 3000, paid 3000, closing 0.
	resp, body := payDo(t, buyerEnv, http.MethodGet, "/api/v1/payments/statements?period="+period, "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("buyer statement: %d: %s", resp.StatusCode, string(body))
	}
	var stmt struct {
		BuyerID      int64 `json:"buyer_id"`
		OpeningMinor int64 `json:"opening_minor"`
		InvoicedMinor int64 `json:"invoiced_minor"`
		PaidMinor    int64 `json:"paid_minor"`
		ClosingMinor int64 `json:"closing_minor"`
		Lines        []struct {
			Kind   string `json:"kind"`
			Amount int64  `json:"amount_minor"`
		} `json:"lines"`
	}
	_ = json.Unmarshal(body, &stmt)
	if stmt.BuyerID != buyerID || stmt.InvoicedMinor != 3000 || stmt.PaidMinor != 3000 || stmt.ClosingMinor != 0 {
		t.Errorf("unexpected statement: %+v", stmt)
	}
	kinds := map[string]bool{}
	for _, l := range stmt.Lines {
		kinds[l.Kind] = true
	}
	if !kinds["invoice"] || !kinds["payment"] {
		t.Errorf("expected invoice+payment lines: %+v", stmt.Lines)
	}

	// Admin statement for the buyer matches.
	resp, body = payDo(t, staffEnv, http.MethodGet,
		fmt.Sprintf("/api/v1/admin/payments/statements?buyer_id=%d&period=%s", buyerID, period), "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admin statement: %d: %s", resp.StatusCode, string(body))
	}
	var adminStmt struct {
		ClosingMinor int64 `json:"closing_minor"`
	}
	_ = json.Unmarshal(body, &adminStmt)
	if adminStmt.ClosingMinor != 0 {
		t.Errorf("admin closing mismatch: %+v", adminStmt)
	}

	// Malformed period → 400 on both surfaces.
	resp, _ = payDo(t, buyerEnv, http.MethodGet, "/api/v1/payments/statements?period=not-a-period", "", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("bad period buyer: expected 400, got %d", resp.StatusCode)
	}
	resp, _ = payDo(t, staffEnv, http.MethodGet,
		fmt.Sprintf("/api/v1/admin/payments/statements?buyer_id=%d&period=%s", buyerID, "xx"), "", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("bad period admin: expected 400, got %d", resp.StatusCode)
	}

	// Buyer cannot read admin statements → 403.
	resp, _ = payDo(t, buyerEnv, http.MethodGet,
		fmt.Sprintf("/api/v1/admin/payments/statements?buyer_id=%d&period=%s", buyerID, period), "", nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("buyer admin statement: expected 403, got %d", resp.StatusCode)
	}
	_ = orderCode
	_ = oid
}
