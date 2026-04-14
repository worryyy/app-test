# 项目编码规范与模块布局


## 1. 适用范围

- 适用于 `cmd/ecampus`、`cmd/ecampus-crm` 两个入口服务。
- 适用于 `internal/` 下所有业务模块、公共能力和装配层代码。
- 本文优先描述代码里已经稳定存在的模式；没有在代码中形成共识的内容，不额外发明规则。

## 2. 代码库结构

### 2.1 双服务结构

| 服务 | 入口 | 主要路由前缀 | 职责 |
|------|------|-------------|------|
| `ecampus` | `cmd/ecampus/main.go` | `/api/**` | 用户端 API、聊天、文件、校园认证、MQ consumer |
| `ecampus-crm` | `cmd/ecampus-crm/main.go` | `/admin/**` | 管理端 API |

两个服务共享 `internal/` 中的大部分业务实现，差异主要体现在应用装配层和路由挂载层。

### 2.2 目录边界

| 目录 | 角色 |
|------|------|
| `internal/app/bootstrap/` | 基础设施初始化、Gin 引擎创建、HTTP Server 启停 |
| `internal/app/ecampus/` | 用户端装配：创建 service、handler、producer，并注册用户端路由 |
| `internal/app/ecampuscrm/` | 管理端装配：创建 service、handler，并注册管理端路由 |
| `internal/<module>/` | 业务模块，如 `user`、`topic`、`comment`、`chat`、`school` |
| `internal/platform/` | 通用能力，如 `responses`、`bizerr`、`pagination`、`jwtutil`、`adminjwt` |
| `internal/integration/` | 外部服务集成，如微信、COS |
| `internal/middleware/` | 通用 Gin 中间件 |
| `internal/mq/` | MQ 生产者、消费者和拓扑定义 |

## 3. 分层与装配规范

### 3.1 分层方式

当前代码的主流结构是：

`Handler -> Service -> Repository`

具体边界如下：

- Handler 负责 HTTP 入口、参数绑定、从上下文取用户信息、调用 service、写响应。
- Service 负责业务编排、业务校验、调用 repository、调用外部依赖、组织聚合结果。
- Repository 负责 MySQL / MongoDB 的具体访问与查询组装。

Repository 在当前项目里不是独立抽象层，不额外定义接口作为测试替身；它更像是 service 内部的数据访问拆分。

### 3.2 装配职责

应用装配统一放在 `internal/app/ecampus/app.go` 和 `internal/app/ecampuscrm/app.go`：

- 基础设施先通过 `bootstrap.LoadInfrastructure` 初始化。
- service 在 app 层创建，并在这里注入 DB、Mongo、Redis、config、logger、producer。
- handler 也在 app 层创建，再统一注册到路由。
- MQ producer、consumer、敏感词过滤器等跨模块依赖，也在 app 层串起来。

业务模块内部不要自己创建全局单例，不要在模块内部偷偷初始化外部依赖。

### 3.3 构造函数风格

代码里已形成的构造模式：

- `NewService(...)`
- `NewRepository(...)`
- `NewHandler(...)`
- `NewAdminHandler(...)`

常见 service 构造函数签名形态：

```go
func NewService(
    db *gorm.DB,
    mongoDB *mongo.Database,
    rds *redis.Client,
    cfg *config.Config,
    logger *zap.Logger,
) *Service
```

当前代码中的一致做法：

- 依赖通过构造函数注入。
- 不使用 DI 框架。
- `logger == nil` 时通常回退为 `zap.NewNop()`。

## 4. Handler 编码风格

### 4.1 Handler 结构

典型写法是：

```go
type Handler struct {
    svc *Service
}

func NewHandler(svc *Service) *Handler {
    return &Handler{svc: svc}
}
```

管理端单独使用 `AdminHandler`，和用户端 handler 分开。

### 4.2 Handler 的职责边界

从现有模块看，Handler 应只做以下事情：

