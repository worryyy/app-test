package cosutil

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/tencentyun/cos-go-sdk-v5"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/config"
)

type Client struct {
	cfg            config.COSConfig
	logger         *zap.Logger
	baseClient     *cos.Client
	compressClient *cos.Client
}

func NewClient(cfg config.COSConfig, logger *zap.Logger) *Client {
	if logger == nil {
		logger = zap.NewNop()
	}

	baseClient, baseErr := newCOSClient(cfg.BaseURL, cfg.BucketName, cfg.Region, cfg.AccessKeyID, cfg.AccessKeySecret)
	if baseErr != nil {
		logger.Warn("init base cos client failed", zap.Error(baseErr))
	}

	compressClient, compressErr := newCOSClient("", cfg.CompressBucket, cfg.Region, cfg.AccessKeyID, cfg.AccessKeySecret)
	if compressErr != nil && cfg.CompressBucket != "" {
		logger.Warn("init compress cos client failed", zap.Error(compressErr))
	}

	return &Client{
		cfg:            cfg,
		logger:         logger,
		baseClient:     baseClient,
		compressClient: compressClient,
	}
}

func (c *Client) PutWithImageProcess(ctx context.Context, objectKey string, data []byte, process string) (string, error) {
	contentType := http.DetectContentType(data)
	if strings.TrimSpace(process) != "" && c.compressClient != nil {
		if _, err := putObject(ctx, c.compressClient, objectKey, data, contentType); err == nil {
			return buildURL(c.cfg.CompressBaseCDN, objectKey), nil
		}
		c.logger.Warn("upload to compress bucket failed, fallback to base bucket", zap.String("key", objectKey))
	}
	return c.Put(ctx, objectKey, data, contentType)
}

func (c *Client) Put(ctx context.Context, objectKey string, data []byte, contentType string) (string, error) {
	if c.baseClient == nil {
		return "", fmt.Errorf("base cos client not initialized")
	}
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	if _, err := putObject(ctx, c.baseClient, objectKey, data, contentType); err != nil {
		return "", fmt.Errorf("upload object to cos: %w", err)
	}
	return buildURL(c.cfg.BaseCDN, objectKey), nil
}

func (c *Client) Delete(ctx context.Context, objectKey string) error {
	if c.baseClient != nil {
		if _, err := c.baseClient.Object.Delete(ctx, objectKey); err != nil {
			return fmt.Errorf("delete object from base cos: %w", err)
		}
	}
	if c.compressClient != nil {
		if _, err := c.compressClient.Object.Delete(ctx, objectKey); err != nil {
			c.logger.Warn("delete object from compress cos failed", zap.Error(err), zap.String("objectKey", objectKey))
		}
	}
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

func newCOSClient(
	baseURL string,
	bucket string,
	region string,
	secretID string,
	secretKey string,
) (*cos.Client, error) {
	endpoint := strings.TrimSpace(baseURL)
	if endpoint == "" && strings.TrimSpace(bucket) != "" && strings.TrimSpace(region) != "" {
		endpoint = fmt.Sprintf("https://%s.cos.%s.myqcloud.com", bucket, region)
	}
	if endpoint == "" {
		return nil, fmt.Errorf("cos endpoint is empty")
	}

	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse cos endpoint: %w", err)
	}
	client := cos.NewClient(&cos.BaseURL{BucketURL: u}, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  secretID,
			SecretKey: secretKey,
		},
	})
	return client, nil
}

func putObject(
	ctx context.Context,
	client *cos.Client,
	objectKey string,
	data []byte,
	contentType string,
) (*cos.Response, error) {
	return client.Object.Put(ctx, objectKey, bytes.NewReader(data), &cos.ObjectPutOptions{
		ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{
			ContentType: contentType,
		},
	})
}
