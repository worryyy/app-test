package school

import "github.com/Milchstrassse/Ecampus-go/internal/pkg/bizerr"

const errMsgInvalidParam = "参数错误"

var (
	ErrUserNotFound    = bizerr.NotFound("用户不存在")
	ErrTermNotFound    = bizerr.NotFound("学期不存在")
	ErrTermDuplicated  = bizerr.Biz("学期已存在")
	ErrNeedCampusAuth  = bizerr.Biz("请先进行校园认证")
	ErrUserNotLogin    = bizerr.Biz("用户未登录")
)
