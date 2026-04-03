package chat

import "github.com/Milchstrassse/Ecampus-go/internal/pkg/bizerr"

const errMsgInvalidParam = "参数错误"

var (
	ErrConversationNotFound     = bizerr.NotFound("会话不存在")
	ErrConversationPeerNotFound = bizerr.NotFound("会话中未找到聊天对象")
	ErrConversationUserNotFound = bizerr.NotFound("会话中用户信息未找到")
	ErrConversationAccessDenied = bizerr.Biz("无权访问该会话")
	ErrConversationDeleteDenied = bizerr.Biz("无权删除该会话")
	ErrConversationUpdateFailed = bizerr.Biz("更新会话失败")
	ErrConversationDeleteFailed = bizerr.Biz("删除会话失败")
	ErrMessageParseFailed       = bizerr.Biz("消息解析失败")

	ErrNotificationTypeRequired = bizerr.Param("type不能为空")
	ErrConversationIDRequired   = bizerr.Param("会话ID不能为空")
	ErrTargetUserIDRequired     = bizerr.Param("目标用户ID不能为空")
	ErrReceiverIDRequired       = bizerr.Param("接收者ID不能为空")
	ErrLastMessageIDRequired    = bizerr.Param("最后一条消息ID不能为空")
)
