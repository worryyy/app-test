package responses

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code       int    `json:"code"`
	Message    string `json:"message"`
	Data       any    `json:"data,omitempty"`
	RequestID  string `json:"requestId,omitempty"`
	HTTPStatus int    `json:"-"`
}

func New(code int, message string, httpStatus int) Response {
	return Response{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
	}
}

func (r Response) Resp(ctx *gin.Context) {
	ctx.JSON(r.httpStatus(), r.build(ctx, r.Message, nil))
}

func (r Response) RespData(ctx *gin.Context, data any) {
	ctx.JSON(r.httpStatus(), r.build(ctx, r.Message, data))
}

func (r Response) RespMessage(ctx *gin.Context, message string) {
	ctx.JSON(r.httpStatus(), r.build(ctx, message, nil))
}

func (r Response) RespMessageData(ctx *gin.Context, message string, data any) {
	ctx.JSON(r.httpStatus(), r.build(ctx, message, data))
}

func (r Response) build(ctx *gin.Context, message string, data any) Response {
	resp := r
	resp.Message = message
	resp.Data = data
	resp.RequestID = requestIDFromContext(ctx)
	return resp
}

func (r Response) httpStatus() int {
	if r.HTTPStatus == 0 {
		return 200
	}
	return r.HTTPStatus
}

func requestIDFromContext(ctx *gin.Context) string {
	if v, ok := ctx.Get("request_id"); ok && v != nil {
		return fmt.Sprint(v)
	}
	if v, ok := ctx.Get("requestId"); ok && v != nil {
		return fmt.Sprint(v)
	}
	return ""
}
