package service

// Voucher plumbing shared by cart apply, quote, and checkout. One voucher
// per cart (no stacking — the documented precedence rule). Validation lives
// in the promotions module; commerce owns cart state, discount math, and
// redemption recording.

import (
	"context"
	"fmt"

	"github.com/atlas-platform/backend/internal/modules/commerce/repository"
	"github.com/atlas-platform/backend/internal/modules/commerce/schema"
	promotions_repo "github.com/atlas-platform/backend/internal/modules/promotions/repository"
)

// voucherDeal is a validated promotion plus its code, before subtotal math.
type voucherDeal struct {
	promo *promotions_repo.Promotion
	code  string
}

// resolveVoucher validates the cart's applied code strictly (errors
// propagate to checkout/apply). Quote uses it advisoryly and treats any
// error as no voucher.
func (s *CommerceService) resolveVoucher(ctx context.Context, buyerID int64, cart repository.Cart) (*voucherDeal, error) {
	if s.promotionsSvc == nil || cart.VoucherCode == nil || *cart.VoucherCode == "" {
		return nil, nil
	}
	promo, err := s.promotionsSvc.ValidateVoucher(ctx, buyerID, *cart.VoucherCode)
	if err != nil {
		return nil, err
	}
	return &voucherDeal{promo: promo, code: promo.Code}, nil
}

// splitDiscount pro-ratas a cart-level discount across per-supplier
// subtotals. Shares are floored; the remainder goes to the first (largest,
// order-stable) group so shares always sum exactly to the discount.
func splitDiscount(subtotals []int64, discount int64) []int64 {
	shares := make([]int64, len(subtotals))
	if discount <= 0 {
		return shares
	}
	var total, first int64
	for i, s := range subtotals {
		total += s
		if i == 0 || s > subtotals[first] {
			first = int64(i)
		}
	}
	if total <= 0 {
		return shares
	}
	var assigned int64
	for i, s := range subtotals {
		shares[i] = discount * s / total
		assigned += shares[i]
	}
	shares[first] += discount - assigned
	return shares
}

// ApplyVoucher validates a code and applies it to the buyer's cart,
// replacing any previously applied voucher. Last write wins.
func (s *CommerceService) ApplyVoucher(ctx context.Context, buyerID int64, code string) (*schema.CartResponse, error) {
	if s.promotionsSvc == nil {
		return nil, fmt.Errorf("promotions not configured")
	}
	cart, err := s.getOrCreateCart(ctx, buyerID)
	if err != nil {
		return nil, fmt.Errorf("get or create cart: %w", err)
	}
	promo, err := s.promotionsSvc.ValidateVoucher(ctx, buyerID, code)
	if err != nil {
		return nil, err
	}
	if _, err := s.repo.SetCartVoucher(ctx, cart.ID, promo.Code); err != nil {
		return nil, fmt.Errorf("apply voucher: %w", err)
	}
	return s.GetCart(ctx, buyerID)
}
