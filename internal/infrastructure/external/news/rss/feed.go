// Package rss implements provider.Provider by polling a standard RSS 2.0
// feed URL.
package rss

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"

	"trading-core/internal/infrastructure/external/news/provider"
)

type rssDocument struct {
	Channel struct {
		Items []struct {
			Title       string `xml:"title"`
			Link        string `xml:"link"`
			Description string `xml:"description"`
			PubDate     string `xml:"pubDate"`
		} `xml:"item"`
	} `xml:"channel"`
}

type Feed struct {
	httpClient *http.Client
	name       string
	feedURL    string
}

func NewFeed(name, feedURL string) *Feed {
	return &Feed{httpClient: &http.Client{Timeout: 10 * time.Second}, name: name, feedURL: feedURL}
}

var _ provider.Provider = (*Feed)(nil)

func (f *Feed) Name() string { return f.name }

func (f *Feed) Fetch(ctx context.Context) ([]provider.NewsItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.feedURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("rss: http %d fetching %s", resp.StatusCode, f.feedURL)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var doc rssDocument
	if err := xml.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("rss: parsing feed %s: %w", f.feedURL, err)
	}

	items := make([]provider.NewsItem, 0, len(doc.Channel.Items))
	for _, item := range doc.Channel.Items {
		publishedAt, err := time.Parse(time.RFC1123Z, item.PubDate)
		if err != nil {
			publishedAt = time.Now().UTC()
		}
		items = append(items, provider.NewsItem{
			Source:      f.name,
			URL:         item.Link,
			Title:       item.Title,
			Content:     item.Description,
			PublishedAt: publishedAt,
		})
	}
	return items, nil
}
