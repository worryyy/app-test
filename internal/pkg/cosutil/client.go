package cosutil

import (
	"context"
	"strings"

	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/config"
)

type Client struct {
	cfg    config.COSConfig
	logger *zap.Logger
}

func NewClient(cfg config.COSConfig, logger *zap.Logger) *Client {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Client{cfg: cfg, logger: logger}
}

func (c *Client) PutWithImageProcess(ctx context.Context, objectKey string, data []byte, process string) (string, error) {
	_ = ctx
	_ = data
	_ = process
	url := buildURL(c.cfg.CompressBaseCDN, objectKey)
	if url == "" {
		url = buildURL(c.cfg.BaseCDN, objectKey)
	}
	return url, nil
}

func (c *Client) Put(ctx context.Context, objectKey string, data []byte, contentType string) (string, error) {
	_ = ctx
	_ = data
	_ = contentType
	return buildURL(c.cfg.BaseCDN, objectKey), nil
}

func (c *Client) Delete(ctx context.Context, objectKey string) error {
	_ = ctx
	c.logger.Info("cos delete object", zap.String("objectKey", objectKey))
	return nil
}

func buildURL(base, objectKey string) string {
	if base == "" {
		return objectKey
	}
	if strings.HasSuffix(base, "/") {
		return base + objectKey
	}
	return base + "/" + objectKey
}
