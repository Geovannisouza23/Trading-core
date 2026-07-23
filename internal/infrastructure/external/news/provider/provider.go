// Package provider defines the common shape every concrete news source
// implements, so a scheduler can poll them uniformly and feed results into
// application/usecase.ProcessMarketEvent.
package provider

import (
	"context"
	"time"
)

// NewsItem is a raw item fetched from an external source, before any
// sanitization/validation — that happens inside ProcessMarketEvent itself.
type NewsItem struct {
	Source      string
	URL         string
	Title       string
	Content     string
	PublishedAt time.Time
}

// Provider is implemented by every concrete news source (RSS feeds, future
// news APIs).
type Provider interface {
	Name() string
	Fetch(ctx context.Context) ([]NewsItem, error)
}
