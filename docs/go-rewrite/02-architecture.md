# 架构设计与技术选型

> 本项目采用 **monorepo 双服务** 架构，两个独立可部署的 Go 二进制共享同一套域包和数据库。

## 1. 双服务边界

| | ecampus（用户端） | ecampus-crm（管理后台） |
|--|------------------|----------------------|
| 路由前缀 | `/api/**`、`/file/**`、`/chat`(WS) | `/admin/**` |
| 端口 | 8080 | 8081 |
| 中间件链 | JWT → BlackList → Log | JWT → BlackList → Log → Admin |
| 定时任务 | 运行所有 cron job | 不运行 |
| MQ 消费者 | 运行所有 consumer | 不运行 |
| WebSocket | 提供 `/chat` | 不提供 |
| Health/Metrics | `/health`、`/metrics` | `/health` |

**共享资源**：同一个 MySQL、MongoDB、Redis、RabbitMQ 实例。两个服务各自连接，各自读写。

---

## 2. 技术选型

| 类别 | Java 原技术 | Go 选型 | 理由 |
|------|------------|---------|------|
| Web 框架 | Spring Boot + Spring MVC | **Gin** | Go 生态最成熟 HTTP 框架 |
| MySQL ORM | MyBatis-Plus 3.4.2 | **GORM v2** | 软删除、分页、钩子，对等 MyBatis-Plus |
| MongoDB | Spring Data MongoDB | **mongo-go-driver v1** | 官方驱动，完整聚合管道 |
| Redis | Spring Data Redis (Lettuce) | **go-redis/redis/v9** | 完整数据结构支持 |
| RabbitMQ | Spring AMQP | **rabbitmq/amqp091-go** | 官方 AMQP 0.9.1 客户端 |
| WebSocket | Spring WebSocket | **gorilla/websocket** | 成熟稳定 |
| JWT | Auth0 java-jwt | **golang-jwt/jwt/v5** | Go 标准 JWT 库 |
| 配置 | Spring profiles + YAML | **spf13/viper** | YAML + 环境变量 + profile |
| 日志 | Logback | **uber-go/zap** | 高性能结构化日志 |
| 参数校验 | Spring Validation | **go-playground/validator/v10** | Gin 默认集成 |
| 定时任务 | @Scheduled | **robfig/cron/v3** | Go 标准 cron 库 |
| HTTP 客户端 | Forest + OkHttp | **net/http**（标准库） | 标准库足够 |
| COS | cos_api (Java SDK) | **tencentyun/cos-go-sdk-v5** | 官方 Go SDK |
| 图片压缩 | Thumbnailator | COS 万象 CI 处理 | 压缩由云端完成 |
| 中文分词 | Jieba (huaban) | **go-ego/gse** | 纯 Go，无 CGO，同 Jieba 算法 |
| Snowflake ID | 自实现 | **bwmarrin/snowflake** | 轻量广泛 |
| 邮件 | Spring Mail + Thymeleaf | **gopkg.in/gomail.v2** + **html/template** | 简单邮件 + Go 原生模板 |
| Metrics | Micrometer + Prometheus | **prometheus/client_golang** | Prometheus 官方 Go 客户端 |
| 加密 | 自实现 AES/DES | **crypto/aes** + **crypto/des**（标准库） | 标准库密码学 |

---

## 3. 整体架构图

