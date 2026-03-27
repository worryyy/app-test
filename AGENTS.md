# Ecampus-Go Project Guide

## Project Overview

Ecampus-Go 是校园社区平台的 Go 重写版本，原版为 Java Spring Boot。
Go 版本采用 Gin + GORM + mongo-driver 技术栈，拆分为两个独立服务。

### 双服务架构

| 服务 | 入口 | 端口 | 职责 |
|------|------|------|------|
| ecampus | `cmd/ecampus/main.go` | 8080 | 用户端 API (`/api/**`) |
| ecampus-crm | `cmd/ecampus-crm/main.go` | 8081 | 管理后台 API (`/admin/**`) |

两个服务共享 `internal/` 下的所有代码，通过各自的 `routes.go` 注册不同的路由和中间件。

### Java 基线项目

路径：`/Users/guozhanfan/IdeaProjects/Ecampus`

Go 重写以 Java 现网版本为功能基线。API 路径、请求响应字段名、数据库 schema 必须与 Java 版本保持一致。但内部实现必须用 Go 惯用方式，不复制 Java 设计模式。

## Tech Stack

- **Web**: Gin
- **ORM**: GORM v2 (MySQL)
- **NoSQL**: mongo-go-driver v1 (MongoDB)
- **Cache**: go-redis/v9
- **MQ**: amqp091-go (RabbitMQ)
- **WebSocket**: gorilla/websocket
- **Auth**: golang-jwt/v5
- **Config**: spf13/viper
- **Log**: uber-go/zap
- **Cron**: robfig/cron/v3
- **File Storage**: tencentyun/cos-go-sdk-v5
- **Segmentation**: go-ego/gse
- **ID**: bwmarrin/snowflake
- **Metrics**: prometheus/client_golang
- **Validation**: go-playground/validator/v10
- **Encryption**: crypto/aes, crypto/des (stdlib)

## Project Structure

```
cmd/
  ecampus/          main.go + routes.go  (user-facing service)
  ecampus-crm/      main.go + routes.go  (admin service)
internal/
  user/             model.go, service*.go, handler*.go, admin.go
  topic/            model.go, service*.go, handler.go, admin.go
  comment/          model.go, service.go, handler.go, admin.go
  theme/            model.go, service.go, handler.go, admin.go
  chat/             model.go, service.go, handler.go, ws.go, realtime.go
  file/             model.go, service.go, handler.go, admin.go
  level/            model.go, service.go, handler.go
  school/           model.go, service.go, jw.go, handler.go, admin.go
  event/            model.go, service.go, handler.go, admin.go
  other/            model.go, service*.go, handler*.go, *_admin.go
  monitor/          model.go, service.go, admin.go
  mq/               config.go, producer.go, base.go, consumer*.go, topology.go
  cron/             scheduler.go, suggest.go, event_flush.go, exp_flush.go, metrics.go
  middleware/       jwt.go, admin.go, blacklist.go, cors.go, log.go
  pkg/
    config/         config.go, infra.go
    result/         result.go, page.go
    jwtutil/        helper.go
    rediskey/       keys.go
    snowflake/      id.go
    encrypt/        aes.go, des.go
    wxutil/         client.go
    cosutil/        client.go
configs/
  ecampus/          application.yml, application-dev.yml
  ecampus-crm/      application.yml, application-dev.yml
deployments/
  ecampus/          Dockerfile
  ecampus-crm/      Dockerfile
  docker-compose.yml
```

## Architecture Rules

### Package Convention

每个业务领域一个 package，package 内按文件职责拆分：

| 文件 | 职责 | 被谁使用 |
|------|------|---------|
| `model.go` | 数据结构（MySQL/MongoDB struct） | 两个服务共享 |
| `model_req.go` | 请求/响应结构体 | 两个服务共享 |
| `service.go` | 核心业务逻辑 | 两个服务共享 |
| `service_*.go` | 拆分的业务逻辑（如 service_follow.go） | 两个服务共享 |
| `handler.go` | 用户端 HTTP handler | 仅 ecampus |
| `handler_*.go` | 拆分的 handler | 仅 ecampus |
| `admin.go` | 管理端 HTTP handler | 仅 ecampus-crm |

