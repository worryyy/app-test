package metrics

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

const (
	businessCodeContextKey = "metrics_business_code"
	httpStatusContextKey   = "metrics_http_status"
)

func SetBusinessCode(ctx *gin.Context, code int) {
	if ctx == nil {
		return
	}
	ctx.Set(businessCodeContextKey, fmt.Sprint(code))
}

func BusinessCode(ctx *gin.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	value, ok := ctx.Get(businessCodeContextKey)
	if !ok || value == nil {
		return "", false
	}
	code := fmt.Sprint(value)
	if code == "" {
		return "", false
	}
	return code, true
}

func SetHTTPStatus(ctx *gin.Context, status int) {
	if ctx == nil {
		return
	}
	ctx.Set(httpStatusContextKey, fmt.Sprint(status))
}

func HTTPStatus(ctx *gin.Context, fallback int) string {
	if ctx == nil {
		return fmt.Sprint(fallback)
	}
	value, ok := ctx.Get(httpStatusContextKey)
	if !ok || value == nil {
		return fmt.Sprint(fallback)
	}
	status := fmt.Sprint(value)
	if status == "" {
		return fmt.Sprint(fallback)
	}
	return status
}
