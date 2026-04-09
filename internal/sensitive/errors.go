package sensitive

import "github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"

const errMsgInvalidParam = "参数错误"

var (
	ErrSensitiveWordNotFound = bizerr.NotFound("敏感词不存在")
	ErrSensitiveWordExists   = bizerr.Biz("敏感词已存在")
)
