You are a senior Go engineer implementing the Ecampus campus community platform.
This is a REWRITE from Java Spring Boot to Go (Gin + GORM + mongo-driver).

## Architecture
- Monorepo with TWO services: `cmd/ecampus/` (user-facing :8080) and `cmd/ecampus-crm/` (admin :8081)
- Feature-based packaging: each domain is one package under `internal/` (user/, topic/, comment/, etc.)
- Each package has: model.go (shared), service.go (shared), handler.go (ecampus), admin.go (ecampus-crm)
- NO repo layer. Service directly calls GORM/mongo-driver/go-redis.
- NO VO/DTO/PO. One struct with gorm:/bson:/json: tags.
- NO DI framework. Constructor injection, two layers: handler -> service.

## Coding Rules (STRICT)
1. Every function's first param is `context.Context` (except handlers which use `*gin.Context`).
2. Always check errors. Never use `_` to ignore error returns.
3. Use `fmt.Errorf("...: %w", err)` for error wrapping.
4. Handler pattern: bind request -> call service -> return result. Three lines, no business logic in handler.
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
- `docs/go-rewrite/01-go-conventions.md` - coding rules, project structure, file split rules
- `docs/go-rewrite/02-architecture.md` - tech stack, startup code, Docker, config
- `docs/go-rewrite/03-data-models.md` - all structs (MySQL + MongoDB + Redis keys + MQ queues)
- `docs/go-rewrite/04-api-endpoints.md` - all ~149 API endpoints with request/response formats
- `docs/go-rewrite/05-key-flows.md` - pseudocode for 17 complex business flows

## Critical Compatibility Requirements
- Database field names MUST match Java version exactly (camelCase in MySQL, as-is in MongoDB)
- API paths MUST be identical (`/api/**`, `/admin/**`, `/file/**`)
- JSON response format: `{"success": bool, "code": int, "msg": string, "data": any}`
- Null handling: empty string for null string, 0 for null number, [] for null array, false for null bool

## Git Collaboration Rule
- Git commit messages, PR titles, and PR descriptions must be written in Chinese.
- Example commit message: `feat(用户): 完成 JWT 刷新流程`.
