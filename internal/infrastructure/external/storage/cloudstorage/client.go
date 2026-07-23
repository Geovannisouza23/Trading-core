// Package cloudstorage is a placeholder for a future Google Cloud
// Storage-backed adapter (e.g. long-term audit export of snapshots/
// incidents). Nothing in the current pipeline needs blob storage yet.
package cloudstorage

import (
	"context"
	"errors"
)

var ErrNotConfigured = errors.New("cloud storage is not configured in this deployment")

type Client struct {
	bucket string
}

func NewClient(bucket string) *Client {
	return &Client{bucket: bucket}
}

func (c *Client) Upload(ctx context.Context, key string, data []byte) error {
	return ErrNotConfigured
}

func (c *Client) Download(ctx context.Context, key string) ([]byte, error) {
	return nil, ErrNotConfigured
}
