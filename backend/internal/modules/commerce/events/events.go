package events

// Domain events published by the commerce module. Shapes follow
// docs/06-data-and-events.md: event_id, event_type, occurred_at, version,
// payload, correlation_id. Delivery is at-least-once; consumers dedupe on
// event_id. No outbox exists yet: checkout publishes after commit on first
// execution only, so HTTP retries never duplicate events (broker-level
// redelivery remains possible and is the consumer's to dedupe).

import (
	"time"

	"github.com/google/uuid"
)

const (
	EventOrderPlaced = "commerce.order.placed"
	EventVersionV1    = 1
)

type Envelope struct {
	EventID       string      `json:"event_id"`
	EventType     string      `json:"event_type"`
	OccurredAt    time.Time   `json:"occurred_at"`
	Version       int         `json:"version"`
	Payload       interface{} `json:"payload"`
	CorrelationID string      `json:"correlation_id,omitempty"`
}

type OrderPlacedPayload struct {
	OrderCode      string `json:"order_code"`
	BuyerID        int64  `json:"buyer_id"`
	TotalMinor     int64  `json:"total_minor"`
	Currency       string `json:"currency"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
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