```
                     ┌─────────────────────────────────────────┐
                     │            Nginx / 网关                  │
                     │   /api/** → ecampus:8080                │
                     │   /admin/** → ecampus-crm:8081          │
                     └────────┬──────────────┬─────────────────┘
                              │              │
              ┌───────────────▼──┐    ┌──────▼───────────────┐
              │   ecampus:8080   │    │  ecampus-crm:8081    │
              │                  │    │                      │
              │ /api/**          │    │ /admin/**            │
              │ /file/**         │    │                      │
              │ /chat (WS)       │    │ Middleware:           │
              │ /health /metrics │    │ JWT→BlackList→Log    │
              │                  │    │ →AdminCheck           │
              │ Middleware:       │    │                      │
              │ JWT→BlackList    │    │ AdminHandlers only   │
              │ →Log             │    │                      │
              │                  │    │ 无 cron / 无 MQ 消费  │
              │ Handlers         │    │ 无 WebSocket          │
              │ + Cron           │    └──────┬───────────────┘
              │ + MQ Consumers   │           │
              │ + WebSocket      │           │
              └──────┬──────────┘           │
                     │                       │
        ┌────────────▼───────────────────────▼─────┐
        │         internal/ (共享域包)               │
        │                                          │
        │  user/  topic/  comment/  theme/  file/  │
        │  chat/  level/  school/   other/  event/ │
        │  monitor/                                │
        │                                          │
        │  每个包 = model + service + handler/admin │
        │  service 直接操作 DB，无 repo 层           │
        └───┬──────────┬──────────┬────────────────┘
            │          │          │
       ┌────▼───┐ ┌────▼───┐ ┌───▼────────┐
       │   DB   │ │   MQ   │ │  External  │
       │ MySQL  │ │RabbitMQ│ │  WX API    │
       │ Mongo  │ │Producer│ │  COS API   │
       │ Redis  │ │Consumer│ │  JW API    │
       └────────┘ └────────┘ └────────────┘
```

---

## 4. 模块映射

### 4.1 Java → Go 包映射

| Java 模块 | Go 包 (`internal/`) | ecampus 用 | CRM 用 |
|-----------|---------------------|:----------:|:------:|
| service-base | `middleware/`, `pkg/*` | Y | Y |
| user + user-entity | `user/` | handler.go | admin.go |
| theme + theme-entity | `theme/`, `topic/`, `comment/` | handler.go | admin.go |
| level | `level/` | handler.go | - |
| file | `file/` | handler.go | admin.go |
| school | `school/` | handler.go | admin.go |
| chat | `chat/` | handler.go + ws.go | - |
| mq | `mq/` | consumer + producer | - |
| monitor | `monitor/` | - | admin.go |
| aop | `middleware/auth_check.go` | Y | - |
| other | `other/` | handler 部分 | admin 部分 |
| front-event | `event/` | handler.go | admin.go |

### 4.2 Handler 归属明细

**ecampus 注册的 handler**（`handler.go` 文件）：

| 域包 | handler 文件 | 路由前缀 |
|------|-------------|---------|
| user | handler.go | `/api/user/**` |
| topic | handler.go | `/api/topic/**` |
| comment | handler.go | `/api/comment/**` |
| theme | handler.go | `/api/theme/**`（公开） |
| file | handler.go | `/file/**` |
| chat | handler.go + ws.go | `/api/conversation/**`, `/api/message/**`, `/api/notify/**`, `/chat`(WS) |
| level | handler.go | `/api/getUserSignDetail`, `/api/sign_in`, `/api/UserExp` |
| school | handler.go | `/api/term/**`, `/api/course_color` |
| other | ad.go, notice.go, vote.go, report.go, support.go | `/api/ad/**`, `/api/notice/**`, `/api/vote/**`, `/api/report_comment`, `/api/support/**` |
| event | handler.go | `/api/event` |

**ecampus-crm 注册的 handler**（`admin.go` 文件）：

| 域包 | admin 文件 | 路由前缀 |
|------|-----------|---------|
| user | admin.go | `/admin/user/**` |
| topic | admin.go | `/admin/topic/**` |
| comment | admin.go | `/admin/comment/**` |
| theme | admin.go | `/admin/theme/**` |
| file | admin.go | `/admin/file/**` |
| school | admin.go | `/admin/term/**` |
| other | ad_admin.go, notice_admin.go, sensitive_admin.go, report_admin.go, merchant_admin.go, support_admin.go | `/admin/ad/**`, `/admin/notice/**`, `/admin/sensitive/**`, `/admin/report_comment/**`, `/admin/merchant_theme/**`, `/admin/support/**` |
| event | admin.go | `/admin/event/**` |
| monitor | admin.go | `/admin/local_cache/**` |

