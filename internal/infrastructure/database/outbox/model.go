// Package outbox implements output.OutboxRepository (transactional outbox
// pattern) plus the worker that drains it and publishes to the EventBus.
package outbox

import "time"

type model struct {
	ID          string
	EventType   string
	Payload     []byte
	Status      string
	Attempts    int
	CreatedAt   time.Time
	ProcessedAt *time.Time
	LastError   string
}
