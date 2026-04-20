# Module Layout Convention

后续各业务模块统一参考 `internal/topic` 的组织方式。

## 完整文件清单

每个业务模块可能包含以下文件：

| 文件 | 职责 | 备注 |
|------|------|------|
| `model.go` | 核心实体（同时携带 gorm/bson/json tag） | 两服务共享 |
| `model_req.go` | 请求体结构 | |
| `model_resp.go` | 响应体结构 | 可选，字段少时可放 model.go |
| `model_<业务>.go` | 子域模型（如 follow） | 可选，量大时拆出 |
| `errors.go` | 模块级错误定义 | 使用 `bizerr` 包级变量 |
| `service.go` | 主接口 + `NewService` | |
| `service_<业务>.go` | 子域能力 | |
| `repository.go` | 仓储基座（struct + 连接获取） | |
| `repository_<业务>.go` | 具体数据操作 | |
| `handler.go` | 主 HTTP handler + `NewHandler` | |
| `handler_<业务>.go` | 子域 handler | 与 service 拆分对称 |
| `handler_bindings.go` | 请求绑定结构体 | |
| `handler_helpers.go` | 模块独有的 handler 辅助函数 | 仅放该模块独有的 |
| `admin.go` | 管理端 handler | |
| `routes.go` | `RegisterPublicRoutes` / `RegisterProtectedRoutes` | |
| `routes_admin.go` | `RegisterAdminRoutes` | |
| `producer.go` | MQ 事件发送接口 | 可选 |
| `page_result.go` | 分页结果 | **旧模块遗留**（见第 10 节已知债务）；新模块直接用 `internal/platform/pagination.PageResult[T]` |

## Service 拆分规则

### 什么留在 `service.go`

属于模块**主流程**的接口：

- Create
- Delete
- Update
- GetByID / Detail
- Mine / ListMine
- 主业务列表接口

### 什么拆到 `service_<业务>.go`

属于**扩展能力或子业务**的接口：

- Search — `service_search.go`
- Social（点赞/收藏） — `service_social.go`
- AdminOps — `service_admin.go`
- Identity — `service_identity.go`
- Follow — `service_follow.go`
- Notify — `service_notify.go`
- 微信等外部集成 — `service_wx.go`

## Repository 拆分规则

### 什么留在 `repository.go`

仓储**基座能力**：

- `Repository` 结构体定义
- `NewRepository`
- `gormDB(ctx)` / `mongoCollection(name)`
- 模块范围内通用常量（集合名等）
- 所有子仓储都会复用的基础内部类型

### 什么拆到 `repository_<业务>.go`

所有**具体数据库操作**：

- 主实体 CRUD — `repository_topic.go`
- 搜索查询 — `repository_search.go`
- 关系操作（点赞/收藏/关注） — `repository_social.go`
- 聚合查询 / 统计更新
- 审核 / 状态流转

## Handler 拆分规则

Handler 的拆分与 Service 保持**对称**：

- 主流程 handler 放 `handler.go`
- 子域 handler 放 `handler_<业务>.go`（如 `handler_follow.go`、`handler_identity.go`）
- 管理端 handler 统一用 `admin.go`
- 请求绑定结构体（query/uri struct + Resolved 方法）放 `handler_bindings.go`

### handler_helpers.go 的定位

只放该模块**独有**的 handler 辅助函数（如 `writeTopicListResult`）。

跨模块通用的 handler 工具函数（如 `bindJSON`、`bindQuery`、`pageSize`）目标是统一收敛到 `internal/platform/` 下的公共工具包（计划路径 `internal/platform/ginutil/`，尚未落地）。在该公共包落地之前，各模块保留现有 `handler_helpers.go` 实现；迁移时按独立 PR 进行，不与业务改动混合（见 AGENTS.md 第 3 节迁移原则）。

## Routes 规则

每个模块提供路由注册函数，由 `internal/app/ecampus/routes.go` 统一调用：

- `routes.go` — 暴露 `RegisterPublicRoutes` / `RegisterProtectedRoutes`
- `routes_admin.go` — 暴露 `RegisterAdminRoutes`
- 路由注册函数只做路由映射，不含业务逻辑

## Errors 规则

模块级错误统一在 `errors.go` 中定义为包级变量：

```go
var (
    ErrTopicNotFound = bizerr.NotFound("帖子不存在")
    ErrTopicAlreadyLiked = bizerr.Biz("已经点赞过该帖子")
)
```

使用 `bizerr.NotFound` / `bizerr.Biz` / `bizerr.Param` / `bizerr.Unauthorized` 构造。
不在 service 函数内部临时创建错误，除非错误信息需要动态拼接。

## 辅助函数规则

基于**使用范围**决定放置位置，而非行数：

