package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type FeedClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewFeedClient(baseURL string) *FeedClient {
	return &FeedClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 3 * time.Second},
	}
}

type indexPayload struct {
	ID       string   `json:"id"`
	AuthorID string   `json:"author_id"`
	Tags     []string `json:"tags"`
}

// IndexPost tells the feed-service to add/update this post in the feed index.
// Called fire-and-forget after a public post is created or updated.
func (c *FeedClient) IndexPost(ctx context.Context, id, authorID string, tags []string) error {
	b, _ := json.Marshal(indexPayload{ID: id, AuthorID: authorID, Tags: tags})
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/internal/feed/index", bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("feed index request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("feed index returned %d", resp.StatusCode)
	}
	return nil
}