1. 绑定 URI / Query / JSON 参数。
2. 从中间件上下文读取 claims、userID 等认证信息。
3. 调用 service。
4. 使用 `responses` 统一输出。

不应在 Handler 中直接写数据库访问、复杂业务规则或跨模块编排。

### 4.3 参数绑定

当前代码统一使用 `ShouldBind*` 家族，并在模块内封装了轻量 helper：

- `bindJSON`
- `bindQuery`
- `bindURI`

helper 的行为在多个模块中高度一致：

- 调用 `c.ShouldBindJSON` / `c.ShouldBindQuery` / `c.ShouldBindUri`
- 绑定失败时直接 `responses.Fail(c, bizerr.Param(...))`
- 返回 `bool`，Handler 用 early return 退出

典型结构：

```go
var req CreateCommentReq
if !bindJSON(c, &req) {
    return
}
```

### 4.4 上下文传递

当前项目的统一风格是：

- Handler 用 `*gin.Context`
- Service / Repository 用 `context.Context`
- Handler 调 Service 时统一传 `c.Request.Context()`

这条规则在 `user`、`topic`、`comment`、`school`、`theme` 等模块中都一致。

### 4.5 认证信息读取

当前代码的主流写法是通过 middleware 在 Gin context 中写入 claims，然后在 Handler 中读取：

- `middleware.GetUserID(c)`
- `currentClaims(c)`
- `currentUserID(c)`

Handler 不自己解析 JWT，不在业务模块里重复做认证逻辑。

## 5. Service 编码风格

### 5.1 Service 结构

Service 一般持有：

- `repo`
- `logger`
- `cfg`
- `redis`
- producer / 外部 client / filter 等可选依赖

例如：

```go
type Service struct {
    repo   *Repository
    logger *zap.Logger
}
```

### 5.2 Service 职责

根据当前代码，Service 主要负责：

- 参数语义校验
- 领域规则校验
- 调用 repository
- 调用 MQ、敏感词过滤器、外部接口
- 组装最终返回模型
- 兜底包装错误

Service 中常见模式：

- 对无效参数先返回 `bizerr.Param(...)`
- 对 repo 或外部依赖错误用 `bizerr.InternalWrap(...)`
- 对正常业务失败返回 `bizerr.Biz(...)` / `bizerr.NotFound(...)` / `bizerr.Forbidden(...)`
- 使用 `zap` 记录降级日志，例如 MQ 发送失败但主流程继续

### 5.3 错误包装

本地代码里已经形成两层错误处理方式：

1. Go 原生错误包装：`fmt.Errorf("...: %w", err)`，多出现在 bootstrap、platform、integration 层。
2. 业务错误包装：`bizerr.*`，多出现在业务 service 层。

如果错误要暴露给接口层，优先收敛成 `bizerr`。

## 6. Repository 编码风格

### 6.1 Repository 结构

Repository 的基础形态已经比较固定：

```go
type Repository struct {
    db      *gorm.DB
    mongoDB *mongo.Database
}
```

### 6.2 基础能力放在 `repository.go`

当前代码和 `module-layout` 的共识是，`repository.go` 只放仓储基座：

- `Repository` 结构定义
- `NewRepository`
- `gormDB(ctx)`
- `mongoCollection(name)`
- 模块范围内通用常量
- 所有子仓储共享的基础内部类型

### 6.3 具体数据操作拆到子文件

具体 CRUD、关系操作、搜索查询、聚合更新等，拆到：

- `repository_topic.go`
- `repository_social.go`
- `repository_search.go`
- `repository_user.go`

也就是按“子域/能力”而不是按数据库类型继续拆。

## 7. 响应与错误规范

### 7.1 默认 JSON 响应入口

绝大多数 JSON 接口统一使用 `internal/platform/responses`：

- `responses.Success.Resp(c)`
- `responses.Success.RespData(c, data)`
- `responses.Fail(c, err)`
- `responses.FailMessage(c, err, message)`

