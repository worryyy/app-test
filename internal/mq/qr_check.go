package mq

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type qrCheckRequest struct {
	URLs []string `json:"urls"`
}

type qrCheckResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    []qrCheckResult `json:"data"`
}

type qrCheckResult struct {
	URL   string `json:"url"`
	HasQR bool   `json:"has_qr"`
}

func (c *Consumers) filterImagesWithQRCode(ctx context.Context, urls []string) ([]string, error) {
	if len(urls) == 0 {
		return urls, nil
	}

	results, err := c.checkQRCode(ctx, urls)
	if err != nil || results == nil {
		return urls, err
	}

	hit := make(map[string]struct{})
	for _, result := range results {
		if result.HasQR {
			hit[result.URL] = struct{}{}
		}
	}
	if len(hit) == 0 {
		return urls, nil
	}

	filtered := make([]string, 0, len(urls))
	for _, url := range urls {
		if _, ok := hit[url]; ok {
			continue
		}
		filtered = append(filtered, url)
	}
	return filtered, nil
}

func (c *Consumers) checkQRCode(ctx context.Context, urls []string) ([]qrCheckResult, error) {
	if c.cfg == nil || strings.TrimSpace(c.cfg.JW.BaseURL) == "" {
		return nil, fmt.Errorf("jw base url not configured")
	}

	body, err := json.Marshal(qrCheckRequest{URLs: urls})
	if err != nil {
		return nil, fmt.Errorf("marshal qr check request: %w", err)
	}

	endpoint := strings.TrimRight(c.cfg.JW.BaseURL, "/") + "/qr/check"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create qr check request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(c.cfg.JW.APIKey) != "" {
		req.Header.Set("X-API-Key", c.cfg.JW.APIKey)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request qr check: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("qr check status code: %d", resp.StatusCode)
	}

	var raw qrCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode qr check response: %w", err)
	}
	if raw.Code != http.StatusOK {
		return nil, fmt.Errorf("qr check failed: code=%d message=%s", raw.Code, raw.Message)
	}
	return raw.Data, nil
}
