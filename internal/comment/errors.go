package comment

import "github.com/Milchstrassse/Ecampus-go/internal/pkg/bizerr"

const errMsgInvalidParam = "参数错误"

var (
	ErrTopicNotFound                = bizerr.NotFound("帖子不存在")
	ErrCommentNotFound              = bizerr.NotFound("评论不存在")
	ErrUserNotFound                 = bizerr.NotFound("用户不存在")
	ErrInvalidRootID                = bizerr.Param("请传入有效的root_id")
	ErrTargetUserIDRequired         = bizerr.Param("目标用户id不能为空")
	ErrCommentDeleteFailed          = bizerr.Biz("删除评论失败")
	ErrCommentLikeFailed            = bizerr.Biz("点赞失败")
	ErrCommentUnlikeFailed          = bizerr.Biz("取消点赞失败")
	ErrCommentLikeNotFound          = bizerr.Biz("还没有对该评论进行点赞")
	ErrAnonymousCommentForbidden    = bizerr.Biz("匿名用户禁止评论非匿名帖")
	ErrCommentSelfRolePlayForbidden = bizerr.Biz("禁止左右脑互搏和自导自演")
)
