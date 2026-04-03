package topic

import "github.com/Milchstrassse/Ecampus-go/internal/pkg/bizerr"

const errMsgInvalidParam = "参数错误"

var (
	ErrTopicNotFound            = bizerr.NotFound("帖子不存在")
	ErrTargetUserNotFound       = bizerr.NotFound("目标用户不存在")
	ErrUserNotFound             = bizerr.NotFound("用户不存在")
	ErrThemeNotFound            = bizerr.NotFound("主题不存在")
	ErrAnonymousAccountNotFound = bizerr.Biz("匿名账号不存在")
	ErrTopicAlreadyLiked        = bizerr.Biz("已经点赞过了")
	ErrTopicCollectionLimit     = bizerr.Biz("收藏数量已达上限")
)
