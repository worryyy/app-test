# Codex/GPT Implementation Prompt for Ecampus Go Rewrite

> 将此文件内容作为 system prompt 或 AGENTS.md 喂给 Codex/GPT。
> 下方 [PHASE N] 模板用于分阶段提交任务。

---

## System Prompt（放入 system message 或 custom instructions）

```
You are a senior Go engineer implementing the Ecampus campus community platform.
This is a REWRITE from Java Spring Boot to Go (Gin + GORM + mongo-driver).

## Architecture
- Monorepo with TWO services: `cmd/ecampus/` (user-facing :8080) and `cmd/ecampus-crm/` (admin :8081)
- Feature-based packaging: each domain is one package under `internal/` (user/, topic/, comment/, etc.)
- Each package has: model.go (shared), service.go (shared), handler.go (ecampus), admin.go (ecampus-crm)
- NO repo layer. Service directly calls GORM/mongo-driver/go-redis.
- NO VO/DTO/PO. One struct with gorm:/bson:/json: tags.
- NO DI framework. Constructor injection, two layers: handler → service.

## Coding Rules (STRICT)
1. Every function's first param is `context.Context` (except handlers which use `*gin.Context`)
2. Always check errors. Never use `_` to ignore error returns.
3. Use `fmt.Errorf("...: %w", err)` for error wrapping.
4. Handler pattern: bind request → call service → return result. Three lines, no business logic in handler.
5. Max 80 lines per function, max 300 lines per file. Split when exceeded.
6. Use early return to reduce nesting depth.
7. nil slice must be converted to empty slice before JSON response (use `result.EnsureSlice()`).
8. All JSON response fields must match Java version EXACTLY (same key names, same structure).
9. Use `zap` for logging. Never log passwords, tokens, or PII.
10. No `init()` functions. No global mutable state. No `panic` except startup.

## Tech Stack
- Gin, GORM v2 (MySQL), mongo-go-driver v1 (MongoDB), go-redis/v9, amqp091-go (RabbitMQ)
- gorilla/websocket, golang-jwt/v5, spf13/viper, uber-go/zap, robfig/cron/v3
- go-playground/validator/v10, tencentyun/cos-go-sdk-v5, go-ego/gse, bwmarrin/snowflake
- gomail/v2, prometheus/client_golang, crypto/aes, crypto/des (stdlib)

## Design Documents
All implementation details are in these 5 documents (READ THEM COMPLETELY before coding):
- 01-go-conventions.md — coding rules, project structure, file split rules
- 02-architecture.md — tech stack, startup code, Docker, config
- 03-data-models.md — ALL structs (MySQL + MongoDB + Redis keys + MQ queues)
- 04-api-endpoints.md — ALL ~149 API endpoints with request/response formats
- 05-key-flows.md — pseudocode for 17 complex business flows

## Critical Compatibility Requirements
- Database field names MUST match Java version exactly (camelCase in MySQL, as-is in MongoDB)
- API paths MUST be identical (/api/**, /admin/**, /file/**)
- JSON response format: {"success": bool, "code": int, "msg": string, "data": any}
- Null handling: empty string for null string, 0 for null number, [] for null array, false for null bool
```

---

## 分阶段实施 Prompt 模板

### Phase 1: 项目骨架 + 基础设施

```
Read all 5 design documents in docs/go-rewrite/.

Create the project skeleton:
1. go.mod with all dependencies from 02-architecture.md Section 9
2. internal/pkg/config/config.go — Config struct + Load() from viper
3. internal/pkg/result/result.go — Result, Success(), Fail(), HandleError(), response codes
4. internal/pkg/result/page.go — PageResult[T], CusPage[T], EnsureSlice()
5. internal/pkg/rediskey/keys.go — ALL Redis key definitions from 03-data-models.md Section 4
6. internal/pkg/jwtutil/helper.go — JWT Claims, GenerateTokenPair, Parse, ParseAndVerify
7. internal/pkg/snowflake/id.go — Snowflake ID generator init
8. internal/middleware/jwt.go — JWTAuth middleware
9. internal/middleware/admin.go — AdminCheck middleware (dual verification: JWT + admin table)
10. internal/middleware/blacklist.go — BlackListCheck (rootUserId + 1s timeout fail-open)
11. internal/middleware/cors.go — CORS middleware
12. internal/middleware/log.go — Request logging middleware

After creating each file, run `go build ./...` to verify compilation.
```

### Phase 2: 数据模型

```
Read 03-data-models.md completely.

Create ALL model.go files with exact struct definitions:
1. internal/user/model.go — User, Admin, Follow, UserBlacklist, OfficialCertification, AdminLoginReq, UserProfile
2. internal/topic/model.go — Topic, TopicSearch, TopicLike, TopicCollection, CreateTopicReq, SuggestListVO
3. internal/comment/model.go — Comment, CommentUser, CommentLike
4. internal/theme/model.go — Theme
5. internal/file/model.go — File, CompressFile
6. internal/chat/model.go — Conversation, ConversationMember, Message, Notification
7. internal/level/model.go — ExpDetail, SignDetailVO
8. internal/school/model.go — UserCourse, Term, CurTerm, Course
9. internal/event/model.go — Event
10. internal/other/model.go — Ad, Notice, SensitiveWord, VoteInfo, VoteOption, VoteAns, Task, ReportComment, MerchantTheme, FrontendSupport
11. internal/monitor/model.go — CacheStats (if needed)
12. internal/mq/config.go — Exchange, Queue, RoutingKey constants
13. internal/mq/model.go — MQMessage, MQLog, all message structs

CRITICAL: All struct field names, gorm column names, bson field names, and json keys
must EXACTLY match the Java version. Do not rename anything.

Run `go build ./...` after each file.
```

