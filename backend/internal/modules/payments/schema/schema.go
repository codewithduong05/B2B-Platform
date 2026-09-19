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
