package schema

import (
	"time"
)

type PaymentMethodResponse struct {
	Code        string    `json:"code"`
	MethodType  string    `json:"method_type"`
	DisplayName string    `json:"display_name"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateIntentRequest struct {
	OrderCode  string  `json:"order_code"`
	MethodCode *string `json:"method_code,omitempty"`
	IdemKey    string  `json:"idem_key"`
}

type IntentResponse struct {
	Code         string            `json:"code"`
	OrderCode    string            `json:"order_code"`
	AmountMinor  int64             `json:"amount_minor"`
	Currency     string            `json:"currency"`
	MethodCode   *string           `json:"method_code,omitempty"`
	Status       string            `json:"status"`
	Attempts     []AttemptResponse `json:"attempts,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

type AttemptResponse struct {
	Result     string    `json:"result"`
	GatewayRef *string   `json:"gateway_ref,omitempty"`
	Note       string    `json:"note,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type MarkIntentRequest struct {
	Note string `json:"note,omitempty"`
}

type RefundResponse struct {
	Code        string    `json:"code"`
	IntentCode  string    `json:"intent_code"`
	AmountMinor int64     `json:"amount_minor"`
	Currency    string    `json:"currency"`
	Reason      string    `json:"reason"`
	Status      string    `json:"status"`
	RequestedBy *int64    `json:"requested_by,omitempty"`
	ApprovedBy  *int64    `json:"approved_by,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type RequestRefundRequest struct {
	AmountMinor int64  `json:"amount_minor"`
	Reason      string `json:"reason"`
}

type ReconResponse struct {
	OrderID        int64  `json:"order_id"`
	OrderCode      string `json:"order_code"`
	InvoicedTotal  int64  `json:"invoiced_total"`
	PaidApplied    int64  `json:"paid_applied"`
	SucceededTotal int64  `json:"succeeded_total"`
	Status         string `json:"status"`
}

type WebhookResponse struct {
	Provider   string    `json:"provider"`
	EventID    string    `json:"event_id"`
	EventType  string    `json:"event_type"`
	IntentCode string    `json:"intent_code,omitempty"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

type CreditResponse struct {
	BuyerID          int64     `json:"buyer_id"`
	CreditLimitMinor int64     `json:"credit_limit_minor"`
	Terms            string    `json:"terms"`
	OnHold           bool      `json:"on_hold"`
	HoldReason       *string   `json:"hold_reason,omitempty"`
	ExposureMinor    int64     `json:"exposure_minor"`
	AvailableMinor   int64     `json:"available_minor"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type SetCreditRequest struct {
	CreditLimitMinor int64   `json:"credit_limit_minor"`
	Terms            string  `json:"terms"`
	OnHold           bool    `json:"on_hold"`
	HoldReason       *string `json:"hold_reason,omitempty"`
}
