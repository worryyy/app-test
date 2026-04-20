@AGENTS.md

# CLAUDE.md — Ecampus-go 项目规范

> 通用 agent 行为准则见 [AGENTS.md](./AGENTS.md)。本文件是 Claude Code 专属的项目细节契约。

---

## 1. 代码库与本地运行

### 双服务

| 服务 | 入口 | 端口（application.yml 默认） | 路由前缀 | 职责 |
|------|------|------|---------|------|
| `ecampus` | `cmd/ecampus/main.go` | 8080 | `/api/**` | 用户端 API、聊天、文件、MQ consumer |
| `ecampus-crm` | `cmd/ecampus-crm/main.go` | 8081 | `/admin/**` | 管理端 API |

端口由 `configs/<service>/application*.yml` 的 `server.port` 决定；bootstrap 层用 `fmt.Sprintf(":%d", port)` 绑定（见 `internal/app/bootstrap/http.go:57`）。

### 目录职责

- `internal/app/bootstrap/` — 基础设施初始化、Gin 引擎、HTTP Server
- `internal/app/{ecampus,ecampuscrm}/` — 应用装配 + 路由注册
- `internal/<module>/` — 业务模块（`user` / `topic` / `comment` / `chat` / `school` / `theme` …）
- `internal/platform/` — 通用能力（`responses` / `bizerr` / `pagination` / `jwtutil` / `adminjwt` …）
- `internal/integration/` — 外部集成（微信、COS）
- `internal/middleware/` — 通用 Gin 中间件
- `internal/mq/` — MQ 生产者 / 消费者 / 拓扑

### 本地启动 / 构建 / 测试

```bash
go run ./cmd/ecampus          # 本地默认读 application-local.yml（profile 由 config loader 决定）
go run ./cmd/ecampus-crm
go build ./...                # 编译自查
go test ./internal/<mod>/...  # 改完必跑相关模块；PR 前跑 go test ./...
gofmt -w . && golangci-lint run ./...
```

CI/CD 走 `.github/workflows/deploy.yml`（Docker + SSH），本地无需执行部署命令。

---

## 2. 分层契约

**层次**：`Handler → Service → Repository`。Repository **不是抽象层**，不定义 interface 做测试替身（见 AGENTS.md 第 0.2 节）。

### 2.1 Handler 只做 4 件事

1. 绑定 URI / Query / JSON 参数（用模块内 `bindJSON` / `bindQuery` / `bindURI`）
2. 从 context 读认证信息（`middleware.GetUserID(c)` / `currentClaims(c)` / `currentUserID(c)`）
3. 调用 service，**传 `c.Request.Context()`**（不要把 `*gin.Context` 带进 service）
4. 用 `responses.*` 统一输出

**IMPORTANT：Handler 禁止**直接访问数据库、写业务规则、跨模块编排、自己解析 JWT。

> Why：Handler 一旦混入业务逻辑，单元测试必须起 Gin 引擎；thin handler 让 service 用普通 `context.Context` 测。

### 2.2 Service 负责业务

职责：参数语义校验 / 领域规则校验 / 调用 repository / 调用 MQ producer / 敏感词过滤器 / 外部接口 / 组装返回模型 / 统一错误包装（见第 3 节）。

依赖从构造器注入（**不用 DI 框架**）；`logger == nil` 回退 `zap.NewNop()`。

### 2.3 Repository 按子域拆文件

- `repository.go` 只放基座：`Repository` struct、`NewRepository`、`gormDB(ctx)`、`mongoCollection(name)`、模块常量、子仓储共用内部类型。
- 具体 CRUD / 搜索 / 关系操作 / 聚合统计 → `repository_<子域>.go`（如 `repository_topic.go`、`repository_social.go`）。

### 2.4 装配职责

**IMPORTANT：业务模块内部 YOU MUST NOT 创建全局单例，YOU MUST NOT 偷偷初始化外部依赖。**

> Why：全局单例让双服务（ecampus / ecampus-crm）共享模块时无法给不同实例注入不同配置；测试里也没法换掉外部依赖。

所有跨模块依赖组装在 `internal/app/ecampus/app.go` 和 `internal/app/ecampuscrm/app.go`：

1. 先 `bootstrap.LoadInfrastructure`
2. 创建 service / handler / producer，注入 DB / Mongo / Redis / config / logger
3. 注册路由

---

## 3. 响应与错误规范

### 3.1 HTTP 状态码收敛（最易踩雷）

**YOU MUST** 理解：本项目真实 HTTP Status 只有 `200` 或 `400`（由 `responses.normalizeCode` 决定），业务语义走响应 body 的 `httpstatus` 字段。

> 陷阱：结构体字段名是 `HTTPStatus`，但它**不是** HTTP 协议层 Status Code，而是"业务侧状态码"（例如 403 表示"无权限"这一业务语义，真实响应依然是 HTTP 200/400）。客户端判断成败读 `success` / `httpstatus`，不能读 HTTP Status。

```go
// ✅ 正确
responses.Fail(c, bizerr.Forbidden("无权访问"))

// ❌ 禁止：会破坏客户端契约
c.JSON(http.StatusForbidden, gin.H{...})
c.AbortWithStatus(http.StatusNotFound)
```

### 3.2 统一响应入口

所有 JSON 接口走 `internal/platform/responses`：

- `responses.Success.Resp(c)` — 无 data 成功
- `responses.Success.RespData(c, data)` — 有 data 成功
- `responses.Fail(c, err)` — 失败（内部 `c.Error(err)` 便于日志采集）
- `responses.FailMessage(c, err, message)` — 失败 + 覆盖消息

只有文件 / 二进制 / 纯字符串才用 Gin 原生输出（`c.String` / `c.File` …）。

