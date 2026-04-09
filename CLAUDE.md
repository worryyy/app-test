# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 协作原则

1. **目标不清就停下来讨论** — 不猜测需求，有歧义立即提问
2. **方法不是最优的，直接说更好的办法** — 不迁就次优方案
3. **遇到问题追根因，不打补丁** — 找到真正的原因再修
4. **具备长期主义** — 当前修改要考虑未来的可读性、可维护性和扩展性
5. **输出只说重点** — 不废话，不重复已知信息

## 项目概述

校园社区平台的 Go 重写版本（原 Java Spring Boot）。API 路径、字段名、数据库 schema 必须与 Java 版本保持一致，内部实现用 Go 惯用方式。

## 双服务架构

| 服务 | 入口 | 端口 | 配置目录 | 职责 |
|------|------|------|---------|------|
| ecampus | `cmd/ecampus/main.go` | 8080 | `configs/ecampus/` | 用户端 `/api/**` + WebSocket + Cron + MQ Consumer |
| ecampus-crm | `cmd/ecampus-crm/main.go` | 8081 | `configs/ecampus-crm/` | 管理后台 `/admin/**` |

两服务共享 `internal/`，通过各自 `routes.go` 注册不同路由。ecampus 独有 Cron 调度和 RabbitMQ 消费者。

## 常用命令

```bash
# 编译
go build ./cmd/ecampus
go build ./cmd/ecampus-crm

# 静态检查
go vet ./...

# 运行测试
go test ./...                           # 全部
go test ./internal/chat/...             # 单个包
go test -run TestFuncName ./internal/chat/...  # 单个测试

# 提交前必须
go mod tidy

# 部署（push main 自动触发 GitHub Actions → Docker → SSH 部署）
```

依赖基础设施：MySQL、MongoDB、Redis、RabbitMQ（本地开发需先启动）。
配置通过 `APP_PROFILE` 环境变量选择 profile（默认 `dev`），加载 `application.yml` + `application-{profile}.yml`。

## 架构规则

**两层架构：Handler → Service。** Service 通过内部 Repository 组织数据访问代码（GORM/mongo-driver/go-redis），但 Repository 不抽象为接口，不用于测试替换——本质上仍然是 Service 直接操作数据库，Repository 只是物理文件拆分。

**Package 结构**（每个业务领域一个 package，详见 `docs/module-layout.md`）：
- `model.go` / `model_req.go` / `model_resp.go` — 数据结构，两服务共享
- `service.go` / `service_<业务>.go` — 业务逻辑，按子域拆分，两服务共享
- `repository.go` / `repository_<业务>.go` — 数据访问，按子域拆分
- `handler.go` / `handler_<业务>.go` — 用户端 HTTP handler
- `admin.go` — 管理端 HTTP handler
- `routes.go` / `routes_admin.go` — 路由自注册
- `errors.go` — 模块级错误定义

**构造函数注入**：`NewXxxService(db, mongoDB, rds, cfg, logger)` 模式，不用 DI 框架。

**一个实体一个 struct**：同时携带 `gorm:`、`bson:`、`json:` tag，不建 VO/DTO/PO 分层。

**目录划分**：
- `internal/platform/` — 基础能力（bizerr, config, jwtutil, responses, encrypt, snowflake, pagination）
- `internal/integration/` — 外部服务集成（cosutil, wxutil）
- `internal/app/bootstrap/` — 基础设施初始化与 HTTP server
- `internal/app/ecampus/` / `internal/app/ecampuscrm/` — 服务组装与路由注册

## 编码规范

- Handler 用 `*gin.Context`，其他函数第一个参数用 `context.Context`
- 始终检查 error，不用 `_` 忽略 error 返回值
- 错误包装：`fmt.Errorf("...: %w", err)`
- 函数最大 80 行，文件最大 300 行，超过拆分
- Early return 减少嵌套
- nil slice 在 JSON 返回前转空 slice（`result.normalizeData` 已自动处理）
- JSON 字段名与 Java 版本完全一致
- 日志用 `zap`，不记录密码/token/PII
- 不用 `init()`，不用全局可变状态，不用 `panic`（启动阶段除外）
- 软删除：`deleted_at`（`NOW()` = 已删除，`NULL` = 未删除）

## API 规范

统一响应格式：`{"success": bool, "code": int, "msg": "string", "data": any}`

使用 `responses.Success.RespData(c, data)` / `responses.Fail(c, err)` / `responses.Success.Resp(c)` 返回。
业务错误用 `bizerr.Biz(msg)` / `bizerr.Param(msg)` / `bizerr.NotFound(msg)` 构造，在 `responses.Fail` 中自动解析。

**Null 处理**：JSON 响应不出现 `null`（string→`""`，number→`0`，bool→`false`，array→`[]`）。

**认证**：
- `/api/**` — JWT 认证
- `/admin/**` — JWT + 管理员权限（位运算校验 + admin 表查询）
- `/health`、`/metrics`、`/file/:md5` — 公开

## MQ 架构

RabbitMQ direct exchange，死信队列模式。Producer 在 `internal/mq/producer.go`，消费者按业务拆分为 `consumer_*.go`。
队列拓扑定义在 `internal/mq/topology.go`。

## Cron

定时任务在 `internal/cron/scheduler.go`，仅 ecampus 服务启动。包含推荐刷新、经验刷新、事件刷新、监控指标等。

## 提交规范

- commit 信息用**中文**，简洁说明改了什么
- 每次提交前执行 `go mod tidy`，检查 go.mod/go.sum 变更是否与本次任务相关
- 提交前检查代码：`go vet ./...`，确认无编译错误和明显问题
