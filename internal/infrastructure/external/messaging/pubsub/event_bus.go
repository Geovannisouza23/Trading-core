// Package pubsub is a placeholder for a future Google Pub/Sub-backed
// output.EventBus. It exists so the folder structure documents where that
// substitution would live; it is not wired into internal/app because the
// MVP uses messaging/memory instead (spec section 8.7).
package pubsub

import (
	"context"
	"errors"

	"trading-core/internal/application/ports/output"
)

var ErrNotConfigured = errors.New("pubsub event bus is not implemented in this deployment; use the in-memory bus")

type Client struct {
	projectID string
}

func NewClient(projectID string) *Client {
	return &Client{projectID: projectID}
}

var _ output.EventBus = (*Client)(nil)

func (c *Client) Publish(ctx context.Context, event output.DomainEvent) error {
	return ErrNotConfigured
}

func (c *Client) Subscribe(eventName string, handler output.EventHandler) {}
