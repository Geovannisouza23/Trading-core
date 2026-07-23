package usecase

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"trading-core/internal/application/ports/output"
)

// newOutboxEvent serializes a domain event into a pending outbox row. Callers
// must insert the result within the same transaction as the aggregate write
// it accompanies.
func newOutboxEvent(eventType string, payload any, now time.Time) (output.OutboxEvent, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return output.OutboxEvent{}, err
	}
	return output.OutboxEvent{
		ID:        uuid.NewString(),
		EventType: eventType,
		Payload:   body,
		Status:    output.OutboxStatusPending,
		CreatedAt: now,
	}, nil
}
