package router_test

// TASK-006A payments probes: intent idempotency races, concurrent
// settlement, double-payment allocation races.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"

	payments_service "github.com/atlas-platform/backend/internal/modules/payments/service"
	payments_schema "github.com/atlas-platform/backend/internal/modules/payments/schema"
)

type recordingPayPublisher struct {
	mu   sync.Mutex
	keys []string
}

func (p *recordingPayPublisher) Publish(ctx context.Context, routingKey string, payload interface{}) error {
	p.mu.Lock()
	p.keys = append(p.keys, routingKey)
	p.mu.Unlock()
	return nil
}

func (p *recordingPayPublisher) count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.keys)
}

// payIssueInvoice drafts and issues an invoice covering the order total.
func payIssueInvoice(t *testing.T, staffEnv *payEnv, env *payEnv, orderID int64) {
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
	resp, body = payDoCommerce(t, staffEnv, http.MethodPost, fmt.Sprintf("/api/v1/admin/invoices/%d/issue", invID), "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("issue: %d: %s", resp.StatusCode, string(body))
	}
}

// payFixtureOrder checks out stock-backed goods and returns the order code.
func payFixtureOrder(t *testing.T, env *payEnv, buyerEnv *payEnv, key string, qty int) string {
	t.Helper()
	prodCode, productID, supplierID := payProduct(t, env)
	payStock(t, env, productID, supplierID, 20)
	resp, body := payDoCommerce(t, buyerEnv, http.MethodPost, "/api/v1/commerce/cart/items",
		fmt.Sprintf(`{"product_code":%q,"quantity":%d}`, prodCode, qty), nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("add: %d: %s", resp.StatusCode, string(body))
	}
	resp, body = payDoCommerce(t, buyerEnv, http.MethodPost, "/api/v1/commerce/checkout", "",
		map[string]string{"Idempotency-Key": key})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("checkout: %d: %s", resp.StatusCode, string(body))
	}
	var out struct {
		Orders []struct {
			Code string `json:"code"`
		} `json:"orders"`
	}
	_ = json.Unmarshal(body, &out)
	return out.Orders[0].Code
}

func payIntent(t *testing.T, env *payEnv, orderCode, key string) (int, []byte) {
	t.Helper()
	resp, body := payDo(t, env, http.MethodPost, "/api/v1/payments/intents",
		fmt.Sprintf(`{"order_code":%q,"idem_key":%q}`, orderCode, key), nil)
	return resp.StatusCode, body
}

