package school

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/config"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/encrypt"
)

type JWClient struct {
	cfg       *config.Config
	logger    *zap.Logger
	httpMaker func(jar http.CookieJar) *http.Client
}

func NewJWClient(cfg *config.Config, logger *zap.Logger) *JWClient {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &JWClient{
		cfg:    cfg,
		logger: logger,
		httpMaker: func(jar http.CookieJar) *http.Client {
			return &http.Client{
				Jar: jar,
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse
				},
				Timeout: 15 * time.Second,
			}
		},
	}
}

func (j *JWClient) Login(ctx context.Context, stuNum, stuPwd string) ([]*http.Cookie, error) {
	if j.cfg == nil || j.cfg.JW.BaseURL == "" {
		return []*http.Cookie{}, nil
	}

	key := []byte("PassB01I")[:8]
	encrypted, err := encrypt.DESECBEncrypt([]byte(stuPwd), key)
	if err != nil {
		return nil, fmt.Errorf("jw encrypt password: %w", err)
	}
	encoded := base64.StdEncoding.EncodeToString(encrypted)

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("create cookie jar: %w", err)
	}
	client := j.httpMaker(jar)

	form := url.Values{
		"username": {stuNum},
		"password": {encoded},
	}
	loginURL := strings.TrimRight(j.cfg.JW.BaseURL, "/") + "/auth"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, loginURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create jw login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("jw login request: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			j.logger.Warn("close jw login response body failed", zap.Error(closeErr))
		}
	}()

	baseURL, err := url.Parse(j.cfg.JW.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse jw base url: %w", err)
	}
	return jar.Cookies(baseURL), nil
}

func (j *JWClient) GetCourse(ctx context.Context, cookies []*http.Cookie, term string, week int) (interface{}, error) {
	_ = ctx
	_ = cookies
	return map[string]interface{}{
		"term":  term,
		"week":  week,
		"items": []interface{}{},
	}, nil
}