只有文件、二进制内容、纯字符串等非 JSON 场景，才直接用 Gin 原生输出，例如 `c.String(...)`。

### 7.2 Response 结构

当前统一响应结构定义在 `internal/platform/responses/response.go`：

```go
type Response struct {
    Success    bool   `json:"success"`
    Code       int    `json:"code"`
    HTTPStatus int    `json:"httpstatus"`
    Msg        string `json:"msg"`
    Data       any    `json:"data,omitempty"`
    RequestID  string `json:"requestId,omitempty"`
}
```

要注意本项目的一个本地约定：

- 实际 HTTP 响应码统一收敛为 `200` 或 `400`
- 更细的状态语义通过响应体里的 `httpstatus` 表达

也就是说，接口层判断成功失败时，不能只看 `httpstatus`，要按现有响应结构理解。

### 7.3 业务错误

业务错误统一放在 `internal/platform/bizerr/`，常用构造函数包括：

- `bizerr.Param`
- `bizerr.Biz`
- `bizerr.NotFound`
- `bizerr.Unauthorized`
- `bizerr.Forbidden`
- `bizerr.Internal`

以及对应的 `*Wrap` 版本。

模块内的稳定业务错误，优先定义在 `errors.go` 里作为包级变量，例如：

```go
var (
    ErrTopicNotFound = bizerr.NotFound("帖子不存在")
)
```

代码中已经形成的实际习惯是：

- 固定错误优先定义成包级变量
- 带动态上下文的错误在 service 中即时构造

### 7.4 Request ID 与错误记录

`responses.Response` 会从 Gin context 里读取：

- `request_id`
- `requestId`

`responses.Fail` 同时会把错误写入 `c.Error(err)`，这样请求日志中间件可以统一采集错误信息。

## 8. 日志、中间件与启动方式

### 8.1 Gin 引擎创建

统一在 `internal/app/bootstrap/http.go` 中创建引擎：

- `gin.New()`
- `engine.HandleMethodNotAllowed = true`
- `gin.Recovery()`
- `middleware.CORS()`
- `RegisterCommonRoutes(engine)`

自定义 validator 也在 bootstrap 层统一注册，而不是散落到各业务模块里。

### 8.2 请求日志

请求日志由 `internal/middleware/log.go` 中的 `RequestLog` 统一处理，当前字段包括：

- `method`
- `path`
- `status`
- `latency`
- `clientIP`
- `error`

现有代码里还有两个值得保持的细节：

- 请求路径里的敏感 query 参数会被打码
- `c.Errors` 中记录的错误会被串起来写入日志

### 8.3 认证与权限中间件

中间件挂载主要在 app 路由装配层完成，而不是在模块内部零散处理：

- 用户端：`JWTAuth`、`RequestLog`、`CertifiedUserCheck`
- 管理端：`AdminJWTAuth`、`RequestLog`、`AdminPermissionCheck`

模块本身只暴露 `Register*Routes`，中间件怎么挂由上层决定。

### 8.4 HTTP Server

当前 bootstrap 里的 HTTP server 约定：

- 使用 `http.Server`
- 显式设置 `ReadHeaderTimeout`
- 用 `signal` + `Shutdown` 做优雅退出

这部分属于项目已有运行方式，新服务或重构时应保持一致。

## 9. 模型与字段风格

### 9.1 模型定义

从现有代码看，模型通常直接承担“数据库映射 + JSON 输出”双重职责：

- MySQL 模型常见 `gorm + json`
- Mongo 模型常见 `bson + json`
- 某些模型同时携带多类 tag

例如：

```go
type User struct {
    ID          int64  `gorm:"column:id" json:"id"`
    AccountType string `gorm:"column:account_type" json:"accountType"`
}
```

### 9.2 字段命名

接口字段名以现有 API 兼容为优先，当前大量使用 camelCase，例如：

- `accountType`
- `rootUserId`
- `createdTime`
- `themeId`

