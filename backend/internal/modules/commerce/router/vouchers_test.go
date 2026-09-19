package router_test

// TASK-009 voucher integration tests: apply flows, quote math, checkout
// discounts + redemption + idempotency, budget races, isolation, RBAC.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/atlas-platform/backend/internal/modules/commerce/schema"
	commerce_service "github.com/atlas-platform/backend/internal/modules/commerce/service"
	inventory_service "github.com/atlas-platform/backend/internal/modules/inventory/service"
	pricing_service "github.com/atlas-platform/backend/internal/modules/pricing/service"
	promotions_schema "github.com/atlas-platform/backend/internal/modules/promotions/schema"
	promotions_service "github.com/atlas-platform/backend/internal/modules/promotions/service"
)

func promoSvc(t *testing.T, env *testEnv) *promotions_service.PromotionService {
	t.Helper()
	return promotions_service.NewPromotionService(env.db, nil)
}

// makePromo creates and publishes a promotion with optional window/budget.
// The returned code is suffixed for cross-iteration uniqueness.
func makePromo(t *testing.T, env *testEnv, base, kind string, value int64, maxRed *int, validFrom, validTo *string) string {
	t.Helper()
	code := base + "-" + uniqueSuffix()
	svc := promoSvc(t, env)
	req := promotions_schema.CreatePromotionRequest{
		Code: code, Name: code, Kind: kind, ValueMinor: value,
		ValidFrom: validFrom, ValidTo: validTo, MaxRedemptions: maxRed,
	}
	p, err := svc.CreatePromotion(context.Background(), req)
	if err != nil {
		t.Fatalf("create promo: %v", err)
	}
	if _, err := svc.PublishPromotion(context.Background(), promoID(t, env, p.Code)); err != nil {
		t.Fatalf("publish promo: %v", err)
	}
	return code
}

func intPtr(i int) *int { return &i }

func strPtr(s string) *string { return &s }

func promoID(t *testing.T, env *testEnv, code string) int64 {
	t.Helper()
	var id int64
	if err := env.db.Pool.QueryRow(context.Background(), `SELECT id FROM promotions.promotion WHERE code=$1`, code).Scan(&id); err != nil {
		t.Fatalf("promo id: %v", err)
	}
	return id
}

func applyVoucher(t *testing.T, env *testEnv, code string, want int) *schema.CartResponse {
	t.Helper()
	resp, body := doJSON(t, env, http.MethodPost, "/api/v1/commerce/cart/voucher",
		fmt.Sprintf(`{"code":%q}`, code), nil)
	if resp.StatusCode != want {
		t.Fatalf("apply voucher: expected %d, got %d: %s", want, resp.StatusCode, string(body))
	}
	if want != http.StatusOK {
		return nil
	}
	var cart schema.CartResponse
	_ = json.Unmarshal(body, &cart)
	return &cart
}

func TestVoucher_ApplyFlow(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	buyerID := createTestBuyer(t, env)
	prodCode, _, _ := createTestProduct(t, env)
	buyerEnv := setupEnv(t, buyerID, withPrincipalMiddleware(buyerID), nopMiddleware)
	addItem(t, buyerEnv, prodCode, 1)

	// Unknown code → 404. Empty code → 400.
	applyVoucher(t, buyerEnv, "nope-missing", http.StatusNotFound)
	resp, body := doJSON(t, buyerEnv, http.MethodPost, "/api/v1/commerce/cart/voucher", `{"code":""}`, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("empty code: %d: %s", resp.StatusCode, string(body))
	}

	// Draft promo → 422 invalid.
	svc := promoSvc(t, env)
	draftCode := "draft-1-" + uniqueSuffix()
	draft, err := svc.CreatePromotion(context.Background(), promotions_schema.CreatePromotionRequest{
		Code: draftCode, Name: draftCode, Kind: "percent", ValueMinor: 10,
	})
	if err != nil {
		t.Fatalf("draft: %v", err)
	}
	_ = draft
	applyVoucher(t, buyerEnv, draftCode, http.StatusUnprocessableEntity)

	// Publish → apply works, cart echoes the code.
	ten := makePromo(t, env, "ten-1", "percent", 10, nil, nil, nil)
	cart := applyVoucher(t, buyerEnv, ten, http.StatusOK)
	if cart.VoucherCode == nil || *cart.VoucherCode != ten {
		t.Fatalf("cart missing voucher: %+v", cart)
	}

	// Replace with another promo (last write wins).
	five := makePromo(t, env, "five-1", "fixed", 500, nil, nil, nil)
	cart = applyVoucher(t, buyerEnv, five, http.StatusOK)
	if cart.VoucherCode == nil || *cart.VoucherCode != five {
		t.Fatalf("voucher not replaced: %+v", cart)
	}
}

