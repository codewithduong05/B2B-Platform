package service

// TASK-007: payment reversal allocation and credit exposure reads.
// ReversePayment restores invoice balances newest-first (LIFO undo of the
// oldest-first RecordPayment application), capped at each invoice total.

import (
	"context"
	"fmt"

	"github.com/atlas-platform/backend/internal/database"
	"github.com/atlas-platform/backend/internal/modules/commerce/repository"
)

func (s *CommerceService) ReversePayment(ctx context.Context, orderID, amountMinor int64) (int64, error) {
	if amountMinor <= 0 {
		return amountMinor, ErrInvalidQuantity
	}
	var remaining int64
	err := s.db.WithTx(ctx, func(tx *database.Tx) error {
		txRepo := repository.NewCommerceRepositoryWithTx(tx)
		// Locked reads: concurrent reversals serialize on the same rows.
		invoices, err := txRepo.ListInvoicesForUpdate(ctx, orderID)
		if err != nil {
			return fmt.Errorf("list invoices: %w", err)
		}
		remaining = amountMinor
		for _, inv := range invoices {
			if remaining <= 0 {
				break
			}
			if inv.Status != "issued" {
				continue
			}
			room := inv.TotalMinor - inv.BalanceMinor
			if room <= 0 {
				continue
			}
			restore := remaining
			if restore > room {
				restore = room
			}
			if _, err := txRepo.RestoreInvoiceBalance(ctx, inv.ID, restore); err != nil {
				return fmt.Errorf("restore invoice balance: %w", err)
			}
			remaining -= restore
		}
		if remaining > 0 {
			return ErrOverpayment
		}
		return nil
	})
	if err != nil {
		return amountMinor, err
	}
	return 0, nil
}

// OrderMoney returns invoiced totals and applied payments for one order
// (issued invoices only). Read surface for the payments module.
func (s *CommerceService) OrderMoney(ctx context.Context, orderID int64) (invoiced, paid int64, err error) {
	invoices, err := s.repo.ListInvoices(ctx, orderID, 0, 0)
	if err != nil {
		return 0, 0, fmt.Errorf("list invoices: %w", err)
	}
	for _, inv := range invoices {
		if inv.Status != "issued" {
			continue
		}
		invoiced += inv.TotalMinor
		paid += inv.TotalMinor - inv.BalanceMinor
	}
	return invoiced, paid, nil
}

// OutstandingExposure sums issued invoice balances across the buyer's
// orders. Used by the credit gate; always derived, never stored.
func (s *CommerceService) OutstandingExposure(ctx context.Context, buyerID int64) (int64, error) {
	orders, err := s.repo.ListOrdersByBuyer(ctx, buyerID)
	if err != nil {
		return 0, fmt.Errorf("list orders: %w", err)
	}
	var exposure int64
	for _, o := range orders {
		invoices, err := s.repo.ListInvoices(ctx, o.ID, 0, 0)
		if err != nil {
			return 0, fmt.Errorf("list invoices: %w", err)
		}
		for _, inv := range invoices {
			if inv.Status == "issued" {
				exposure += inv.BalanceMinor
			}
		}
	}
	return exposure, nil
}