---

## 5. 程序启动

### 5.1 ecampus（用户端）

```go
// cmd/ecampus/main.go
package main

import (
    "context"
    "fmt"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/your-org/ecampus/internal/user"
    "github.com/your-org/ecampus/internal/topic"
    "github.com/your-org/ecampus/internal/comment"
    "github.com/your-org/ecampus/internal/theme"
    "github.com/your-org/ecampus/internal/file"
    "github.com/your-org/ecampus/internal/chat"
    "github.com/your-org/ecampus/internal/level"
    "github.com/your-org/ecampus/internal/school"
    "github.com/your-org/ecampus/internal/other"
    "github.com/your-org/ecampus/internal/event"
    "github.com/your-org/ecampus/internal/mq"
    "github.com/your-org/ecampus/internal/middleware"
    "github.com/your-org/ecampus/internal/cron"
    "github.com/your-org/ecampus/internal/pkg/config"
    "github.com/your-org/ecampus/internal/pkg/jwtutil"
)

func main() {
    time.LoadLocation("Asia/Shanghai")

    // 1. 基础设施
    cfg := config.Load("configs/ecampus")
    logger := initLogger(cfg)
    db := initMySQL(cfg)
    mongoDB := initMongo(cfg)
    rds := initRedis(cfg)
    amqpConn := initRabbitMQ(cfg)
    jwtHelper := jwtutil.NewHelper(cfg.JWT)

    // 2. MQ
    producers := mq.NewProducers(amqpConn, rds)

    // 3. Services
    userSvc    := user.NewService(db, mongoDB, rds, cfg, logger)
    topicSvc   := topic.NewService(mongoDB, rds, producers, logger)
    commentSvc := comment.NewService(mongoDB, rds, producers, logger)
    themeSvc   := theme.NewService(mongoDB, rds, logger)
    fileSvc    := file.NewService(mongoDB, rds, cfg, logger)
    chatSvc    := chat.NewService(db, mongoDB, rds, logger)
    levelSvc   := level.NewService(db, rds, logger)
    schoolSvc  := school.NewService(db, mongoDB, rds, cfg, logger)
    otherSvc   := other.NewService(db, rds, logger)
    eventSvc   := event.NewService(db, rds, logger)

    // 4. User-facing handlers
    userH    := user.NewHandler(userSvc)
    topicH   := topic.NewHandler(topicSvc)
    commentH := comment.NewHandler(commentSvc)
    themeH   := theme.NewHandler(themeSvc)
    fileH    := file.NewHandler(fileSvc)
    chatH    := chat.NewHandler(chatSvc, jwtHelper, rds)
    levelH   := level.NewHandler(levelSvc)
    schoolH  := school.NewHandler(schoolSvc)
    otherH   := other.NewHandlers(otherSvc) // 返回包含多个子 handler 的 bundle
    eventH   := event.NewHandler(eventSvc)

    // 5. Router
    r := gin.New()
    r.Use(gin.Recovery(), middleware.CORS())
    registerUserRoutes(r, cfg, jwtHelper, rds, userSvc,
        userH, topicH, commentH, themeH, fileH,
        chatH, levelH, schoolH, otherH, eventH)

    // 6. Cron（ecampus 负责所有定时任务）
    cr := cron.New(topicSvc, eventSvc, rds, logger)
    cr.Start()

    // 7. MQ consumers（ecampus 负责所有消费者）
    consumers := mq.NewConsumers(amqpConn, rds, mongoDB, db, cfg, logger,
        topicSvc, commentSvc, schoolSvc)
    consumers.Start()

    // 8. 启动 + 优雅关闭
    srv := &http.Server{
        Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
        Handler: r,
    }
    go srv.ListenAndServe()
    logger.Info("ecampus started", zap.Int("port", cfg.Server.Port))

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    cr.Stop()
    consumers.Close()
    srv.Shutdown(ctx)
    logger.Info("ecampus exited")
}
```