func TestVoucher_QuoteDiscount(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	buyerID := createTestBuyer(t, env)
	prodCode, _, _ := createTestProduct(t, env)
	buyerEnv := setupEnv(t, buyerID, withPrincipalMiddleware(buyerID), nopMiddleware)
	addItem(t, buyerEnv, prodCode, 2) // subtotal 3000

	quote := func() schema.CartQuoteResponse {
		t.Helper()
		resp, body := doJSON(t, buyerEnv, http.MethodPost, "/api/v1/commerce/cart/quote", "", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("quote: %d: %s", resp.StatusCode, string(body))
		}
		var q schema.CartQuoteResponse
		_ = json.Unmarshal(body, &q)
		return q
	}

	// No voucher → zero discount.
	if q := quote(); q.DiscountsMinor != 0 || q.TotalMinor != 3000 {
		t.Fatalf("no-voucher quote: %+v", q)
	}

	// Percent 10 → 300 off.
	ten := makePromo(t, env, "ten-2", "percent", 10, nil, nil, nil)
	applyVoucher(t, buyerEnv, ten, http.StatusOK)
	if q := quote(); q.DiscountsMinor != 300 || q.TotalMinor != 2700 {
		t.Fatalf("percent quote: %+v", q)
	} else if len(q.Suppliers) != 1 || q.Suppliers[0].DiscountsMinor != 300 {
		t.Fatalf("group discount: %+v", q)
	}

	// Fixed 5000 capped at subtotal → 3000 off, total 0.
	big := makePromo(t, env, "big-2", "fixed", 5000, nil, nil, nil)
	applyVoucher(t, buyerEnv, big, http.StatusOK)
	if q := quote(); q.DiscountsMinor != 3000 || q.TotalMinor != 0 {
		t.Fatalf("capped quote: %+v", q)
	}
}

func TestVoucher_Checkout(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	buyerID := createTestBuyer(t, env)
	prodCode, productID, supplierID := createTestProduct(t, env)
	addStock(t, env, productID, supplierID, 10, false)
	buyerEnv := setupEnv(t, buyerID, withPrincipalMiddleware(buyerID), nopMiddleware)
	addItem(t, buyerEnv, prodCode, 2)

	ten3 := makePromo(t, env, "ten-3", "percent", 10, nil, nil, nil)
	applyVoucher(t, buyerEnv, ten3, http.StatusOK)

	resp, body := doCheckout(t, buyerEnv, "vch-key-1", "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("checkout: %d: %s", resp.StatusCode, string(body))
	}
	out := decodeCheckout(t, body)
	if len(out.Orders) != 1 {
		t.Fatalf("orders: %+v", out)
	}
	ord := out.Orders[0]
	if ord.SubtotalMinor != 3000 || ord.DiscountsMinor != 300 || ord.TotalMinor != 2700 {
		t.Errorf("order money wrong: %+v", ord)
	}

	// Redemption recorded exactly once; budget consumed.
	var redemptions, budget int
	_ = env.db.Pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM promotions.voucher_redemption WHERE buyer_id=$1`, buyerID).Scan(&redemptions)
	_ = env.db.Pool.QueryRow(context.Background(), `SELECT redeemed_count FROM promotions.promotion WHERE code=$1`, ten3).Scan(&budget)
	if redemptions != 1 || budget != 1 {
		t.Errorf("redemptions=%d budget=%d, want 1/1", redemptions, budget)
	}

	// Voucher cleared from cart.
	resp, body = doJSON(t, buyerEnv, http.MethodGet, "/api/v1/commerce/cart", "", nil)
	var cart schema.CartResponse
	_ = json.Unmarshal(body, &cart)
	if cart.VoucherCode != nil {
		t.Errorf("voucher not cleared: %+v", cart.VoucherCode)
	}

	// Replay same key → identical totals, no second redemption.
	resp, body = doCheckout(t, buyerEnv, "vch-key-1", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("replay: %d: %s", resp.StatusCode, string(body))
	}
	if second := decodeCheckout(t, body); !second.Replayed || second.Orders[0].TotalMinor != 2700 {
		t.Errorf("replay mismatch: %+v", second)
	}
	_ = env.db.Pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM promotions.voucher_redemption WHERE buyer_id=$1`, buyerID).Scan(&redemptions)
	if redemptions != 1 {
		t.Errorf("replay double-redeemed: %d", redemptions)
	}
}

