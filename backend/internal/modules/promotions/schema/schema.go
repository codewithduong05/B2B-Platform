package schema

import (
	"time"
)

type PromotionResponse struct {
	Code           string     `json:"code"`
	Name           string     `json:"name"`
	Kind           string     `json:"kind"`
	ValueMinor     int64      `json:"value_minor"`
	Currency       string     `json:"currency"`
	Status         string     `json:"status"`
	ValidFrom      *time.Time `json:"valid_from,omitempty"`
	ValidTo        *time.Time `json:"valid_to,omitempty"`
	MaxRedemptions *int       `json:"max_redemptions,omitempty"`
	RedeemedCount  int        `json:"redeemed_count"`
	Redeemable     *bool      `json:"redeemable,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type CreatePromotionRequest struct {
	Code           string  `json:"code"`
	Name           string  `json:"name"`
	Kind           string  `json:"kind"`
	ValueMinor     int64   `json:"value_minor"`
	Currency       string  `json:"currency,omitempty"`
	ValidFrom      *string `json:"valid_from,omitempty"`
	ValidTo        *string `json:"valid_to,omitempty"`
	MaxRedemptions *int    `json:"max_redemptions,omitempty"`
}

type UpdatePromotionRequest struct {
	Name           *string `json:"name,omitempty"`
	ValueMinor     *int64  `json:"value_minor,omitempty"`
	ValidFrom      *string `json:"valid_from,omitempty"`
	ValidTo        *string `json:"valid_to,omitempty"`
	MaxRedemptions *int    `json:"max_redemptions,omitempty"`
	Status         *string `json:"status,omitempty"`
}

type ApplyVoucherRequest struct {
	Code string `json:"code"`
}

type ReportRow struct {
	Code            string `json:"code"`
	Name            string `json:"name"`
	Status          string `json:"status"`
	RedeemedCount   int    `json:"redeemed_count"`
	TotalCostMinor  int64  `json:"total_cost_minor"`
	BudgetRemaining *int   `json:"budget_remaining,omitempty"`
}
