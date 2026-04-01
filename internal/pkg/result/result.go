package result

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"

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
	CodeRTKError           = 10008
	CodeTokenChangeError   = 10009
	CodeFollowSelf         = 1001
	CodeFollowRepeat    = 1002
	CodeFollowNotFollow = 1003
)

var (
	ErrParam           = errors.New("请求参数错误")
	ErrIDZero          = errors.New("id 需要大于0")
	ErrNotExisted      = errors.New("资源不存在")
	ErrRepeated        = errors.New("资源重复")
	ErrForbidden       = errors.New("权限不足")
	ErrFileLimited     = errors.New("File size exceeds the limit.")
	ErrBodyIsNull      = errors.New("请求体不能为空")
	ErrTokenError      = errors.New("生成 token 出现错误")
	ErrTokenNotExisted = errors.New("token 不存在,或已过期")
	ErrTokenInvalid    = errors.New("token invalid")
	ErrAuthNotExisted  = errors.New("authorization 找不到")
	ErrRTKNotExisted   = errors.New("refresh_token 不存在, 或已过期")
	ErrRTKUsed         = errors.New("refresh_token 已被使用")
	ErrRTKError           = errors.New("续费失败")
	ErrTokenChangeError   = errors.New("token 状态更新失败")
	ErrFollowSelf         = errors.New("用户不可关注自己")
	ErrFollowRepeat       = errors.New("不可重复关注")
	ErrFollowNotFollow    = errors.New("不可对未关注用户取关")
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

// SuccessData mirrors Java's R.success().data(x) pattern.
// When data is nil, it returns {success:false, code:200, msg:"不存在", data:null}
// matching the Java Result.data(null) chaining behavior.
func SuccessData(c *gin.Context, data interface{}) {
	if data == nil || isNilInterface(data) {
		Write(c, http.StatusOK, false, CodeSuccess, "不存在", nil)
		return
	}
	Write(c, http.StatusOK, true, CodeSuccess, "成功", data)
}

func isNilInterface(v interface{}) bool {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Interface:
		return rv.IsNil()
	default:
		return false
	}
}

func SuccessMsg(c *gin.Context, msg string, data interface{}) {
	Write(c, http.StatusOK, true, CodeSuccess, msg, data)
}

func SuccessCodeMsg(c *gin.Context, code int, msg string, data interface{}) {
	Write(c, http.StatusOK, true, code, msg, data)
}

func Write(c *gin.Context, status int, success bool, code int, msg string, data interface{}) {
	c.Set("result_success", success)
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
		Fail(c, CodeParamError, ErrParam.Error())
	case errors.Is(err, ErrIDZero):
		Fail(c, CodeIDZero, ErrIDZero.Error())
	case errors.Is(err, ErrNotExisted):
		Fail(c, CodeNotExisted, ErrNotExisted.Error())
	case errors.Is(err, ErrRepeated):
		Fail(c, CodeRepeated, ErrRepeated.Error())
	case errors.Is(err, ErrForbidden):
		Fail(c, CodeForbidden, ErrForbidden.Error())
	case errors.Is(err, ErrFileLimited):
		Fail(c, CodeFileLimited, ErrFileLimited.Error())
	case errors.Is(err, ErrBodyIsNull):
		Fail(c, CodeBodyIsNull, ErrBodyIsNull.Error())
	case errors.Is(err, ErrTokenError):
		Fail(c, CodeTokenError, ErrTokenError.Error())
	case errors.Is(err, ErrTokenNotExisted):
		Fail(c, CodeTokenNotExisted, ErrTokenNotExisted.Error())
	case errors.Is(err, ErrTokenInvalid):
		Fail(c, CodeTokenInvalid, ErrTokenInvalid.Error())
	case errors.Is(err, ErrAuthNotExisted):
		Fail(c, CodeAuthNotExisted, ErrAuthNotExisted.Error())
	case errors.Is(err, ErrRTKNotExisted):
		Fail(c, CodeRTKNotExisted, ErrRTKNotExisted.Error())
	case errors.Is(err, ErrRTKUsed):
		Fail(c, CodeRTKUsed, ErrRTKUsed.Error())
	case errors.Is(err, ErrRTKError):
		Fail(c, CodeRTKError, ErrRTKError.Error())
	case errors.Is(err, ErrTokenChangeError):
		Fail(c, CodeTokenChangeError, ErrTokenChangeError.Error())
	case errors.Is(err, ErrFollowSelf):
		Fail(c, CodeFollowSelf, ErrFollowSelf.Error())
	case errors.Is(err, ErrFollowRepeat):
		Fail(c, CodeFollowRepeat, ErrFollowRepeat.Error())
	case errors.Is(err, ErrFollowNotFollow):
		Fail(c, CodeFollowNotFollow, ErrFollowNotFollow.Error())
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
	return normalizeValue(reflect.ValueOf(data))
}