### 5.2 ecampus-crm（管理后台）

```go
// cmd/ecampus-crm/main.go
package main

func main() {
    time.LoadLocation("Asia/Shanghai")

    // 1. 基础设施（与 ecampus 相同的连接方式）
    cfg := config.Load("configs/ecampus-crm")
    logger := initLogger(cfg)
    db := initMySQL(cfg)
    mongoDB := initMongo(cfg)
    rds := initRedis(cfg)
    jwtHelper := jwtutil.NewHelper(cfg.JWT)
    // 注意：CRM 不需要 RabbitMQ 连接（不消费、不生产）

    // 2. Services（复用同一套 service，但部分参数为 nil）
    userSvc    := user.NewService(db, mongoDB, rds, cfg, logger)
    topicSvc   := topic.NewService(mongoDB, rds, nil, logger)  // 无 MQ producer
    commentSvc := comment.NewService(mongoDB, rds, nil, logger)
    themeSvc   := theme.NewService(mongoDB, rds, logger)
    fileSvc    := file.NewService(mongoDB, rds, cfg, logger)
    schoolSvc  := school.NewService(db, mongoDB, rds, cfg, logger)
    otherSvc   := other.NewService(db, rds, logger)
    eventSvc   := event.NewService(db, rds, logger)
    monitorSvc := monitor.NewService(rds, logger)

    // 3. Admin handlers
    userAdmin    := user.NewAdminHandler(userSvc)
    topicAdmin   := topic.NewAdminHandler(topicSvc)
    commentAdmin := comment.NewAdminHandler(commentSvc)
    themeAdmin   := theme.NewAdminHandler(themeSvc)
    fileAdmin    := file.NewAdminHandler(fileSvc)
    schoolAdmin  := school.NewAdminHandler(schoolSvc)
    otherAdmins  := other.NewAdminHandlers(otherSvc) // 包含多个子 admin handler
    eventAdmin   := event.NewAdminHandler(eventSvc)
    monitorAdmin := monitor.NewAdminHandler(monitorSvc)

    // 4. Router
    r := gin.New()
    r.Use(gin.Recovery(), middleware.CORS())
    registerAdminRoutes(r, cfg, jwtHelper, rds,
        userAdmin, topicAdmin, commentAdmin, themeAdmin,
        fileAdmin, schoolAdmin, otherAdmins, eventAdmin, monitorAdmin)

    // 5. 启动（CRM 不需要 cron、不需要 MQ consumer）
    srv := &http.Server{
        Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
        Handler: r,
    }
    go srv.ListenAndServe()
    logger.Info("ecampus-crm started", zap.Int("port", cfg.Server.Port))

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    srv.Shutdown(ctx)
    logger.Info("ecampus-crm exited")
}
```

### 5.3 共享初始化函数

两个 `main.go` 都需要的初始化逻辑提取为共享函数：

```go
// internal/pkg/config/infra.go — 基础设施初始化
func InitMySQL(cfg *Config) *gorm.DB { ... }
func InitMongo(cfg *Config) *mongo.Database { ... }
func InitRedis(cfg *Config) *redis.Client { ... }
func InitRabbitMQ(cfg *Config) *amqp.Connection { ... }
func InitLogger(cfg *Config) *zap.Logger { ... }
```

两个 `main.go` import 这些函数，避免代码重复。

---

## 6. 路由注册

### 6.1 ecampus 路由注册

