package result

import (
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type Result struct {
	Success bool        `json:"success"`
	Code    int         `json:"code"`
	Msg     string      `json:"msg"`
	Data    interface{} `json:"data"`
}

type BizError struct {
	Status int
	Code   int
	Msg    string
}

func (e *BizError) Error() string {
	if e == nil {
		return ""
	}
	return e.Msg
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
	CodeFollowSelf      = 1001
	CodeFollowRepeat    = 1002
	CodeFollowNotFollow = 1003
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
	Write(c, http.StatusOK, true, CodeSuccess, "成功", data)
}

func Fail(c *gin.Context, code int, msg string) {
	FailWithStatus(c, http.StatusOK, code, msg)
}

func FailWithStatus(c *gin.Context, status int, code int, msg string) {
	Write(c, status, false, code, msg, nil)
}

func Data(c *gin.Context, data interface{}) {
	Write(c, http.StatusOK, true, CodeData, "", data)
}

func SuccessMsg(c *gin.Context, msg string, data interface{}) {
	Write(c, http.StatusOK, true, CodeSuccess, msg, data)
}

func SuccessCodeMsg(c *gin.Context, code int, msg string, data interface{}) {
	Write(c, http.StatusOK, true, code, msg, data)
}

func Write(c *gin.Context, status int, success bool, code int, msg string, data interface{}) {
	c.JSON(status, Result{
		Success: success,
		Code:    code,
		Msg:     msg,
		Data:    normalizeData(data),
	})
}

func NewBizError(code int, msg string) error {
	return &BizError{
		Status: http.StatusOK,
		Code:   code,
		Msg:    msg,
	}
}

func NewBizStatusError(status, code int, msg string) error {
	return &BizError{
		Status: status,
		Code:   code,
		Msg:    msg,
	}
}

func BindJSON(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		status, code, msg := bindJSONError(err, obj)
		FailWithStatus(c, status, code, msg)
		return false
	}
	return true
}

func bindJSONError(err error, obj interface{}) (int, int, string) {
	if errors.Is(err, io.EOF) {
		return http.StatusBadRequest, CodeBodyIsNull, ErrBodyIsNull.Error()
	}
	if msg := validationErrorMessage(err, obj); msg != "" {
		return http.StatusBadRequest, CodeParamError, msg
	}
	return http.StatusBadRequest, CodeParamError, ErrParam.Error()
}

func validationErrorMessage(err error, obj interface{}) string {
	var validationErrs validator.ValidationErrors
	if !errors.As(err, &validationErrs) {
		return ""
	}

	var builder strings.Builder
	for _, fieldErr := range validationErrs {
		field := jsonFieldName(obj, fieldErr.StructField())
		if field == "" {
			field = fieldErr.Field()
		}
		builder.WriteString(field)
		builder.WriteString(": ")
		builder.WriteString(validationMessage(fieldErr))
		builder.WriteString("; ")
	}
	return builder.String()
}

func jsonFieldName(obj interface{}, fieldName string) string {
	typ := reflect.TypeOf(obj)
	if typ == nil {
		return ""
	}
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return ""
	}
	field, ok := typ.FieldByName(fieldName)
	if !ok {
		return ""
	}
	name := strings.Split(field.Tag.Get("json"), ",")[0]
	if name == "" || name == "-" {
		return ""
	}
	return name
}

func validationMessage(fieldErr validator.FieldError) string {
	switch fieldErr.Tag() {
	case "required":
		return "不能为空"
	case "oneof":
		return "取值非法"
	default:
		return fieldErr.Error()
	}
}

func HandleError(c *gin.Context, err error) {
	switch {
	case err == nil:
		Success(c, nil)
	case asBizError(err) != nil:
		bizErr := asBizError(err)
		Write(c, bizErr.Status, false, bizErr.Code, bizErr.Msg, nil)
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

func asBizError(err error) *BizError {
	var bizErr *BizError
	if errors.As(err, &bizErr) {
		return bizErr
	}
	return nil
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