如果为了兼容旧客户端或旧字段命名，需要保留兼容字段，也可以像现有 query struct 一样同时保留主字段和兼容字段。

### 9.3 序列化兼容

当前代码里允许模型为了兼容现有接口格式自定义 `MarshalJSON`，例如评论相关模型会把 `time.Time` 格式化成字符串日期输出。

也就是说，序列化兼容优先级高于“所有模型都必须零定制”。

## 10. 分页与空列表约定

当前项目已经有统一分页结构 `internal/platform/pagination.PageResult[T]`：

```go
type PageResult[T any] struct {
    Data    []T   `json:"data"`
    Current int   `json:"current"`
    Total   int64 `json:"total"`
    Size    int   `json:"size"`
}
```

已经稳定存在的分页习惯：

- 默认 `page=1`
- 默认 `size=15`
- `size <= 0` 时回退到默认值
- `data == nil` 时返回空切片而不是 `null`

旧模块里仍然存在模块内自带的 `page_result.go`。新代码优先复用 `internal/platform/pagination/`，旧代码可逐步迁移。

## 11. 模块布局规范

`docs/module-layout.md` 里的主体方向与当前代码基本一致，下面整合为本仓库的落地版本。

### 11.1 推荐文件清单

| 文件 | 角色 | 说明 |
|------|------|------|
| `model.go` | 核心模型 | 必备 |
| `model_req.go` | 请求结构体 | 按需 |
| `model_resp.go` | 响应结构体 | 按需 |
| `model_<业务>.go` | 子域模型 | 按需 |
| `errors.go` | 模块级稳定错误 | 推荐 |
| `service.go` | 主流程 service | 必备 |
| `service_<业务>.go` | 子域 service | 按需 |
| `repository.go` | 仓储基座 | 必备 |
| `repository_<业务>.go` | 子域 repository | 按需 |
| `handler.go` | 用户端主 handler | 用户端模块使用 |
| `handler_<业务>.go` | 子域 handler | 按需 |
| `handler_bindings.go` | URI / Query / JSON 绑定结构 | 推荐 |
| `handler_helpers.go` | 模块私有 handler helper | 仅在确实需要时保留 |
| `admin.go` | 管理端 handler | 有管理端接口时使用 |
| `routes.go` | 用户端路由注册 | 推荐 |
| `routes_admin.go` | 管理端路由注册 | 有管理端接口时使用 |
| `producer.go` | MQ 发送能力 | 有消息生产时使用 |
| `page_result.go` | 模块内分页结构 | 旧模块兼容，优先迁到 `platform/pagination` |

### 11.2 `service.go` 放什么

`service.go` 只放主流程接口，例如：

- Create
- Delete
- Update
- GetByID / Detail
- Mine / ListMine
- 主业务列表接口

### 11.3 `service_<业务>.go` 放什么

扩展能力或子业务拆到单独文件，例如：

- `service_search.go`
- `service_social.go`
- `service_admin.go`
- `service_follow.go`
- `service_identity.go`
- `service_notify.go`
- `service_wx.go`

当前 `topic`、`user`、`comment`、`chat` 等模块都在沿这个方向拆分。

### 11.4 `repository.go` 放什么

`repository.go` 放基座能力：

- `Repository`
- `NewRepository`
- 连接获取 helper
- 模块公用常量
- 子仓储共用内部类型

### 11.5 `repository_<业务>.go` 放什么

`repository_<业务>.go` 放具体数据操作：

- CRUD
- 搜索
- 点赞/收藏/关注等关系操作
- 聚合与统计更新
- 审核与状态流转

### 11.6 Handler 拆分规则

Handler 和 Service 拆分保持对称：

- 主流程 handler 放 `handler.go`
- 子域 handler 放 `handler_<业务>.go`
- 管理端 handler 放 `admin.go`
- 绑定结构体放 `handler_bindings.go`

### 11.7 `handler_helpers.go` 的边界

现有代码里很多模块仍然保留了：

- `bindJSON`
- `bindQuery`
- `bindURI`
- `pageSize`