// T7: concurrent settles of one intent — exactly one succeeds.
func TestAudit006A_ConcurrentMark(t *testing.T) {
	env := setupPayEnv(t, 0, map[int64]bool{})
	buyerID := payBuyer(t, env)
	staff := map[int64]bool{777001: true}
	buyerEnv := setupPayEnv(t, buyerID, staff)
	staffEnv := setupPayEnv(t, 777001, staff)
	orderCode := payFixtureOrder(t, env, buyerEnv, "amark-1", 1)
	oid := payOrderID(t, env, orderCode)
	payIssueInvoice(t, staffEnv, env, oid)

	st, body := payIntent(t, buyerEnv, orderCode, "amark-intent-1")
	if st != http.StatusCreated {
		t.Fatalf("intent: %d: %s", st, string(body))
	}
	var intent payments_schema.IntentResponse
	_ = json.Unmarshal(body, &intent)

	const n = 6
	var statuses [n]int
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			resp, _ := payDo(t, staffEnv, http.MethodPost,
				"/api/v1/admin/payments/intents/"+intent.Code+"/complete", `{}`, nil)
			statuses[i] = resp.StatusCode
		}(i)
	}
	close(start)
	wg.Wait()

	ok, conflict := 0, 0
	for i, s := range statuses {
		switch s {
		case http.StatusOK:
			ok++
		case http.StatusConflict:
			conflict++
		default:
			t.Errorf("req %d: unexpected %d", i, s)
		}
	}
	if ok != 1 || conflict != n-1 {
		t.Errorf("expected 1 ok + %d conflicts, got %v", n-1, statuses)
	}
	var attempts int
	_ = env.db.Pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM payments.payment_attempt WHERE intent_id=(SELECT id FROM payments.payment_intent WHERE code=$1)`, intent.Code).Scan(&attempts)
	if attempts != 1 {
		t.Errorf("expected exactly 1 attempt row, got %d", attempts)
	}
}

// T8: concurrent same-key same-payload intents — one row, one 201.
func TestAudit006A_ConcurrentSameKeyIntent(t *testing.T) {
	env := setupPayEnv(t, 0, map[int64]bool{})
	buyerID := payBuyer(t, env)
	staff := map[int64]bool{}
	buyerEnv := setupPayEnv(t, buyerID, staff)
	orderCode := payFixtureOrder(t, env, buyerEnv, "akey-1", 1)

	const n = 8
	var statuses [n]int
	var codes [n]string
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			st, body := payIntent(t, buyerEnv, orderCode, "race-key-8")
			statuses[i] = st
			var out payments_schema.IntentResponse
			if json.Unmarshal(body, &out) == nil {
				codes[i] = out.Code
			}
		}(i)
	}
	close(start)
	wg.Wait()

	created := 0
	for i, s := range statuses {
		switch s {
		case http.StatusCreated:
			created++
		case http.StatusOK:
			if codes[i] != codes[0] && codes[0] != "" {
				// codes[0] may itself be a replay; compare against any created code below.
			}
		default:
			t.Errorf("req %d: unexpected %d", i, s)
		}
	}
	if created != 1 {
		t.Errorf("expected exactly 1 created, got %d (%v)", created, statuses)
	}
	first := ""
	for _, c := range codes {
		if c == "" {
			t.Fatalf("empty intent code in response")
		}
		if first == "" {
			first = c
		} else if c != first {
			t.Errorf("divergent intent codes: %q vs %q", first, c)
		}
	}
	var rows int
	_ = env.db.Pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM payments.payment_intent WHERE buyer_id=$1 AND idem_key='race-key-8'`, buyerID).Scan(&rows)
	if rows != 1 {
		t.Errorf("expected 1 intent row, got %d", rows)
	}
}