```go
// cmd/ecampus/routes.go
func registerUserRoutes(
    r *gin.Engine,
    cfg *config.Config,
    jwt *jwtutil.Helper,
    rds *redis.Client,
    userSvc *user.Service,
    userH *user.Handler,
    topicH *topic.Handler,
    // ... 其余 handler
) {
    // ===== 公开接口（无需认证） =====
    pub := r.Group("")
    {
        pub.POST("/api/user/login", userH.Login)
        pub.POST("/api/user/refresh", userH.RefreshToken)
        pub.PUT("/api/user/pre_authentication", userH.PreAuth)
        pub.POST("/api/user/official/login", userH.OfficialLogin)
        pub.POST("/api/user/official/certification", userH.OfficialCert)
        pub.GET("/api/user/nickname/random", userH.RandomNickname)

        pub.POST("/api/theme/campus/init", themeH.InitCampusThemes)
        pub.GET("/api/theme/campus", themeH.GetCampusThemes)

        pub.GET("/api/support/:key", otherH.Support.GetByKey)
        pub.GET("/api/support/list", otherH.Support.List)
        pub.GET("/api/term/list", schoolH.TermList)
        pub.GET("/api/term", schoolH.CurrentTerm)
        pub.GET("/api/notice/list", otherH.Notice.List)
        pub.GET("/api/ad/list_level", otherH.Ad.ListByLevel)

        pub.GET("/file/:md5", fileH.Download)
        pub.GET("/file", fileH.ListPublic)

        pub.POST("/api/wx/unlimited/wxa_code", userH.UnlimitedWXACode)
    }

    // ===== 用户接口（JWT + BlackList + Log） =====
    api := r.Group("/api")
    api.Use(
        middleware.JWTAuth(jwt, rds),
        middleware.BlackListCheck(rds),
        middleware.RequestLog(logger),
    )
    {
        // User
        api.GET("/user", userH.GetCurrent)
        api.PUT("/user", userH.Edit)
        api.POST("/user/authentication", userH.Authenticate)
        // ... 完整列表见 04-api-endpoints.md

        // Topic（创建需认证检查）
        api.POST("/topic", middleware.RequireVerified(userSvc), topicH.Create)
        api.DELETE("/topic/:id", topicH.Delete)
        api.GET("/topic/:id", topicH.GetByID)
        // ...

        // Comment, Like, Collection, Vote, Event ...
    }

    // ===== File（仅 JWT） =====
    fileAuth := r.Group("/file")
    fileAuth.Use(middleware.JWTAuth(jwt, rds))
    {
        fileAuth.POST("/upload", fileH.Upload)
        fileAuth.DELETE("/del/:md5", fileH.Delete)
    }

    // ===== WebSocket =====
    r.GET("/chat", chatH.HandleUpgrade)

    // ===== Health & Metrics =====
    r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "UP"}) })
    r.GET("/metrics", gin.WrapH(promhttp.Handler()))
}
```

### 6.2 ecampus-crm 路由注册

```go
// cmd/ecampus-crm/routes.go
func registerAdminRoutes(
    r *gin.Engine,
    cfg *config.Config,
    jwt *jwtutil.Helper,
    rds *redis.Client,
    userAdmin *user.AdminHandler,
    topicAdmin *topic.AdminHandler,
    // ... 其余 admin handler
) {
    // ===== Admin 登录（排除 JWT） =====
    r.POST("/admin/user/login", userAdmin.Login)

    // ===== Admin 接口（JWT + BlackList + Log + AdminCheck） =====
    admin := r.Group("/admin")
    admin.Use(
        middleware.JWTAuth(jwt, rds),
        middleware.BlackListCheck(rds),
        middleware.RequestLog(logger),
        middleware.AdminCheck(cfg),
    )
    {
        // User management
        admin.POST("/user", userAdmin.Add)
        admin.POST("/user/add", userAdmin.AddAdmin)
        admin.DELETE("/user/:id", userAdmin.Delete)
        admin.PUT("/user/:id", userAdmin.Edit)
        admin.GET("/user/:id", userAdmin.GetByID)
        admin.GET("/user/list", userAdmin.List)
        admin.POST("/user/clear", userAdmin.ClearAuth)
        admin.POST("/user/course", userAdmin.FetchCourse)
        admin.POST("/user/add_black_list", userAdmin.AddBlackList)
        admin.DELETE("/user/del_black_list", userAdmin.DelBlackList)
        admin.GET("/user/black_list", userAdmin.GetBlackList)
        admin.GET("/user/certification/list", userAdmin.CertList)
        admin.POST("/user/certification/review", userAdmin.CertReview)

        // Topic, Comment, Theme, File, School, Other, Event, Monitor...
        // 完整列表见 04-api-endpoints.md
    }

    // Health
    r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "UP"}) })
}
```

