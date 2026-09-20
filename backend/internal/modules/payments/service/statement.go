package service

// TASK-008: account statements (derived reads, no new tables). A statement
// lists period activity (issued invoices, succeeded intents) plus the
// current outstanding across the buyer's orders. Balances already reflect
// applied payments and credit notes. Period format: YYYY-MM.

import (
	"context"
	"fmt"
	"time"
)

type StatementLine struct {
	Kind        string `json:"kind"`
	Code        string `json:"code"`
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
	Status      string `json:"status"`
	At          string `json:"at"`
}

type Statement struct {
	BuyerID       int64           `json:"buyer_id"`
	Period        string          `json:"period"`
	OpeningMinor  int64           `json:"opening_minor"`
	InvoicedMinor int64           `json:"invoiced_minor"`
	PaidMinor     int64           `json:"paid_minor"`
	ClosingMinor  int64           `json:"closing_minor"`
	Currency      string          `json:"currency"`
	Lines         []StatementLine `json:"lines"`
}

func parsePeriod(period string) (time.Time, time.Time, error) {
	start, err := time.Parse("2006-01", period)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return start.UTC(), start.UTC().AddDate(0, 1, 0), nil
}

func inPeriod(t time.Time, start, end time.Time) bool {
	return !t.Before(start) && t.Before(end)
}

// BuyerStatement builds the buyer's statement for a month.
func (s *PaymentService) BuyerStatement(ctx context.Context, buyerID int64, period string) (*Statement, error) {
	start, end, err := parsePeriod(period)
	if err != nil {
		return nil, ErrInvalidIntent
	}
	orders, err := s.commerceSvc.ListOrders(ctx, buyerID)
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	intents, err := s.repo.ListIntents(ctx, buyerID, "", 0, 0)
	if err != nil {
		return nil, fmt.Errorf("list intents: %w", err)
	}

	stmt := &Statement{BuyerID: buyerID, Period: period, Currency: "USD", Lines: []StatementLine{}}
	var invoiced, paid int64
	for _, o := range orders {
		invoices, err := s.commerceSvc.OrderInvoices(ctx, buyerID, o.Code)
		if err != nil {
			return nil, fmt.Errorf("list invoices: %w", err)
		}
		for _, inv := range invoices {
			if inv.Currency != "" {
				stmt.Currency = inv.Currency
			}
			stmt.ClosingMinor += inv.BalanceMinor
			if inv.IssuedAt != nil && inPeriod(*inv.IssuedAt, start, end) {
				invoiced += inv.TotalMinor
				stmt.Lines = append(stmt.Lines, StatementLine{
					Kind: "invoice", Code: inv.Code, AmountMinor: inv.TotalMinor,
					Currency: inv.Currency, Status: inv.Status, At: inv.IssuedAt.Format(time.RFC3339),
				})
			}
		}
	}
	for _, in := range intents {
		if in.Status != IntentStatusSucceeded {
			continue
		}
		if in.Currency != "" && stmt.Currency == "USD" {
			stmt.Currency = in.Currency
		}
		if inPeriod(in.UpdatedAt, start, end) {
			paid += in.AmountMinor
			stmt.Lines = append(stmt.Lines, StatementLine{
				Kind: "payment", Code: in.Code, AmountMinor: in.AmountMinor,
				Currency: in.Currency, Status: in.Status, At: in.UpdatedAt.Format(time.RFC3339),
			})
		}
	}
	stmt.InvoicedMinor = invoiced
	stmt.PaidMinor = paid
	stmt.OpeningMinor = stmt.ClosingMinor - invoiced + paid
	return stmt, nil
}

// AdminStatement builds any buyer's statement (staff path).
func (s *PaymentService) AdminStatement(ctx context.Context, buyerID int64, period string) (*Statement, error) {
	if buyerID <= 0 {
		return nil, ErrInvalidIntent
	}
	return s.BuyerStatement(ctx, buyerID, period)
}