### Two-Layer Architecture

只有两层：Handler → Service。

- **Handler**: 绑定请求参数 → 调用 Service → 返回响应。三步，不含业务逻辑。
- **Service**: 直接操作 GORM/mongo-driver/go-redis。不经过 Repository 层。

### One Struct Per Entity

一个数据实体只有一个 struct，同时携带 `gorm:`、`bson:`、`json:` tag。
不创建 VO/DTO/PO 分层。请求/响应如果与实体差异大，才在 `model_req.go` 中单独定义。

### Constructor Injection

不使用 DI 框架。Service struct 持有依赖，通过 `NewXxxService(...)` 构造函数注入：

```go
type Service struct {
    db     *gorm.DB
    mongo  *mongo.Database
    redis  *redis.Client
    cfg    *config.Config
    logger *zap.Logger
}
```

## Coding Rules

1. Handler 使用 `*gin.Context`，其他函数第一个参数是 `context.Context`
2. 始终检查 error，不用 `_` 忽略 error 返回值
3. 错误包装用 `fmt.Errorf("...: %w", err)`
4. 函数最大 80 行，文件最大 300 行。超过则拆分
5. 使用 early return 减少嵌套
6. nil slice 在 JSON 返回前转为空 slice
7. JSON 字段名必须与 Java 版本完全一致（同名、同结构）
8. 用 `zap` 记录日志，不记录密码、token、PII
9. 不使用 `init()` 函数，不使用全局可变状态，不使用 `panic`（启动阶段除外）
10. 软删除：`deleted_at` 字段（`NOW()` 表示已删除，`NULL` 表示未删除）
11. 编写和修改代码需要具备长期主义，要思考到未来这个代码会如何使用会如何变更，写出来的代码要具备可读性和高可维护性

## API Compatibility

### Response Format

所有 API 统一响应格式：
```json
{"success": bool, "code": int, "msg": "string", "data": any}
```

### Null Handling

JSON 响应中不出现 `null`：
- null string → `""`
- null number → `0`
- null bool → `false`
- null array → `[]`

### Auth

- `/api/**` 需要 JWT 认证
- `/admin/**` 需要 JWT + 管理员权限（双重校验：JWT power 位运算 + admin 表查询）
- 其他路径公开

## Build & Run

```bash
# 编译两个服务
go build ./cmd/ecampus
go build ./cmd/ecampus-crm

# 静态检查
go vet ./...

# 运行（需先启动 MySQL/Redis/MongoDB/RabbitMQ）
./ecampus
./ecampus-crm
```

## Commit Checklist

- 每次提交前必须先执行 `go mod tidy`
- 如果 `go mod tidy` 改动了 `go.mod` 或 `go.sum`，提交前必须检查这些变更是否与本次任务相关

## Design Documents

详细设计文档在 Java 项目中：`/Users/guozhanfan/IdeaProjects/Ecampus/docs/go-rewrite/`

| 文档 | 内容 |
|------|------|
| 01-go-conventions.md | 编码规范、项目结构 |
| 02-architecture.md | 技术栈、启动代码、部署 |
| 03-data-models.md | 所有数据结构定义 |
| 04-api-endpoints.md | 全部 ~149 个 API 端点 |
| 05-key-flows.md | 17 个核心业务流程伪代码 |
| AUDIT-AGENT.md | 审计任务专用 Agent 指令 |
| AUDIT-TASKS.md | 分 Batch 的审计任务提示词 |

## 补充
- 本次重构切换后，不存在 Java 与 Go 双版本同时部署。
- 旧 Redis 缓存允许在切换前清空。
- 如果有任何不清晰的地方，请向我提问到清晰无误为止。
- commit填写的信息要用中文
