package outbox

import (
	"context"
	"encoding/json"
	"fmt"

	"trading-core/internal/application/ports/output"
	"trading-core/internal/domain/event"
	"trading-core/internal/domain/operation"
	"trading-core/internal/domain/order"
	"trading-core/internal/domain/position"
)

// Publisher decodes a persisted OutboxEvent back into its concrete domain
// event type and publishes it on the EventBus. It is the only place that
// needs to know the mapping between an event_type string and its payload
// shape.
type Publisher struct {
	bus output.EventBus
}

func NewPublisher(bus output.EventBus) *Publisher {
	return &Publisher{bus: bus}
}

// Publish decodes evt and forwards it to the EventBus.
func (p *Publisher) Publish(ctx context.Context, evt output.OutboxEvent) error {
	domainEvent, err := decode(evt)
	if err != nil {
		return err
	}
	return p.bus.Publish(ctx, domainEvent)
}

func decode(evt output.OutboxEvent) (output.DomainEvent, error) {
	switch evt.EventType {
	case "OrderCreated":
		return decodeInto(evt, &order.OrderCreated{})
	case "OrderSubmitted":
		return decodeInto(evt, &order.OrderSubmitted{})
	case "OrderPartiallyFilled":
		return decodeInto(evt, &order.OrderPartiallyFilled{})
	case "OrderFilled":
		return decodeInto(evt, &order.OrderFilled{})
	case "OrderCancelled":
		return decodeInto(evt, &order.OrderCancelled{})
	case "PositionOpened":
		return decodeInto(evt, &position.PositionOpened{})
	case "PositionClosed":
		return decodeInto(evt, &position.PositionClosed{})
	case "KillSwitchActivated":
		return decodeInto(evt, &operation.KillSwitchActivated{})
	case "CriticalEventDetected":
		return decodeInto(evt, &event.CriticalEventDetected{})
	case "ReconciliationDivergenceDetected":
		return decodeInto(evt, &operation.ReconciliationDivergenceDetected{})
	default:
		return nil, fmt.Errorf("unknown outbox event type %q", evt.EventType)
	}
}

func decodeInto[T output.DomainEvent](evt output.OutboxEvent, target T) (output.DomainEvent, error) {
	if err := json.Unmarshal(evt.Payload, target); err != nil {
		return nil, fmt.Errorf("decoding outbox payload for %s: %w", evt.EventType, err)
	}
	return target, nil
}