### 3.3 业务错误

全走 `internal/platform/bizerr/`。每个构造函数都有对应的 `*Wrap(message, cause)` 版本用于包裹底层 err：

| 构造函数 | Wrap 版本 | 用途 |
|---------|----------|------|
| `Param` | `ParamWrap` | 参数错误 |
| `Biz` | `BizWrap` | 业务规则失败 |
| `NotFound` | `NotFoundWrap` | 资源不存在 |
| `Unauthorized` | `UnauthorizedWrap` | 未登录 |
| `Forbidden` | `ForbiddenWrap` | 无权限 |
| `Internal` | `InternalWrap` | 内部错误（通常包外部依赖的 err） |

**模块级稳定错误放 `errors.go` 包级变量**（如 `var ErrTopicNotFound = bizerr.NotFound("帖子不存在")`）；动态上下文错误（含 ID 拼接）在 service 内即时构造。

两层错误处理：bootstrap / platform / integration 层用 `fmt.Errorf("...: %w", err)`；业务 service 层统一 `bizerr.*`。

---

## 4. 本地易踩雷约定

| 约定 | 要点 |
|------|------|
| **HTTP Status 收敛** | 真实 200/400，业务码在 body.httpstatus（见 3.1） |
| **空列表返回 `[]T{}` 不是 `nil`** | 防止客户端看到 `null` 而做空判失败 |
| **JSON 字段名 camelCase** | `accountType` / `rootUserId` / `createdTime`，保持旧客户端兼容 |
| **时间字段常自定义 `MarshalJSON`** | 评论等模块把 `time.Time` 格式化为字符串——**序列化兼容 > 零定制** |
| **Handler 不解析 JWT** | 统一走 middleware 写入 context，用 `middleware.GetUserID(c)` / `currentClaims(c)` 读 |
| **context 传递** | Handler 用 `*gin.Context`；Service / Repository 用 `context.Context`；Handler 调 Service 时传 `c.Request.Context()` |
| **分页统一** | 新模块用 `internal/platform/pagination.PageResult[T]`；旧模块的 `page_result.go` 不新增（见模块布局第 10 节） |
| **db_columns.go** | 部分模块（如 `topic`）把 GORM 列名抽成常量；**仅在跨多个 repository 文件复用列名时新建**，单文件内别上提 |

---

## 5. 模块布局

标准参考：`internal/topic`。完整文件清单、拆分规则、**已知债务清单** → [`docs/module-layout.md`](./docs/module-layout.md)。

常用文件：`model.go` / `model_req.go` / `errors.go` / `service.go` / `service_<子域>.go` / `repository.go` / `repository_<子域>.go` / `handler.go` / `handler_bindings.go` / `admin.go` / `routes.go` / `routes_admin.go`。

**`service.go` 只放主流程**（Create / Delete / Update / Detail / Mine / 主列表）；扩展能力拆到 `service_<子域>.go`。Handler 与 Service 拆分**保持对称**。

---

## 6. 路由与中间件

### 路由注册函数命名

模块按需暴露以下名字，由 app 层调用：

- 用户端：`RegisterPublicRoutes`（不鉴权）/ `RegisterProtectedRoutes`（鉴权后）
- 管理端：`RegisterAdminPublicRoutes`（不鉴权）/ `RegisterAdminRoutes`（鉴权后）
- 特殊资源可自定义名字（例：`chat.RegisterInfraRoutes` 挂 WebSocket 基础设施；`file.RegisterProtectedRoutes` 签名带专用 group 参数）

模块内按资源拆私有注册函数（`registerTopicRoutes`、`registerFollowRoutes` 等）。路由注册函数**只映射路由，不含业务逻辑**。

### 中间件挂载位置

**IMPORTANT：模块只声明路由，不挂中间件。** 认证 / 日志 / 权限在 app 层统一挂到 group 上：

- 用户端：`JWTAuth` / `RequestLog` / `CertifiedUserCheck`
- 管理端：`AdminJWTAuth` / `RequestLog` / `AdminPermissionCheck`

> Why：同一个模块可能在用户端 / 管理端 / 测试环境下需要不同中间件组合，模块内写死就失去了这种复用弹性。

---

## 7. 跨模块公共能力

| 能力 | 位置 |
|------|------|
| 响应 / 错误 / 分页 | `internal/platform/{responses,bizerr,pagination}/` |
| JWT（用户 / 管理） | `internal/platform/{jwtutil,adminjwt}/` |
| 配置 / 加密 / Redis key / 雪花 ID | `internal/platform/{config,encrypt,rediskey,snowflake}/` |
| 外部服务集成（微信 / COS） | `internal/integration/<service>/` |

**判断规则**：被 ≥ 2 个模块复制 → 上提到 `platform/`；单模块使用 → 留在模块内。上提时走 AGENTS.md 第 3 节迁移原则，不与业务改动混合 PR。

---

## 8. 新模块 30 秒起手式

1. 明确主流程接口范围（Create / Delete / Update / Detail / Mine / 主列表）→ 进 `service.go`
2. 扩展能力拆 `service_<子域>.go`，repository / handler 同步对称拆
3. 模块级稳定错误进 `errors.go`，用 `bizerr.*` 构造
4. 所有 JSON 接口走 `responses.*`，分页走 `platform/pagination.PageResult[T]`
5. 路由在模块内声明 `Register*Routes`；中间件由 app 层挂
6. 依赖在 `internal/app/<service>/app.go` 注入，**模块内不创建全局单例**
7. 涉及兼容字段 / 日期格式 / 错误分支时补窄行为测试；`go build ./... && go test ./internal/<module>/` 通过
