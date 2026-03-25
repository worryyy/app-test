package result

import (
	"errors"
	"io"
	"net/http"
	"reflect"

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
	CodeData            = 0
	CodeFail            = 400
	CodeUnknownError    = -1
	CodeParamError      = 1
	CodeIDZero          = 2
	CodeNotExisted      = 3
	CodeRepeated        = 4
	CodeForbidden       = 5
	CodeFileLimited     = 6
	CodeBodyIsNull      = 7
	CodeTokenError      = 10002
	CodeTokenNotExisted = 10003
	CodeTokenInvalid    = 10004
	CodeAuthNotExisted  = 10005
	CodeRTKNotExisted   = 10006
	CodeRTKUsed         = 10007
	CodeRTKError        = 10008
)

var (
	ErrParam           = errors.New("参数错误")
	ErrIDZero          = errors.New("id 需要大于0")
	ErrNotExisted      = errors.New("资源不存在")
	ErrRepeated        = errors.New("资源重复")
	ErrForbidden       = errors.New("权限不足")
	ErrFileLimited     = errors.New("File size exceeds the limit.")
	ErrBodyIsNull      = errors.New("请求体不能为空")
	ErrTokenError      = errors.New("token error")
	ErrTokenNotExisted = errors.New("token 不存在,或已过期")
	ErrTokenInvalid    = errors.New("token invalid")
	ErrAuthNotExisted  = errors.New("authorization 找不到")
	ErrRTKNotExisted   = errors.New("refresh token 不存在,或已过期")
	ErrRTKUsed         = errors.New("refresh token 已使用")
	ErrRTKError        = errors.New("续费失败")
)

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Result{
		Success: true,
		Code:    CodeSuccess,
		Msg:     "成功",
		Data:    normalizeData(data),
	})
}

func Fail(c *gin.Context, code int, msg string) {
	FailWithStatus(c, http.StatusOK, code, msg)
}

func FailWithStatus(c *gin.Context, status int, code int, msg string) {
	c.JSON(status, Result{
		Success: false,
		Code:    code,
		Msg:     msg,
		Data:    nil,
	})
}

func Data(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Result{
		Success: true,
		Code:    CodeData,
		Msg:     "",
		Data:    normalizeData(data),
	})
}

func BindJSON(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		if errors.Is(err, io.EOF) {
			FailWithStatus(c, http.StatusBadRequest, CodeBodyIsNull, ErrBodyIsNull.Error())
			return false
		}
		FailWithStatus(c, http.StatusBadRequest, CodeParamError, ErrParam.Error())
		return false
	}
	return true
}

func HandleError(c *gin.Context, err error) {
	switch {
	case err == nil:
		Success(c, nil)
	case errors.Is(err, ErrParam):
		Fail(c, CodeParamError, err.Error())
	case errors.Is(err, ErrIDZero):
		Fail(c, CodeIDZero, err.Error())
	case errors.Is(err, ErrNotExisted):
		Fail(c, CodeNotExisted, err.Error())
	case errors.Is(err, ErrRepeated):
		Fail(c, CodeRepeated, err.Error())
	case errors.Is(err, ErrForbidden):
		Fail(c, CodeForbidden, err.Error())
	case errors.Is(err, ErrFileLimited):
		Fail(c, CodeFileLimited, err.Error())
	case errors.Is(err, ErrBodyIsNull):
		Fail(c, CodeBodyIsNull, err.Error())
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
	case errors.Is(err, ErrRTKError):
		Fail(c, CodeRTKError, err.Error())
	default:
		Fail(c, CodeUnknownError, "系统错误")
	}
}

func normalizeData(data interface{}) interface{} {
	if data == nil {
		return nil
	}

	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Slice && v.IsNil() {
		return reflect.MakeSlice(v.Type(), 0, 0).Interface()
	}
	return data
}
