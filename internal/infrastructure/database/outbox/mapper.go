package outbox

import "trading-core/internal/application/ports/output"

func toDomain(m model) output.OutboxEvent {
	return output.OutboxEvent{
		ID:          m.ID,
		EventType:   m.EventType,
		Payload:     m.Payload,
		Status:      output.OutboxStatus(m.Status),
		Attempts:    m.Attempts,
		CreatedAt:   m.CreatedAt,
		ProcessedAt: m.ProcessedAt,
		LastError:   m.LastError,
	}
}
