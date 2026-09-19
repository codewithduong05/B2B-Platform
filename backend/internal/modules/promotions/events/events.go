package events

// Domain events published by the promotions module. Shapes follow
// docs/06-data-and-events.md. Only the catalogued voucher event is emitted.

import (
	"time"

	"github.com/google/uuid"
)

const (
	EventVoucherRedeemed = "promotions.voucher.redeemed"
	EventVersionV1       = 1
)

type Envelope struct {
	EventID       string      `json:"event_id"`
	EventType     string      `json:"event_type"`
	OccurredAt    time.Time   `json:"occurred_at"`
	Version       int         `json:"version"`
	Payload       interface{} `json:"payload"`
	CorrelationID string      `json:"correlation_id,omitempty"`
}

type VoucherRedeemedPayload struct {
	PromotionCode string `json:"promotion_code"`
	BuyerID       int64  `json:"buyer_id"`
	OrderCode     string `json:"order_code"`
	AmountMinor   int64  `json:"amount_minor"`
	Currency      string `json:"currency"`
}

func NewEnvelope(eventType string, payload interface{}, correlationID string) Envelope {
	return Envelope{
		EventID:       uuid.NewString(),
		EventType:     eventType,
		OccurredAt:    time.Now().UTC(),
		Version:       EventVersionV1,
		Payload:       payload,
		CorrelationID: correlationID,
	}
}
