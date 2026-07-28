# Agent Chat 接入说明

`Ecampus-go` 里的 agent 能力接入是一个独立域包 `internal/agentchat`。它不复用现有 `internal/chat` 的私信模型，也不让 agent 直接处理 JWT、限流或 HTTP 协议。边界很简单。`Ecampus-go` 负责鉴权、会话目录、限流、WebSocket 协议和 gRPC 调用。`goagent` 负责会话正文、上下文、能力执行和历史回放。

实现上也保持 `Ecampus-go` 现有域包约定。`handler` 只做绑定、调用和返回，`service` 直接操作 GORM 和 Redis，不额外引入 repo 层。

会话 owner 统一使用 JWT 里的 `root_user_id`。当前身份 `user_id` 和 `account_type` 仍然会透传给 agent，但它们只作为 metadata 进入 prompt，不决定 agent session 的归属。这样做之后，同一个根账号切换实名身份和匿名身份时，会落到同一条 agent 会话线上，目录和上下文都不会裂开。

对前端开放了两类入口。HTTP 入口放在 `/api/agent/**`，包括会话列表、历史、删除和同步问答。实时入口是 `/agent/ws`。WebSocket 的第一帧必须是 `{"type":"auth","token":"..."}`。服务端鉴权成功后返回 `auth_success`。后续发送 `{"type":"agent_turn.start", ...}` 会收到 `agent_turn.accepted`、`agent_turn.status`、`agent_turn.delta`、`agent_turn.final` 或 `agent_turn.error`。服务端可以并发处理多条 turn，但会受本地限流和 per-conversation lock 约束。

本地目录表叫 `agent_conversations`。它只保存目录元数据，例如 `session_id`、`root_user_id`、标题、最后一条用户摘要、最后一条 assistant 摘要和最近一次 request/trace。正文历史不落在 `Ecampus-go`，历史查询一律走 gRPC `GetSessionHistory` 回 agent。

配置项集中在 `configs/ecampus/application*.yml` 的 `agent` 段。`internal/app/ecampus` 启动时只初始化 gRPC client，不再自动建表。`agent_conversations` 由 `db/migrations/000001_create_agent_conversations.*.sql` 纳入统一迁移，并在业务服务启动前通过 `ecampus-migrate up` 执行。agent-core 未启动时，连接错误会在实际 RPC 调用时返回，再由 service 往上抛给接口层统一输出。