### Phase 3: Service 层（核心业务逻辑）

```
Read 05-key-flows.md for complex business logic.
Read 04-api-endpoints.md for all operations needed.

Implement service.go for each domain. Order by dependency:
1. internal/user/service.go — NewService, WechatLogin, RefreshToken, AdminLogin (with secondary password + lockout + legacy migration), GetByID, GetByOpenID, Edit, Follow/Unfollow, GetStats, identity management
2. internal/topic/service.go — CRUD, search (hot sort + keyword), suggest list, like/unlike, collect/uncollect
3. internal/comment/service.go — CRUD, like/unlike
4. internal/theme/service.go — CRUD, suggest config
5. internal/file/service.go — Upload (COS + CI compression fallback), Delete, List
6. internal/chat/service.go — conversations, messages, notifications, HandleMessage
7. internal/level/service.go — SignIn (Redis bitmap), GetSignDetail, GetExp
8. internal/school/service.go — terms, course color
9. internal/school/jw.go — JW system login (DES encryption), course fetching
10. internal/other/[domain].go — ad/notice/vote/sensitive/report/merchant/support services
11. internal/event/service.go — event buffering to Redis
12. internal/monitor/service.go — cache stats

Each service struct holds: *gorm.DB, *mongo.Database, *redis.Client, *config.Config, *zap.Logger
(plus domain-specific deps like MQ producers).

Split files at 300 lines: service_follow.go, service_search.go, etc.
Run `go build ./...` after each file.
```

### Phase 4: Handler 层（HTTP 入口）

```
Read 04-api-endpoints.md for all endpoints.

Implement handlers. Each handler follows the three-step pattern:
  1. Bind request (ShouldBindJSON / Query params)
  2. Call service method
  3. Return result.Success() or result.HandleError()

Order:
1. internal/user/handler.go — all /api/user/** handlers
2. internal/user/admin.go — all /admin/user/** handlers
3. internal/topic/handler.go — /api/topic/**, /api/like/topic/**, /api/collection/**
4. internal/topic/admin.go — /admin/topic/**
5. internal/comment/handler.go — /api/comment/**, /api/comment_like/**
6. internal/comment/admin.go — /admin/comment/**
7. internal/theme/handler.go + admin.go
8. internal/file/handler.go + admin.go
9. internal/chat/handler.go — REST endpoints for conversations/messages/notifications
10. internal/chat/ws.go — WebSocket handler (upgrade, auth, readPump, heartbeat)
11. internal/level/handler.go
12. internal/school/handler.go + admin.go
13. internal/other/*.go — all sub-domain handlers and admin handlers
14. internal/event/handler.go + admin.go
15. internal/monitor/admin.go

Run `go build ./...` after each file.
```

### Phase 5: MQ + Cron + 路由注册 + main.go

```
Implement infrastructure and wire everything:

1. internal/mq/producer.go — BaseProducer + Send(), confirm/return callbacks
2. internal/mq/base.go — HandleWithDedup (dedup + 3x retry)
3. internal/mq/consumer.go — all 12 queue consumers (topic check, comment add, search index, course fetch, notify, etc.)
4. internal/cron/suggest.go — recommend ranking generation (daily 02:01)
5. internal/cron/event_flush.go — event batch insert (every 15min)
6. internal/cron/metrics.go — DAU/WAU/MAU refresh (every 60s)
7. internal/cron/exp_flush.go — experience detail batch insert (every 5min)
8. internal/pkg/wxutil/client.go — WeChat API (jscode2session, stable_token, msg_sec_check, subscribe msg, wxa_code)
9. internal/pkg/cosutil/client.go — COS upload with CI compression fallback
10. internal/pkg/encrypt/aes.go + des.go — AES/DES encryption matching Java version
11. cmd/ecampus/main.go + routes.go — full startup, route registration, graceful shutdown
12. cmd/ecampus-crm/main.go + routes.go — admin service startup
13. configs/ecampus/application-dev.yml
14. configs/ecampus-crm/application-dev.yml
15. deployments/ecampus/Dockerfile + deployments/ecampus-crm/Dockerfile + docker-compose.yml

Run `go build ./cmd/ecampus && go build ./cmd/ecampus-crm` to verify both binaries compile.
Run `go vet ./...` and fix any issues.
```

### Phase 6: 验证

```
Final verification:
1. `go build ./cmd/ecampus && go build ./cmd/ecampus-crm` — both compile
2. `go vet ./...` — no issues
3. `golangci-lint run ./...` — no critical issues
4. Count all registered routes match 04-api-endpoints.md (~149 total)
5. Verify all model structs match 03-data-models.md
6. Check no file exceeds 300 lines: `find . -name '*.go' | xargs wc -l | sort -rn | head -20`
7. Check no function exceeds 80 lines
8. Verify all JSON response keys match Java version
```

---

## 使用技巧

### 上下文管理
- 每个 Phase 作为独立对话提交，避免上下文溢出
- Phase 1-2 放一个对话（骨架+模型，相互依赖）
- Phase 3 每 2-3 个 service 一个对话
- Phase 4 每 3-4 个 handler 一个对话
- Phase 5 一个完整对话
- 每个对话开头都粘贴 System Prompt

### 错误修复循环
```
Build failed. Here's the error:
[paste go build error]

Fix this error. After fixing, run `go build ./...` again to verify.
```

### 文件太长时
```
The file internal/user/service.go is now 350 lines, exceeding the 300-line limit.
Split it into service.go (core CRUD) and service_follow.go (follow/unfollow/stats).
Preserve all existing functionality.
```
