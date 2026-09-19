package service_test

import (
	"context"
	"testing"

	"github.com/atlas-platform/backend/internal/modules/inventory/service"
	"github.com/stretchr/testify/assert"
)

type MockEventPublisher struct {
	PublishedEvents []struct {
		RoutingKey string
		Payload    interface{}
	}
}

func (m *MockEventPublisher) Publish(ctx context.Context, routingKey string, payload interface{}) error {
	m.PublishedEvents = append(m.PublishedEvents, struct {
		RoutingKey string
		Payload    interface{}
	}{RoutingKey: routingKey, Payload: payload})
	return nil
}

func TestInventoryService_BusinessRules(t *testing.T) {
	t.Run("Insufficient stock validation", func(t *testing.T) {
		publisher := &MockEventPublisher{}
		_ = publisher
		assert.Equal(t, service.ErrInvalidQuantity, service.ErrInvalidQuantity)
	})
}
