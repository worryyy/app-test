package agentchat

import "github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"

const errMsgInvalidParam = "参数错误"

var (
	ErrConversationNotFound     = bizerr.NotFound("agent 会话不存在")
	ErrConversationAccessDenied = bizerr.Forbidden("无权访问该 agent 会话")
	ErrConversationDeleteDenied = bizerr.Forbidden("无权删除该 agent 会话")
	ErrTurnContentRequired      = bizerr.Param("content 不能为空")
	ErrAgentRateLimited         = bizerr.Biz("请求过于频繁，请稍后再试")
	ErrAgentConversationBusy    = bizerr.Biz("当前会话正在处理中，请稍后再试")
	ErrAgentUserBusy            = bizerr.Biz("当前账号有未完成请求，请稍后再试")
	ErrAgentUnavailable         = bizerr.Internal("agent 服务暂时不可用")
	ErrAgentStreamProtocol      = bizerr.Internal("agent 流式响应异常")
	ErrInvalidWSMessage         = bizerr.Param("消息格式错误")
	ErrWSAuthTokenRequired      = bizerr.Unauthorized("token 不能为空")
	ErrWSAuthFailed             = bizerr.Unauthorized("ws 鉴权失败")
)
