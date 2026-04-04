package school

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/config"
)

const defaultJWUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

type JWRequestHelper struct {
	cfg       *config.Config
	logger    *zap.Logger
	client    *http.Client
	userAgent string
}

func NewJWRequestHelper(cfg *config.Config, logger *zap.Logger) *JWRequestHelper {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &JWRequestHelper{
		cfg:       cfg,
		logger:    logger,
		client:    &http.Client{Timeout: 15 * time.Second},
		userAgent: defaultJWUserAgent,
	}
}

func (h *JWRequestHelper) DoJSON(
	ctx context.Context,
	method string,
	path string,
	query url.Values,
	payload any,
) (*JWCommonResp, error) {
	if h == nil || h.client == nil {
		return nil, fmt.Errorf("jw request helper not initialized")
	}
	if h.cfg == nil || strings.TrimSpace(h.cfg.JW.BaseURL) == "" {
		return nil, fmt.Errorf("jw base url not configured")
	}

	var body []byte
	var err error
	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal jw request: %w", err)
		}
	}

	endpoint := strings.TrimRight(h.cfg.JW.BaseURL, "/") + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create jw request: %w", err)
	}

	req.Header.Set("User-Agent", h.userAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(h.cfg.JW.APIKey) != "" && h.cfg.JW.APIKey != "replace-me" {
		req.Header.Set("X-API-Key", h.cfg.JW.APIKey)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request jw api: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			h.logger.Warn("close jw response body failed", zap.Error(closeErr))
		}
	}()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read jw response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("jw api status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var commonResp JWCommonResp
	if err := json.Unmarshal(raw, &commonResp); err != nil {
		return nil, fmt.Errorf("decode jw response: %w", err)
	}
	return &commonResp, nil
}
