package events

import (
	"context"
	"encoding/json"
	"time"
)

type EventPublisher interface {
	Publish(ctx context.Context, routingKey string, payload interface{}) error
}

type Event struct {
	EventID       string          `json:"event_id"`
	EventType     string          `json:"event_type"`
	OccurredAt    time.Time       `json:"occurred_at"`
	Version       int             `json:"version"`
	Payload       json.RawMessage `json:"payload"`
	CorrelationID string          `json:"correlation_id,omitempty"`
}

const (
	EventBuyerRegistered       = "identity.buyer.registered"
	EventVerificationSubmitted = "identity.verification.submitted"
	EventVerificationDecided   = "identity.verification.decided"
)

type IdentityEventHandler struct {
	publisher EventPublisher
}

func NewIdentityEventHandler(publisher EventPublisher) *IdentityEventHandler {
	return &IdentityEventHandler{publisher: publisher}
}

func (h *IdentityEventHandler) PublishBuyerRegistered(ctx context.Context, buyerProfile interface{}, user interface{}) error {
	return nil
}

func (h *IdentityEventHandler) PublishVerificationSubmitted(ctx context.Context, app interface{}) error {
	return nil
}

func (h *IdentityEventHandler) PublishVerificationDecided(ctx context.Context, app interface{}, decidedBy int64, reason string) error {
	return nil
}