var timeType = reflect.TypeOf(time.Time{})

func normalizeValue(v reflect.Value) interface{} {
	if !v.IsValid() {
		return nil
	}

	for v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return zeroValueForNilPointer(v.Type().Elem())
		}
		return normalizeValue(v.Elem())
	}

	if v.Type() == timeType {
		t := v.Interface().(time.Time)
		if t.IsZero() {
			return ""
		}
		return t.In(time.Local).Format("2006-01-02 15:04:05")
	}

	if v.CanInterface() {
		if _, ok := v.Interface().(json.Marshaler); ok {
			return v.Interface()
		}
	}

	switch v.Kind() {
	case reflect.Struct:
		return normalizeStruct(v)
	case reflect.Slice:
		if v.IsNil() {
			return []interface{}{}
		}
		out := make([]interface{}, v.Len())
		for i := 0; i < v.Len(); i++ {
			out[i] = normalizeValue(v.Index(i))
		}
		return out
	case reflect.Array:
		out := make([]interface{}, v.Len())
		for i := 0; i < v.Len(); i++ {
			out[i] = normalizeValue(v.Index(i))
		}
		return out
	case reflect.Map:
		if v.IsNil() {
			return nil
		}
		if v.Type().Key().Kind() != reflect.String {
			return v.Interface()
		}
		out := make(map[string]interface{}, v.Len())
		iter := v.MapRange()
		for iter.Next() {
			out[iter.Key().String()] = normalizeValue(iter.Value())
		}
		return out
	case reflect.String:
		return v.String()
	case reflect.Bool:
		return v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint()
	case reflect.Float32, reflect.Float64:
		return v.Float()
	default:
		if v.CanInterface() {
			return v.Interface()
		}
		return nil
	}
}

func normalizeStruct(v reflect.Value) map[string]interface{} {
	t := v.Type()
	out := make(map[string]interface{}, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		}

		name, skip := jsonFieldTagName(field)
		if skip {
			continue
		}

		value := v.Field(i)
		if field.Anonymous && name == "" {
			if merged, ok := normalizeValue(value).(map[string]interface{}); ok {
				for k, item := range merged {
					out[k] = item
				}
				continue
			}
		}

		if name == "" {
			name = field.Name
		}
		out[name] = normalizeValue(value)
	}
	return out
}

func jsonFieldTagName(field reflect.StructField) (string, bool) {
	tag := field.Tag.Get("json")
	if tag == "-" {
		return "", true
	}
	if tag == "" {
		return "", false
	}
	name := strings.Split(tag, ",")[0]
	if name == "-" {
		return "", true
	}
	return name, false
}

func zeroValueForNilPointer(t reflect.Type) interface{} {
	switch t.Kind() {
	case reflect.String, reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64:
		return reflect.Zero(t).Interface()
	case reflect.Slice:
		return reflect.MakeSlice(t, 0, 0).Interface()
	default:
		return nil
	}
}