| 使用范围 | 放置位置 |
|---------|---------|
| 仅在单个函数内使用 | 内联 |
| 同一文件 2+ 处使用 | 同文件末尾提取为私有函数 |
| 同模块跨文件使用 | 放入对应的 `service_<业务>.go` 或 `repository_<业务>.go` |
| 跨模块使用 | 提取到 `internal/platform/` |

**不再使用 `*_helpers.go` 作为默认存放位置。** 只有在辅助函数确实跨多个子域复用、且不适合归入任何一个子域文件时，才保留 helpers 文件。

## 跨模块共享

| 共享内容 | 放置位置 | 状态 |
|---------|---------|------|
| 分页结果 `PageResult[T]` | `internal/platform/pagination/` | 已落地 |
| 业务错误构造 | `internal/platform/bizerr/` | 已落地 |
| 统一响应 | `internal/platform/responses/` | 已落地 |
| JWT 工具（用户端 / 管理端） | `internal/platform/jwtutil/`、`internal/platform/adminjwt/` | 已落地 |
| 加密工具 | `internal/platform/encrypt/` | 已落地 |
| Redis key 规则 | `internal/platform/rediskey/` | 已落地 |
| 雪花 ID | `internal/platform/snowflake/` | 已落地 |
| 外部服务集成（微信、COS） | `internal/integration/<服务>/` | 已落地 |
| Handler 工具（bind, pageSize） | 计划路径 `internal/platform/ginutil/` | **未落地**，现仍在各模块 `handler_helpers.go` |

## Topic 模板参考

`internal/topic` 是标准参考模块：

- [service.go](../internal/topic/service.go) — 主接口：创建、删除、详情、更新、我的帖子、目标用户帖子
- [service_search.go](../internal/topic/service_search.go) — 搜索能力
- [service_social.go](../internal/topic/service_social.go) — 点赞、收藏
- [service_topic.go](../internal/topic/service_topic.go) — 帖子相关辅助逻辑
- [repository.go](../internal/topic/repository.go) — 仓储基座
- [repository_topic.go](../internal/topic/repository_topic.go) — 帖子主数据操作
- [repository_search.go](../internal/topic/repository_search.go) — 搜索数据操作
- [repository_social.go](../internal/topic/repository_social.go) — 点赞、收藏数据操作
- [routes.go](../internal/topic/routes.go) — 路由注册

## 迁移顺序

后续模块整理时按下面顺序做：

1. 先确定模块的"主接口"范围
2. 把主接口留在 `service.go`
3. 把非主接口按子域拆到 `service_<业务>.go`
4. 把 `repository.go` 收缩为仓储基座
5. 把所有具体数据库操作迁到 `repository_<业务>.go`
6. Handler 拆分与 Service 保持对称
7. 提取跨模块重复代码到 `internal/platform/`

## 10. 已知债务

下列是**当前代码与本规范的偏差**。修 bug 时**不要顺手重写**；迁移走独立 PR（见 AGENTS.md 第 3 节）。

| 债务 | 现状 | 迁移方向 |
|------|------|---------|
| `internal/user` 没有 `service.go` | 主流程散落在 `service_user.go` / `service_admin.go` / `service_admin_user.go` / `service_follow.go` / `service_identity.go` / `service_wx.go` | 新模块不要模仿；迁移时抽出主流程进 `service.go` |
| `page_result.go` 尚未消亡 | `internal/chat`、`internal/comment`、`internal/user`、`internal/topic` 四模块仍在用；`writeXxxListResult` 辅助函数的签名依赖 `*PageResult[T]` | 连带重写 list handler 辅助函数时再迁；不要只换类型不换调用点 |
| `handler_helpers.go` 跨模块重复 | `chat` / `comment` / `file` / `school` / `sensitive` / `topic` / `user` 共 7 个模块各自定义 `bindJSON` / `bindQuery` / `bindURI`，文本完全相同 | 目标收敛到 `internal/platform/ginutil/`（**未落地**）；落地前各模块保留现状 |
| 模块级错误常量不统一 | 稳定错误应在 `errors.go`，但部分模块（如 `comment`）用 `errMsgInvalidParam` 常量 + 即时 `bizerr.Param(...)`，未提成 `ErrXxx` | 修 bug 时若遇到动态拼接错误，优先新增包级变量 |
| `internal/topic` 有 `db_columns.go` 抽列名常量 | 其它模块未普及 | 仅在**跨多个 repository 文件复用**同一列名时新建；单文件内别提 |
| 测试覆盖稀疏 | 目前只有 `topic` / `comment` / `user` 少数 service 有窄测试 | 新增兼容字段 / 日期格式 / 错误分支时补窄行为测试；暂不强制补历史 |

## 默认约定

从现在开始，除非有特殊说明，新模块和旧模块重构都默认遵守本文档。与第 10 节不一致的地方以"现有稳定行为不回归"为前提逐步迁移，不要求一次性改完。
