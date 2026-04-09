package school

import (
	"errors"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
)

const errMsgInvalidParam = "参数错误"

var (
	ErrJWClientUnavailable = errors.New("jw client not initialized")
	ErrUserNotFound        = bizerr.NotFound("用户不存在")
	ErrTermNotFound        = bizerr.NotFound("学期不存在")
	ErrTermDuplicated      = bizerr.Biz("学期已存在")
	ErrNeedCampusAuth      = bizerr.Forbidden("请先进行校园认证")
	ErrUserNotLogin        = bizerr.Unauthorized("用户未登录")
)
