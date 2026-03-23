package wxutil

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/config"
)

type Client struct {
	cfg        config.WXConfig
	httpClient *http.Client
	logger     *zap.Logger

	mu          sync.Mutex
	accessToken string
	expiresAt   time.Time
}

type Jscode2SessionResp struct {
	OpenID     string `json:"openid"`
	SessionKey string `json:"session_key"`
	UnionID    string `json:"unionid"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

type MsgSecCheckResp struct {
	Suggest         string `json:"suggest"`
	Label           string `json:"label"`
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

	values := url.Values{}
	values.Set("appid", c.cfg.AppID)
	values.Set("secret", c.cfg.Secret)
	values.Set("js_code", code)
	values.Set("grant_type", "authorization_code")

	endpoint := "https://api.weixin.qq.com/sns/jscode2session?" + values.Encode()
	resp, err := c.doJSONRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("request jscode2session: %w", err)
	}

	var out Jscode2SessionResp
	if err := json.Unmarshal(resp, &out); err != nil {
		return nil, fmt.Errorf("decode jscode2session response: %w", err)
	}
	if out.ErrCode != 0 {
		return nil, fmt.Errorf("jscode2session errcode=%d errmsg=%s", out.ErrCode, out.ErrMsg)
	}
	if out.OpenID == "" {
		return nil, errors.New("openid is empty")
	}
	return &out, nil
}

func (c *Client) MsgSecCheck(ctx context.Context, content, userID string) (*MsgSecCheckResp, error) {
	token, err := c.getStableAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"content": content,
		"version": 2,
		"scene":   1,
		"openid":  userID,
	}
	endpoint := "https://api.weixin.qq.com/wxa/msg_sec_check?access_token=" + token
	resp, err := c.doJSONRequest(ctx, http.MethodPost, endpoint, payload)
	if err != nil {
		return nil, fmt.Errorf("request msg_sec_check: %w", err)
	}

	var raw struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
		Result  struct {
			Suggest string `json:"suggest"`
			Label   string `json:"label"`
		} `json:"result"`
		Detail []struct {
			FilteredContent string `json:"filtered_content"`
		} `json:"detail"`
	}
	if err := json.Unmarshal(resp, &raw); err != nil {
		return nil, fmt.Errorf("decode msg_sec_check response: %w", err)
	}
	if raw.ErrCode != 0 {
		return nil, fmt.Errorf("msg_sec_check errcode=%d errmsg=%s", raw.ErrCode, raw.ErrMsg)
	}

	filtered := content
	if len(raw.Detail) > 0 && strings.TrimSpace(raw.Detail[0].FilteredContent) != "" {
		filtered = raw.Detail[0].FilteredContent
	}
	return &MsgSecCheckResp{
		Suggest:         raw.Result.Suggest,
		Label:           raw.Result.Label,
		FilteredContent: filtered,
	}, nil
}

func (c *Client) SendSubscribeMsg(ctx context.Context, userID, title, content string) error {
	token, err := c.getStableAccessToken(ctx)
	if err != nil {
		return err
	}
	if strings.TrimSpace(c.cfg.SubscribeTemplateID) == "" {
		return errors.New("wx subscribe template id is empty")
	}

	payload := map[string]interface{}{
		"touser":      userID,
		"template_id": c.cfg.SubscribeTemplateID,
		"data": map[string]interface{}{
			"thing1": map[string]string{"value": title},
			"thing2": map[string]string{"value": content},
		},
	}
	endpoint := "https://api.weixin.qq.com/cgi-bin/message/subscribe/send?access_token=" + token
	resp, err := c.doJSONRequest(ctx, http.MethodPost, endpoint, payload)
	if err != nil {
		return fmt.Errorf("request subscribe message: %w", err)
	}

	var raw struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(resp, &raw); err != nil {
		return fmt.Errorf("decode subscribe response: %w", err)
	}
	if raw.ErrCode != 0 {
		return fmt.Errorf("subscribe message errcode=%d errmsg=%s", raw.ErrCode, raw.ErrMsg)
	}
	return nil
}

func (c *Client) UnlimitedWXACode(ctx context.Context, scene, page string) ([]byte, error) {
	token, err := c.getStableAccessToken(ctx)
	if err != nil {
		return nil, err
	}
	payload := map[string]interface{}{
		"scene":      scene,
		"page":       page,
		"check_path": false,
	}
	endpoint := "https://api.weixin.qq.com/wxa/getwxacodeunlimit?access_token=" + token
	body, err := c.doJSONRequest(ctx, http.MethodPost, endpoint, payload)
	if err != nil {
		return nil, fmt.Errorf("request getwxacodeunlimit: %w", err)
	}

	if json.Valid(body) {
		var raw struct {
			ErrCode int    `json:"errcode"`
			ErrMsg  string `json:"errmsg"`
		}
		if err := json.Unmarshal(body, &raw); err == nil && raw.ErrCode != 0 {
			return nil, fmt.Errorf("getwxacodeunlimit errcode=%d errmsg=%s", raw.ErrCode, raw.ErrMsg)
		}
	}
	return body, nil
}

func (c *Client) getStableAccessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.accessToken != "" && time.Now().Before(c.expiresAt) {
		return c.accessToken, nil
	}
	payload := map[string]interface{}{
		"grant_type": "client_credential",
		"appid":      c.cfg.AppID,
		"secret":     c.cfg.Secret,
	}
	resp, err := c.doJSONRequest(ctx, http.MethodPost, "https://api.weixin.qq.com/cgi-bin/stable_token", payload)
	if err != nil {
		return "", fmt.Errorf("request stable_token: %w", err)
	}

	var raw struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}
	if err := json.Unmarshal(resp, &raw); err != nil {
		return "", fmt.Errorf("decode stable_token response: %w", err)
	}
	if raw.ErrCode != 0 {
		return "", fmt.Errorf("stable_token errcode=%d errmsg=%s", raw.ErrCode, raw.ErrMsg)
	}
	if raw.AccessToken == "" {
		return "", errors.New("stable_token access token is empty")
	}

	expiresIn := raw.ExpiresIn
	if expiresIn <= 200 {
		expiresIn = 300
	}
	c.accessToken = raw.AccessToken
	c.expiresAt = time.Now().Add(time.Duration(expiresIn-200) * time.Second)
	return c.accessToken, nil
}

func (c *Client) doJSONRequest(ctx context.Context, method, endpoint string, payload interface{}) ([]byte, error) {
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal wx request payload: %w", err)
		}
		body = strings.NewReader(string(raw))
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("create wx request: %w", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do wx request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read wx response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("wx request status=%d body=%s", resp.StatusCode, string(raw))
	}
	return raw, nil
}

func (c *Client) LogMaskedSubscribeFailure(userID string, err error) {
	c.logger.Warn("wx subscribe message failed",
		zap.String("userID", userID),
		zap.Error(err),
	)
}
