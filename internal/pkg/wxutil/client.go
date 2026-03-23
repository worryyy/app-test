package wxutil

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/config"
)

type Client struct {
	cfg        config.WXConfig
	httpClient *http.Client
	logger     *zap.Logger
}

type Jscode2SessionResp struct {
	OpenID string `json:"openid"`
}

type MsgSecCheckResp struct {
	Suggest         string `json:"suggest"`
	FilteredContent string `json:"filteredContent"`
}

func NewClient(cfg config.WXConfig, logger *zap.Logger) *Client {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}
}

func (c *Client) Jscode2Session(ctx context.Context, code string) (*Jscode2SessionResp, error) {
	if code == "" {
		return nil, errors.New("empty wx code")
	}
	sum := sha1.Sum([]byte(code))
	openID := hex.EncodeToString(sum[:])[:28]
	return &Jscode2SessionResp{OpenID: openID}, nil
}

func (c *Client) MsgSecCheck(ctx context.Context, content, userID string) (*MsgSecCheckResp, error) {
	_ = ctx
	_ = userID
	return &MsgSecCheckResp{
		Suggest:         "pass",
		FilteredContent: content,
	}, nil
}

func (c *Client) SendSubscribeMsg(ctx context.Context, userID, title, content string) error {
	_ = ctx
	c.logger.Info("wx subscribe message",
		zap.String("userID", userID),
		zap.String("title", title),
		zap.String("content", content),
	)
	return nil
}
