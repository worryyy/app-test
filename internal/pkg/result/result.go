package result

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Result struct {
	Success bool        `json:"success"`
	Code    int         `json:"code"`
	Msg     string      `json:"msg"`
	Data    interface{} `json:"data"`
}

const (
	CodeSuccess         = 200
	CodeFail            = 400
	CodeUnknownError    = -1
	CodeParamError      = 1
	CodeNotExisted      = 3
	CodeForbidden       = 5
	CodeTokenError      = 10002
	CodeTokenNotExisted = 10003
	CodeTokenInvalid    = 10004
	CodeAuthNotExisted  = 10005
	CodeRTKNotExisted   = 10006
	CodeRTKUsed         = 10007
)

var (
	ErrParam           = errors.New("参数错误")
	ErrNotExisted      = errors.New("资源不存在")
	ErrForbidden       = errors.New("权限不足")
	ErrTokenError      = errors.New("token error")
	ErrTokenNotExisted = errors.New("token 不存在,或已过期")
	ErrTokenInvalid    = errors.New("token invalid")
	ErrAuthNotExisted  = errors.New("authorization 找不到")
	ErrRTKNotExisted   = errors.New("refresh token 不存在,或已过期")
	ErrRTKUsed         = errors.New("refresh token 已使用")
)

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Result{
		Success: true,
		Code:    CodeSuccess,
		Msg:     "成功",
		Data:    data,
	})
}

func Fail(c *gin.Context, code int, msg string) {
	c.JSON(http.StatusOK, Result{
		Success: false,
		Code:    code,
		Msg:     msg,
		Data:    nil,
	})
}

func HandleError(c *gin.Context, err error) {
	switch {
	case err == nil:
		Success(c, nil)
	case errors.Is(err, ErrParam):
		Fail(c, CodeParamError, err.Error())
	case errors.Is(err, ErrNotExisted):
		Fail(c, CodeNotExisted, err.Error())
	case errors.Is(err, ErrForbidden):
		Fail(c, CodeForbidden, err.Error())
	case errors.Is(err, ErrTokenError):
		Fail(c, CodeTokenError, err.Error())
	case errors.Is(err, ErrTokenNotExisted):
		Fail(c, CodeTokenNotExisted, err.Error())
	case errors.Is(err, ErrTokenInvalid):
		Fail(c, CodeTokenInvalid, err.Error())
	case errors.Is(err, ErrAuthNotExisted):
		Fail(c, CodeAuthNotExisted, err.Error())
	case errors.Is(err, ErrRTKNotExisted):
		Fail(c, CodeRTKNotExisted, err.Error())
	case errors.Is(err, ErrRTKUsed):
		Fail(c, CodeRTKUsed, err.Error())
	default:
		Fail(c, CodeUnknownError, "系统错误")
	}
}