func TestVoucher_StaleAndExpired(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	buyerID := createTestBuyer(t, env)
	prodCode, productID, supplierID := createTestProduct(t, env)
	addStock(t, env, productID, supplierID, 10, false)
	buyerEnv := setupEnv(t, buyerID, withPrincipalMiddleware(buyerID), nopMiddleware)
	addItem(t, buyerEnv, prodCode, 1)

	// Expired window → 422 expired.
	vf, vt := "2020-01-01T00:00:00Z", "2020-02-01T00:00:00Z"
	oldCode := makePromo(t, env, "old-1", "fixed", 100, nil, &vf, &vt)
	applyVoucher(t, buyerEnv, oldCode, http.StatusUnprocessableEntity)

	// Apply valid, then archive → checkout rejects strictly.
	goneCode := makePromo(t, env, "gone-1", "fixed", 100, nil, nil, nil)
	applyVoucher(t, buyerEnv, goneCode, http.StatusOK)
	svc := promoSvc(t, env)
	st := "archived"
	if _, err := svc.UpdatePromotion(context.Background(), promoID(t, env, goneCode),
		promotions_schema.UpdatePromotionRequest{Status: &st}); err != nil {
		t.Fatalf("archive: %v", err)
	}
	resp, body := doCheckout(t, buyerEnv, "vch-stale-1", "")
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("stale checkout: expected 422, got %d: %s", resp.StatusCode, string(body))
	}
	if code := decodeErr(t, body)["code"]; code != "voucher_invalid" {
		t.Errorf("expected voucher_invalid, got %q", code)
	}

	// Already-redeemed code is rejected at apply time with a clear reason,
	// and checkout without it still succeeds (no stale voucher state).
	addItem(t, buyerEnv, prodCode, 1)
	onceCode := makePromo(t, env, "once-1", "fixed", 100, nil, nil, nil)
	applyVoucher(t, buyerEnv, onceCode, http.StatusOK)
	resp, body = doCheckout(t, buyerEnv, "vch-once-1", "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("first: %d: %s", resp.StatusCode, string(body))
	}
	addItem(t, buyerEnv, prodCode, 1)
	applyVoucher(t, buyerEnv, onceCode, http.StatusUnprocessableEntity)
	resp, body = doCheckout(t, buyerEnv, "vch-once-2", "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("second without voucher: expected 201, got %d: %s", resp.StatusCode, string(body))
	}
	if out := decodeCheckout(t, body); out.Orders[0].DiscountsMinor != 0 {
		t.Errorf("expected no discount, got %+v", out.Orders[0])
	}
}