// T9: concurrent same-key different-payload — exactly one wins, rest 409.
func TestAudit006A_ConcurrentKeyConflict(t *testing.T) {
	env := setupPayEnv(t, 0, map[int64]bool{})
	buyerID := payBuyer(t, env)
	staff := map[int64]bool{}
	buyerEnv := setupPayEnv(t, buyerID, staff)
	orderA := payFixtureOrder(t, env, buyerEnv, "akey-a", 1)
	orderB := payFixtureOrder(t, env, buyerEnv, "akey-b", 1)

	const n = 8
	var statuses [n]int
	var codes [n]string
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			target := orderA
			if i%2 == 1 {
				target = orderB
			}
			st, body := payIntent(t, buyerEnv, target, "race-key-9")
			statuses[i] = st
			var out payments_schema.IntentResponse
			if json.Unmarshal(body, &out) == nil {
				codes[i] = out.Code
			}
		}(i)
	}
	close(start)
	wg.Wait()

	// Exactly one insert wins. Same-payload contenders replay it (200 with
	// the identical code); different-payload contenders get 409.
	created := 0
	winner := ""
	for i, s := range statuses {
		switch s {
		case http.StatusCreated:
			created++
			winner = codes[i]
		case http.StatusOK:
			if codes[i] == "" {
				t.Errorf("req %d: empty replay code", i)
			}
		case http.StatusConflict:
		default:
			t.Errorf("req %d: unexpected status %d", i, s)
		}
	}
	if created != 1 {
		t.Errorf("expected exactly 1 created, got %d (%v)", created, statuses)
	}
	for i, s := range statuses {
		if s == http.StatusOK && codes[i] != winner {
			t.Errorf("req %d: replay code %q != winner %q", i, codes[i], winner)
		}
	}
	var rows int
	_ = env.db.Pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM payments.payment_intent WHERE buyer_id=$1 AND idem_key='race-key-9'`, buyerID).Scan(&rows)
	if rows != 1 {
		t.Errorf("expected 1 intent row, got %d", rows)
	}
}

// T10: concurrent double-payment against one invoice — one succeeds, the
// other is rejected as overpayment; balance never negative.
func TestAudit006A_ConcurrentDoublePayment(t *testing.T) {
	env := setupPayEnv(t, 0, map[int64]bool{})
	buyerID := payBuyer(t, env)
	staff := map[int64]bool{777002: true}
	buyerEnv := setupPayEnv(t, buyerID, staff)
	staffEnv := setupPayEnv(t, 777002, staff)
	orderCode := payFixtureOrder(t, env, buyerEnv, "apay-1", 2) // total 3000
	oid := payOrderID(t, env, orderCode)

	// Issue one invoice for the full total.
	resp, body := payDoCommerce(t, staffEnv, http.MethodPost, "/api/v1/admin/invoices",
		fmt.Sprintf(`{"order_id":%d}`, oid), nil)
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

	mkCode := func(key string) string {
		t.Helper()
		st, b := payIntent(t, buyerEnv, orderCode, key)
		if st != http.StatusCreated {
			t.Fatalf("intent %s: %d: %s", key, st, string(b))
		}
		var out payments_schema.IntentResponse
		_ = json.Unmarshal(b, &out)
		return out.Code
	}
	codeA, codeB := mkCode("apay-a"), mkCode("apay-b")

	var stA, stB int
	var wg sync.WaitGroup
	start := make(chan struct{})
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		r, _ := payDo(t, staffEnv, http.MethodPost, "/api/v1/admin/payments/intents/"+codeA+"/complete", `{}`, nil)
		stA = r.StatusCode
	}()
	go func() {
		defer wg.Done()
		<-start
		r, _ := payDo(t, staffEnv, http.MethodPost, "/api/v1/admin/payments/intents/"+codeB+"/complete", `{}`, nil)
		stB = r.StatusCode
	}()
	close(start)
	wg.Wait()

	ok := 0
	for _, s := range []int{stA, stB} {
		switch s {
		case http.StatusOK:
			ok++
		case http.StatusUnprocessableEntity:
		default:
			t.Errorf("unexpected status %d", s)
		}
	}
	if ok != 1 {
		t.Errorf("expected exactly 1 successful settlement, got %d (%d/%d)", ok, stA, stB)
	}
	var balance int64
	_ = env.db.Pool.QueryRow(context.Background(), `SELECT balance_minor FROM commerce.invoice WHERE id=$1`, invID).Scan(&balance)
	if balance != 0 {
		t.Errorf("expected balance 0, got %d", balance)
	}
	var neg int
	_ = env.db.Pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM commerce.invoice WHERE balance_minor < 0`).Scan(&neg)
	if neg != 0 {
		t.Errorf("negative invoice balance detected")
	}
}

// T11: succeeded intent publishes exactly one event, even across retries.
func TestAudit006A_IntentEventOnce(t *testing.T) {
	env := setupPayEnv(t, 0, map[int64]bool{})
	buyerID := payBuyer(t, env)
	staff := map[int64]bool{777003: true}
	buyerEnv := setupPayEnv(t, buyerID, staff)
	staffEnv := setupPayEnv(t, 777003, staff)
	orderCode := payFixtureOrder(t, env, buyerEnv, "aevt-1", 1)
	payIssueInvoice(t, staffEnv, env, payOrderID(t, env, orderCode))

	pub := &recordingPayPublisher{}
	svc := payments_service.NewPaymentService(env.db, buyerEnv.commerceSvc, pub)
	st, body := payIntent(t, buyerEnv, orderCode, "aevt-intent")
	if st != http.StatusCreated {
		t.Fatalf("intent: %d: %s", st, string(body))
	}
	var intent payments_schema.IntentResponse
	_ = json.Unmarshal(body, &intent)

	if _, err := svc.MarkIntent(context.Background(), 4242, intent.Code, true, "wire"); err != nil {
		t.Fatalf("mark: %v", err)
	}
	// Second mark is terminal-rejected and must not publish again.
	if _, err := svc.MarkIntent(context.Background(), 4242, intent.Code, true, "again"); err == nil {
		t.Fatalf("expected terminal error on re-mark")
	}
	if got := pub.count(); got != 1 {
		t.Errorf("expected exactly 1 intent event, got %d", got)
	}
}
