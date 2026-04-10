package responses

import (
	"errors"
	"strings"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
	"github.com/gin-gonic/gin"
)

func FromError(err error) Response {
	if err == nil {
		return Success
	}

	var be *bizerr.Error
	if errors.As(err, &be) {
		resp := defaultResponseForStatus(be.Code)
		if strings.TrimSpace(be.Message) != "" {
			return resp.withMessage(be.Message)
		}
		return resp
	}

	return InternalErr
}

func defaultResponseForStatus(code int) Response {
	switch normalizeHTTPStatus(code) {
	case HTTPStatusUnauthorized:
		return Unauthorized
	case HTTPStatusForbidden:
		return Forbidden
	case HTTPStatusNotFound:
		return NotFound
	case HTTPStatusInternalErr:
		return InternalErr
	default:
		return BizErr
	}
}

func Fail(ctx *gin.Context, err error) {
	recordError(ctx, err)
	FromError(err).Resp(ctx)
}

func FailMessage(ctx *gin.Context, err error, message string) {
	recordError(ctx, err)
	resp := FromError(err)
	resp.RespMessage(ctx, message)
}

func recordError(ctx *gin.Context, err error) {
	if ctx == nil || err == nil {
		return
	}
	_ = ctx.Error(err)
}