---

## 7. 配置文件

### 7.1 ecampus 配置

```yaml
# configs/ecampus/application-dev.yml
server:
  port: 8080

mysql:
  dsn: "root:password@tcp(127.0.0.1:9801)/campus?charset=utf8mb4&parseTime=True&loc=Asia%2FShanghai"
  max_lifetime: 60000

mongo:
  uri: "mongodb://admin:password@127.0.0.1:9803/campus?authSource=admin"
  database: campus

redis:
  addr: "127.0.0.1:9800"
  password: "password"

rabbitmq:
  url: "amqp://campus:password@127.0.0.1:9072/"

jwt:
  secret: "@12asd."
  token_minutes: 60
  refresh_token_minutes: 2880
  issue: campus

cos:
  access_key_id: "xxx"
  access_key_secret: "xxx"
  bucket_name: "campus-1318456870"
  region: "ap-guangzhou"
  base_url: "https://campus-1318456870.cos.ap-guangzhou.myqcloud.com/"
  base_cdn: "https://cdn.fangfangfang.top/"
  compress: "webp"
  compress_bucket_name: "campus-compress-1318456870"
  compress_base_cdn: "https://cdn-compress.fangfangfang.top/"

wx:
  appid: "wx40b4d6894a6c31ed"
  secret: "xxx"

custom:
  default_avatar: "5e39bbaac50af63e6d221b4f9e7fbee8,aaca0f5eb4d2d98a6ce6dffa99f8254b"
  default_anonymous_avatar: "39862f6788f0d8852a2c095e0d4f7057"
  page_size: 15
  max_file_size_mb: 15

jw:
  base_url: "https://www.fangfangfang.top/sztu_jw"
  api_key: "U2FsdGVkX18BNFq4BRJwIzXUPmKQ2Ngj"

encryption:
  key: "W0F7PePvolUJHmZtkv1MusWpwhpVJIJI"

admin:
  power_sign: 999

logging:
  level: info
  file_path: logs/ecampus.log
```

### 7.2 ecampus-crm 配置

```yaml
# configs/ecampus-crm/application-dev.yml
server:
  port: 8081

# 数据库配置与 ecampus 相同（连接同一套）
mysql:
  dsn: "root:password@tcp(127.0.0.1:9801)/campus?charset=utf8mb4&parseTime=True&loc=Asia%2FShanghai"

mongo:
  uri: "mongodb://admin:password@127.0.0.1:9803/campus?authSource=admin"
  database: campus

redis:
  addr: "127.0.0.1:9800"
  password: "password"

# CRM 不需要 RabbitMQ 配置

jwt:
  secret: "@12asd."
  token_minutes: 60
  refresh_token_minutes: 2880
  issue: campus

admin:
  power_sign: 999

logging:
  level: info
  file_path: logs/ecampus-crm.log
```

---

## 8. Docker 部署

### 8.1 ecampus Dockerfile

```dockerfile
# deployments/ecampus/Dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o ecampus ./cmd/ecampus

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
ENV TZ=Asia/Shanghai
WORKDIR /app
COPY --from=builder /build/ecampus .
COPY --from=builder /build/configs/ecampus ./configs/ecampus
EXPOSE 8080
ENTRYPOINT ["./ecampus"]
```

