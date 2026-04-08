package theme

import "github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"

const errMsgInvalidParam = "参数错误"

var (
	ErrThemeNotFound       = bizerr.NotFound("主题不存在")
	ErrCampusThemeNotFound = bizerr.NotFound("校园主题不存在")
)