这说明 `handler_helpers.go` 目前仍在使用，但也说明已经出现跨模块重复。

因此边界应明确为：

- 模块私有 helper 可以放在 `handler_helpers.go`
- 真正跨模块复用的 helper 应提取到 `internal/platform/`

`docs/module-layout.md` 提到的 `internal/platform/ginutil/` 目前仓库里还没有真正落地；它更适合作为后续统一收敛位置，而不是当前必须已经存在的目录。

## 12. 路由规范

### 12.1 路由注册函数命名

当前代码已经形成以下命名：

- `RegisterPublicRoutes`
- `RegisterProtectedRoutes`
- `RegisterAdminPublicRoutes`
- `RegisterAdminRoutes`

### 12.2 路由职责边界

路由注册函数只做路由映射，不写业务逻辑。

典型结构：

```go
func RegisterProtectedRoutes(api *gin.RouterGroup, handler *Handler) {
    registerTopicRoutes(api, handler)
    registerTopicLikeRoutes(api, handler)
}
```

模块内继续按资源或子域拆成私有注册函数，例如：

- `registerProfileRoutes`
- `registerFollowRoutes`
- `registerTopicRoutes`
- `registerTopicLikeRoutes`

### 12.3 中间件挂载位置

模块内只声明路由；认证、日志、权限校验在 app 层挂到 group 上。

这条边界在 `internal/app/ecampus/routes.go` 和 `internal/app/ecampuscrm/routes.go` 里已经很清楚。

## 13. 跨模块公共能力归位

当前公共能力已经比较明确：

| 能力 | 放置位置 |
|------|---------|
| 统一响应 | `internal/platform/responses/` |
| 业务错误 | `internal/platform/bizerr/` |
| 分页 | `internal/platform/pagination/` |
| 用户端 JWT | `internal/platform/jwtutil/` |
| 管理端 JWT | `internal/platform/adminjwt/` |
| 配置 | `internal/platform/config/` |
| 加密 | `internal/platform/encrypt/` |
| Redis key 规则 | `internal/platform/rediskey/` |
| 雪花 ID | `internal/platform/snowflake/` |
| 外部服务集成 | `internal/integration/<service>/` |

如果某段代码已经被多个模块复制，就不应该再继续堆在某个业务模块里，而应该上提到 `internal/platform/` 或 `internal/integration/`。

## 14. 测试风格

从现有测试可以看出几个稳定习惯：

- 测试文件和被测包放在一起，使用 `*_test.go`
- 优先写窄而明确的行为测试，不追求大而全的集成测试
- 路由、响应、中间件相关测试会显式设置 `gin.TestMode`
- 常见测试对象包括：
  - 响应封装
  - 路由暴露结果
  - 序列化兼容
  - helper 行为
  - service 内部辅助逻辑

新代码如果引入了兼容逻辑、字段映射、日期格式化、路由调整，应该优先补对应的小范围测试。

## 15. 新模块与重构清单

新模块开发或旧模块整理时，优先按下面顺序收敛：

1. 先明确主流程和扩展能力边界。
2. 用 `NewService` / `NewRepository` / `NewHandler` 组织依赖。
3. `handler.go` 只保留 HTTP 入口逻辑，业务都下沉到 service。
4. `repository.go` 只保留仓储基座，具体查询拆到子文件。
5. 模块稳定错误收敛到 `errors.go`。
6. 所有 JSON 接口优先统一走 `responses`。
7. 分页优先统一到 `internal/platform/pagination/`。
8. 重复的 handler helper 不再继续复制，逐步提到 `internal/platform/`。
9. 路由在模块内声明，在 app 层统一挂中间件和装配。

## 16. 默认约定

从现在开始：

- 新模块默认遵守本文档。
- 旧模块重构默认向本文档收敛。
- 如果某个旧模块和本文档不一致，以“现有稳定行为不回归”为前提逐步迁移，不要求一次性全量改完。