### 8.2 ecampus-crm Dockerfile

```dockerfile
# deployments/ecampus-crm/Dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o ecampus-crm ./cmd/ecampus-crm

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
ENV TZ=Asia/Shanghai
WORKDIR /app
COPY --from=builder /build/ecampus-crm .
COPY --from=builder /build/configs/ecampus-crm ./configs/ecampus-crm
EXPOSE 8081
ENTRYPOINT ["./ecampus-crm"]
```

### 8.3 docker-compose（本地开发）

```yaml
# deployments/docker-compose.yml
version: "3.8"

services:
  ecampus:
    build:
      context: ../
      dockerfile: deployments/ecampus/Dockerfile
    ports:
      - "8080:8080"
    environment:
      - APP_PROFILE=dev
    depends_on:
      - mysql
      - redis
      - mongo
      - rabbitmq

  ecampus-crm:
    build:
      context: ../
      dockerfile: deployments/ecampus-crm/Dockerfile
    ports:
      - "8081:8081"
    environment:
      - APP_PROFILE=dev
    depends_on:
      - mysql
      - redis
      - mongo

  mysql:
    image: mysql:8.0
    ports: ["9801:3306"]
    environment:
      MYSQL_ROOT_PASSWORD: "Xy8@Kp3!Mv9#Jq"
      MYSQL_DATABASE: campus
    volumes:
      - mysql_data:/var/lib/mysql

  redis:
    image: redis:7-alpine
    ports: ["9800:6379"]
    command: redis-server --requirepass "B6yz7P@x9d&."

  mongo:
    image: mongo:6
    ports: ["9803:27017"]
    environment:
      MONGO_INITDB_ROOT_USERNAME: admin
      MONGO_INITDB_ROOT_PASSWORD: "F6m@t3YhexitRu9k"
    volumes:
      - mongo_data:/data/db

  rabbitmq:
    image: rabbitmq:3-management
    ports:
      - "9072:5672"
      - "9073:15672"
    environment:
      RABBITMQ_DEFAULT_USER: campus
      RABBITMQ_DEFAULT_PASS: "RbbtMq98aA@45zB!"

volumes:
  mysql_data:
  mongo_data:
```

---

## 9. go.mod

```
module github.com/your-org/ecampus

go 1.22

require (
    github.com/gin-gonic/gin               v1.9.1
    gorm.io/gorm                            v1.25.x
    gorm.io/driver/mysql                    v1.5.x
    go.mongodb.org/mongo-driver             v1.13.x
    github.com/redis/go-redis/v9            v9.4.x
    github.com/rabbitmq/amqp091-go          v1.9.x
    github.com/gorilla/websocket            v1.5.x
    github.com/golang-jwt/jwt/v5            v5.2.x
    github.com/spf13/viper                  v1.18.x
    go.uber.org/zap                         v1.27.x
    github.com/robfig/cron/v3               v3.0.x
    github.com/go-playground/validator/v10  v10.17.x
    github.com/tencentyun/cos-go-sdk-v5     v0.7.x
    github.com/prometheus/client_golang      v1.18.x
    github.com/bwmarrin/snowflake           v0.3.x
    gopkg.in/gomail.v2                      v2.0.0
    github.com/go-ego/gse                   v0.80.x
)
```

---

## 10. 生命周期与职责分配

| 职责 | ecampus | ecampus-crm |
|------|---------|-------------|
| 用户端 API | Y | - |
| 管理端 API | - | Y |
| WebSocket 聊天 | Y | - |
| MQ 消费者（内容审核、搜索索引、通知等） | Y | - |
| MQ 生产者（帖子创建、评论等触发） | Y | - |
| 定时任务（推荐排行、埋点入库、指标刷新） | Y | - |
| Prometheus 指标 | Y | - |
| 管理员登录 | - | Y |
| 管理员 CRUD | - | Y |
| 健康检查 | Y | Y |