func TestVoucher_BudgetRace(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	mkBuyer := func() (*testEnv, int64) {
		t.Helper()
		buyerID := createTestBuyer(t, env)
		prodCode, productID, supplierID := createTestProduct(t, env)
		addStock(t, env, productID, supplierID, 10, false)
		be := setupEnv(t, buyerID, withPrincipalMiddleware(buyerID), nopMiddleware)
		addItem(t, be, prodCode, 1)
		return be, buyerID
	}
	envA, buyerA := mkBuyer()
	envB, _ := mkBuyer()

	// One promo, budget 1, shared by both buyers.
	svc := promoSvc(t, env)
	raceCode := "race-1-" + uniqueSuffix()
	p, err := svc.CreatePromotion(context.Background(), promotions_schema.CreatePromotionRequest{
		Code: raceCode, Name: raceCode, Kind: "fixed", ValueMinor: 100, MaxRedemptions: intPtr(1),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := svc.PublishPromotion(context.Background(), promoID(t, env, p.Code)); err != nil {
		t.Fatalf("publish: %v", err)
	}
	applyVoucher(t, envA, raceCode, http.StatusOK)
	applyVoucher(t, envB, raceCode, http.StatusOK)

	var stA, stB int
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		resp, _ := doCheckout(t, envA, "race-key-a", "")
		stA = resp.StatusCode
	}()
	go func() {
		defer wg.Done()
		resp, _ := doCheckout(t, envB, "race-key-b", "")
		stB = resp.StatusCode
	}()
	wg.Wait()

	wins := 0
	for _, st := range []int{stA, stB} {
		switch st {
		case http.StatusCreated:
			wins++
		case http.StatusUnprocessableEntity:
		default:
			t.Errorf("unexpected status %d", st)
		}
	}
	if wins != 1 {
		t.Errorf("expected exactly 1 winner, got %d (%d/%d)", wins, stA, stB)
	}
	var budget int
	_ = env.db.Pool.QueryRow(context.Background(), `SELECT redeemed_count FROM promotions.promotion WHERE code=$1`, raceCode).Scan(&budget)
	if budget != 1 {
		t.Errorf("budget consumed %d times, want 1", budget)
	}
	var ordersA int
	_ = env.db.Pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM commerce."order" WHERE buyer_id=$1`, buyerA).Scan(&ordersA)
	if ordersA > 1 {
		t.Errorf("buyer A has %d orders, want ≤1", ordersA)
	}
}

func TestVoucher_RBAC(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	staff := map[int64]bool{}
	staffID := createTestBuyer(t, env)
	staff[staffID] = true
	buyerID := createTestBuyer(t, env)
	buyerEnv := setupEnv(t, buyerID, withPrincipalMiddleware(buyerID), staffMiddleware(staff))

	for _, p := range [][2]string{
		{http.MethodGet, "/api/v1/admin/promotions"},
		{http.MethodPost, "/api/v1/admin/promotions"},
		{http.MethodGet, "/api/v1/admin/promotions/reports"},
	} {
		resp, _ := doJSON(t, buyerEnv, p[0], p[1], "", nil)
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("%s %s: expected 403, got %d", p[0], p[1], resp.StatusCode)
		}
	}
}

func TestVoucher_EventPublished(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	buyerID := createTestBuyer(t, env)
	prodCode, productID, supplierID := createTestProduct(t, env)
	addStock(t, env, productID, supplierID, 10, false)

	pub := &recordingCheckoutPublisher{}
	svc := commerce_service.NewCommerceService(env.db,
		pricing_service.NewServices(env.db).PriceList,
		inventory_service.NewInventoryService(env.db, nil),
		pub,
		promotions_service.NewPromotionService(env.db, nil))
	ctx := context.Background()

	evtCode := makePromo(t, env, "evt-1", "fixed", 200, nil, nil, nil)
	if _, err := svc.AddCartItem(ctx, buyerID, schema.AddCartItemRequest{ProductCode: prodCode, Quantity: 1}); err != nil {
		t.Fatalf("add: %v", err)
	}
	if _, err := svc.ApplyVoucher(ctx, buyerID, evtCode); err != nil {
		t.Fatalf("apply: %v", err)
	}
	result, err := svc.Checkout(ctx, buyerID, "evt-key-1")
	if err != nil {
		t.Fatalf("checkout: %v", err)
	}
	if result.Response.Orders[0].DiscountsMinor != 200 {
		t.Fatalf("expected 200 discount: %+v", result.Response.Orders[0])
	}
	var placed, redeemed int
	pub.mu.Lock()
	for _, k := range pub.keys {
		switch k {
		case "commerce.order.placed":
			placed++
		case "promotions.voucher.redeemed":
			redeemed++
		}
	}
	pub.mu.Unlock()
	if placed != 1 || redeemed != 1 {
		t.Errorf("expected 1 placed + 1 redeemed event, got %d/%d", placed, redeemed)
	}

	// Replay publishes nothing.
	if _, err := svc.Checkout(ctx, buyerID, "evt-key-1"); err != nil {
		t.Fatalf("replay: %v", err)
	}
	if got := pub.count(); got != 2 {
		t.Errorf("replay published duplicate events: %d", got)
	}
}
