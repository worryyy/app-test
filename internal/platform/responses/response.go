package responses

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Success    bool   `json:"success"`
	Code       int    `json:"code"`
	HTTPStatus int    `json:"httpstatus"`
	Msg        string `json:"msg"`
	Data       any    `json:"data,omitempty"`
	RequestID  string `json:"requestId,omitempty"`
}

func New(success bool, httpStatus int, message string) Response {
	_ = success

	normalizedHTTPStatus := normalizeHTTPStatus(httpStatus)
	normalizedCode := normalizeCode(normalizedHTTPStatus)

	return Response{
		Success:    normalizedCode == CodeSuccess,
		Code:       normalizedCode,
		HTTPStatus: normalizedHTTPStatus,
		Msg:        message,
	}
}

func (r Response) Resp(ctx *gin.Context) {
	r.write(ctx, r.Msg, nil)
}

func (r Response) RespData(ctx *gin.Context, data any) {
	r.write(ctx, r.Msg, data)
}

func (r Response) RespMessage(ctx *gin.Context, message string) {
	r.write(ctx, message, nil)
}

func (r Response) RespMessageData(ctx *gin.Context, message string, data any) {
	r.write(ctx, message, data)
}

func (r Response) write(ctx *gin.Context, message string, data any) {
	resp := r.build(ctx, message, data)
	ctx.JSON(resp.Code, resp)
}

func (r Response) build(ctx *gin.Context, message string, data any) Response {
	resp := r
	httpStatus := resp.HTTPStatus
	if httpStatus == 0 {
		httpStatus = resp.Code
	}
	resp.HTTPStatus = normalizeHTTPStatus(httpStatus)
	resp.Code = normalizeCode(resp.HTTPStatus)
	resp.Success = resp.Code == CodeSuccess
	resp.Msg = message
	resp.Data = data
	resp.RequestID = requestIDFromContext(ctx)
	return resp
}

func (r Response) withMessage(message string) Response {
	r.Msg = message
	return r
}

func normalizeHTTPStatus(code int) int {
	if code >= http.StatusContinue && code <= 999 {
		return code
	}
	return http.StatusInternalServerError
}

func normalizeCode(httpStatus int) int {
	if httpStatus >= http.StatusOK && httpStatus < http.StatusMultipleChoices {
		return CodeSuccess
	}
	return CodeFail
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
