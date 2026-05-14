package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type NotifyClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewNotifyClient(baseURL string) *NotifyClient {
	return &NotifyClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

type sendOTPPayload struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

func (c *NotifyClient) SendOTP(ctx context.Context, email, code string) error {
	payload, _ := json.Marshal(sendOTPPayload{Email: email, Code: code})
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/internal/send-otp", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("notify request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("notify returned %d", resp.StatusCode)
	}
	return nil
}
