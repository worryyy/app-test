# Go 编码规范与项目约定

> Ecampus Go 重写项目的编码"宪法"。所有代码必须遵循本规范。
>
> 本项目是 **monorepo 双服务架构**：
> - **ecampus** — 用户端服务（`/api/**`、`/file/**`、`/chat` WebSocket）
> - **ecampus-crm** — 管理后台服务（`/admin/**`）
>
> 两个服务共享同一套域包、数据库、Redis 和 MQ。

---

## 1. 项目结构

### 1.1 整体目录

```
ecampus/
  cmd/
    ecampus/
      main.go                  # 用户端服务入口
    ecampus-crm/
      main.go                  # 管理后台服务入口

  internal/
    ── 业务域包 ──
    user/                      # 用户域
      model.go                 #   数据结构（两服务共享）
      service.go               #   业务逻辑（两服务共享）
      handler.go               #   用户端 handlers → ecampus 注册
      admin.go                 #   管理端 handlers → ecampus-crm 注册
    topic/                     # 帖子域
      model.go
      service.go
      search.go                #   搜索逻辑（分词 + MongoDB text search）
      handler.go
      admin.go
    comment/                   # 评论域
      model.go
      service.go
      handler.go
      admin.go
    theme/                     # 主题/板块域
      model.go
      service.go
      handler.go               #   公开接口（init、list）
      admin.go                 #   管理接口（编辑、推荐配置等）
    file/                      # 文件上传域
      model.go
      service.go
      handler.go
      admin.go
    chat/                      # 聊天域（仅 ecampus 使用）
      model.go
      service.go
      handler.go               #   REST 接口（会话、消息、通知）
      ws.go                    #   WebSocket handler
    level/                     # 经验签到域（仅 ecampus 使用）
      model.go
      service.go
      handler.go
    school/                    # 教务域
      model.go
      service.go
      handler.go
      admin.go
      jw.go                    #   教务系统 HTTP 对接（登录、课程抓取）
    other/                     # 广告/公告/投票/敏感词/举报/商户主题/前端支持
      model.go                 #   所有小域的数据结构
      ad.go                    #   广告 service + 公开 handler
      ad_admin.go              #   广告管理 handler
      notice.go                #   公告 service + 公开 handler
      notice_admin.go          #   公告管理 handler
      vote.go                  #   投票 service + handler
      sensitive_admin.go       #   敏感词管理 handler + service（纯管理）
      report.go                #   举报 service + 用户 handler
      report_admin.go          #   举报管理 handler
      merchant_admin.go        #   商户主题管理 handler + service（纯管理）
      support.go               #   前端支持 service + 公开 handler
      support_admin.go         #   前端支持管理 handler
    event/                     # 前端埋点
      model.go
      service.go
      handler.go
      admin.go
    monitor/                   # 监控缓存（仅 ecampus-crm 使用）
      service.go
      admin.go

    ── 共享基础设施 ──
    mq/                        # 消息队列
      config.go                #   Exchange/Queue/RoutingKey 常量
      producer.go              #   BaseProducer + 各业务 Producer
      consumer.go              #   各业务 Consumer 注册与启动
      base.go                  #   BaseConsumer（去重 + 重试）
    middleware/                # HTTP 中间件
      jwt.go                   #   JWT 验证
      admin.go                 #   管理员权限检查
      blacklist.go             #   黑名单拦截
      log.go                   #   请求日志
      cors.go                  #   跨域
      auth_check.go            #   认证检查（替代 Java AOP）
    cron/                      # 定时任务（由 ecampus 运行）
      suggest.go               #   推荐排行生成
      event_flush.go           #   埋点批量入库
      metrics.go               #   活跃用户指标刷新
    pkg/                       # 内部共享工具（零业务逻辑）
      result/                  #   统一响应格式 Result/RC
      jwtutil/                 #   JWT 签发/验证/Claims
      wxutil/                  #   微信 API（登录、内容安全、订阅消息、小程序码）
      cosutil/                 #   腾讯 COS 上传（含万象 CI 压缩）
      encrypt/                 #   AES/DES 加解密
      rediskey/                #   所有 Redis key 定义与构造函数
      snowflake/               #   Snowflake ID 生成
      config/                  #   Viper 配置加载

  configs/
    ecampus/
      application.yml          # 基础配置
      application-dev.yml      # 开发环境
      application-prod.yml     # 生产环境
    ecampus-crm/
      application.yml
      application-dev.yml
      application-prod.yml

  deployments/
    ecampus/Dockerfile
    ecampus-crm/Dockerfile
    docker-compose.yml         # 本地开发用（含 MySQL/Redis/Mongo/RabbitMQ）

  go.mod
  go.sum
```

### 1.2 域包内部结构

每个域包遵循统一模式：

| 文件 | 职责 | 谁引用 |
|------|------|--------|
| `model.go` | 数据结构（struct + 枚举 + 错误变量） | 两个服务共享 |
| `service.go` | 业务逻辑 + 直接操作 DB | 两个服务共享 |
| `handler.go` | 用户端 HTTP handlers（`/api/**`） | 仅 `cmd/ecampus/` |
| `admin.go` | 管理端 HTTP handlers（`/admin/**`） | 仅 `cmd/ecampus-crm/` |

**规则**：
- `service.go` 是核心——两个服务的 handler 都调用同一个 service
- 没有 admin API 的域（`chat/`、`level/`）不需要 `admin.go`
- 只有 admin API 的域（`monitor/`）不需要 `handler.go`
- `model.go` 中定义的 struct 同时携带 `gorm:`/`bson:` 和 `json:` tag，**不搞 VO/DTO/PO**

### 1.3 文件大小与拆分

**单文件不超过 300 行**。超过时按子关注点拆分：

```
# 拆分前
user/service.go        (350 行，太长)

# 拆分后
user/service.go        (200 行，核心 CRUD)
user/service_follow.go (150 行，关注/粉丝逻辑)
```

拆分后的文件仍在同一个 package，只是物理分开。常见拆分点：

| 触发条件 | 拆分方式 |
|---------|---------|
| service 超 300 行 | 按子域拆：`service_follow.go`、`service_search.go` |
| handler 超 300 行 | 按功能拆：`handler_identity.go`、`handler_follow.go` |
| admin 超 300 行 | 按子域拆：`admin_blacklist.go`、`admin_cert.go` |
| model 超 300 行 | 拆出请求结构体：`model_req.go` |

### 1.4 包依赖规则

```
cmd/ecampus/        ──→  internal/user/, topic/, ..., middleware/, mq/, cron/, pkg/
cmd/ecampus-crm/    ──→  internal/user/, topic/, ..., middleware/, mq/, pkg/

user/               ──→  pkg/result, pkg/jwtutil, pkg/rediskey, pkg/config
topic/              ──→  pkg/result, pkg/rediskey, mq/
comment/            ──→  pkg/result, pkg/rediskey, mq/
chat/               ──→  pkg/result, pkg/jwtutil, pkg/rediskey
school/             ──→  pkg/result, pkg/rediskey, pkg/encrypt, mq/
other/              ──→  pkg/result, pkg/rediskey
middleware/         ──→  pkg/result, pkg/jwtutil, pkg/rediskey
mq/                 ──→  pkg/rediskey
```

**禁止**：
- 域包之间不能循环依赖。`topic/` 可以 import `user/` 的导出函数，但 `user/` 不能反向 import `topic/`
- 如果出现循环，通过**接口解耦**或**将共享 struct 提取到 `pkg/`**
- `pkg/` 下的包**互相不依赖**，也**不依赖任何域包**

**循环依赖避免策略**：

当 `user/service.go` 需要查帖子点赞数（`campus_topic_like` 集合）来计算用户统计时，**不要 import `topic/`**。直接用 mongo-driver 按集合名查询：

```go
// internal/user/service.go — 直接查 MongoDB 集合，不导入 topic 包
func (s *Service) GetUserStats(ctx context.Context, userID string) (*UserProfile, error) {
    likeCount, _ := s.mongoDB.Collection("campus_topic_like").CountDocuments(ctx,
        bson.M{"userId": userID})
    // ... 不需要 import topic/ 包
}
```

这是 Go 的优势：mongo-driver 不需要导入 model 包就能查询，用 `bson.M` 即可。

---

## 2. 命名

### 包名
- 全小写单词：`user`、`topic`、`comment`
- 工具包用具体功能命名：`jwtutil`、`encrypt`、`rediskey`
- 禁止 `utils`、`helpers`、`common`

### 变量与函数
- 局部短名：`u`（user）、`t`（topic）、`ctx`、`err`
- 导出函数清晰动词开头：`CreateTopic`、`GetByOpenID`
- 未导出函数可更短：`buildQuery`、`parseToken`

### 结构体
- 导出结构体用名词：`Topic`、`User`、`Comment`
- 请求结构体只在 shape 不同于 model 时才定义 `CreateTopicReq`
- 不用 `I` 前缀

### 接口
- 单方法 `-er` 后缀：`Reader`、`Validator`
- **不要为"面向接口编程"定义接口**——只在需要多态或测试 mock 时才用
- 接口在**使用方**定义

### 常量与枚举
```go
type AccountType int

const (
    AccountTypeWechat   AccountType = iota + 1
    AccountTypeOfficial
    AccountTypeAnonymous
)
```

### 文件名
- 全小写下划线分隔：`service_search.go`、`admin_blacklist.go`
- 测试文件：`service_test.go`

---

## 3. 错误处理

### 原则
- **永远检查 error**，不用 `_` 忽略
- **向上传播**，handler 统一转 HTTP 响应
- **不 panic**，除非启动阶段不可恢复

### 模式
```go
// 各域 model.go 中定义业务错误
var (
    ErrUserNotFound     = errors.New("user not found")
    ErrTokenExpired     = errors.New("token expired")
    ErrPermissionDenied = errors.New("permission denied")
    ErrAlreadySigned    = errors.New("already signed today")
)

// service 返回 error
func (s *Service) GetByOpenID(ctx context.Context, openID string) (*User, error) {
    var u User
    err := s.db.WithContext(ctx).Where("openId = ?", openID).First(&u).Error
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, nil // 不存在返回 nil，不是 error
    }
    if err != nil {
        return nil, fmt.Errorf("get user by openid %s: %w", openID, err)
    }
    return &u, nil
}

// handler 三步曲：绑定 → 调 service → 返回
func (h *Handler) Create(c *gin.Context) {
    var req CreateTopicReq
    if err := c.ShouldBindJSON(&req); err != nil {
        result.Fail(c, result.CodeParamError, "参数错误")
        return
    }
    id, err := h.svc.Create(c.Request.Context(), middleware.GetUserID(c), &req)
    if err != nil {
        result.HandleError(c, err)
        return
    }
    result.Success(c, id)
}
```

### 统一错误映射
```go
// internal/pkg/result/error.go
func HandleError(c *gin.Context, err error) {
    switch {
    case errors.Is(err, user.ErrUserNotFound):
        Fail(c, CodeNotExisted, err.Error())
    case errors.Is(err, ErrPermissionDenied):
        Fail(c, CodeForbidden, err.Error())
    default:
        Fail(c, CodeUnknownError, "系统错误")
    }
}
```

---

## 4. 统一响应格式

与 Java 版 JSON 结构**完全一致**：

```go
// internal/pkg/result/result.go
type Result struct {
    Success bool        `json:"success"`
    Code    int         `json:"code"`
    Msg     string      `json:"msg"`
    Data    interface{} `json:"data"`
}

const (
    CodeSuccess       = 200
    CodeFail          = 400
    CodeUnknownError  = -1
    CodeParamError    = 1
    CodeNotExisted    = 3
    CodeForbidden     = 5
    CodeTokenError    = 10002
    CodeTokenNotExisted = 10003
    CodeTokenInvalid  = 10004
    CodeAuthNotExisted = 10005
    CodeRTKNotExisted = 10006
    CodeRTKUsed       = 10007
)

func Success(c *gin.Context, data interface{}) {
    c.JSON(200, Result{Success: true, Code: CodeSuccess, Msg: "成功", Data: data})
}

func Fail(c *gin.Context, code int, msg string) {
    c.JSON(200, Result{Success: false, Code: code, Msg: msg})
}
```

### 零值兼容

Go 默认零值行为与 FastJSON 一致（`""`, `0`, `false`）。唯一注意：nil slice → `null`，空 slice → `[]`。

```go
// 确保 slice 不为 nil
func EnsureSlice[T any](s []T) []T {
    if s == nil { return []T{} }
    return s
}
```

### 分页响应

```go
// internal/pkg/result/page.go

// MySQL 分页（对应 MyBatis-Plus Page，JSON key 必须一致）
type PageResult[T any] struct {
    Records []T   `json:"records"`
    Total   int64 `json:"total"`
    Current int   `json:"current"`
    Size    int   `json:"size"`
    Pages   int   `json:"pages"`
}

func NewPage[T any](records []T, total int64, current, size int) *PageResult[T] {
    pages := int(total) / size
    if int(total)%size > 0 { pages++ }
    return &PageResult[T]{
        Records: EnsureSlice(records), Total: total,
        Current: current, Size: size, Pages: pages,
    }
}

// MongoDB/自定义分页
type CusPage[T any] struct {
    Data    []T   `json:"data"`
    Current int   `json:"current"`
    Total   int64 `json:"total"`
    Size    int   `json:"size"`
}
```

---

## 5. 中间件

### 共享中间件

```go
// internal/middleware/jwt.go — 两个服务都使用
func JWTAuth(helper *jwtutil.Helper, rds *redis.Client) gin.HandlerFunc {
    return func(c *gin.Context) {
        if c.Request.Method == "OPTIONS" { c.Next(); return }
        token := c.GetHeader("Authorization")
        if token == "" {
            result.Fail(c, result.CodeAuthNotExisted, "authorization 找不到")
            c.Abort(); return
        }
        claims, err := helper.ParseAndVerify(token, rds)
        if err != nil {
            result.Fail(c, result.CodeTokenInvalid, "token invalid")
            c.Abort(); return
        }
        c.Set("claims", claims)
        c.Next()
    }
}

func GetUserID(c *gin.Context) int64 {
    claims, _ := c.Get("claims")
    if claims == nil { return 0 }
    return claims.(*jwtutil.Claims).UserID
}

func GetClaims(c *gin.Context) *jwtutil.Claims {
    v, _ := c.Get("claims")
    if v == nil { return nil }
    return v.(*jwtutil.Claims)
}
```

### 各服务中间件链

**ecampus**（用户端）：
```go
api := r.Group("/api")
api.Use(middleware.JWTAuth(...))        // order 1
api.Use(middleware.BlackListCheck(...)) // order 2
api.Use(middleware.RequestLog(...))     // order 3
```

**ecampus-crm**（管理端）：
```go
admin := r.Group("/admin")
admin.Use(middleware.JWTAuth(...))        // order 1
admin.Use(middleware.BlackListCheck(...)) // order 2
admin.Use(middleware.RequestLog(...))     // order 3
admin.Use(middleware.AdminCheck(...))     // order 4（仅 CRM）
```

### Handler 包装（替代 Java AOP）

```go
// internal/middleware/auth_check.go
func RequireVerified(userSvc *user.Service) gin.HandlerFunc {
    return func(c *gin.Context) {
        if c.Request.Method == "GET" || c.Request.Method == "OPTIONS" {
            c.Next(); return
        }
        u, _ := userSvc.GetByID(c.Request.Context(), GetUserID(c))
        if u == nil || !u.StuIsCheck {
            result.Fail(c, result.CodeForbidden, "请先完成认证")
            c.Abort(); return
        }
        c.Next()
    }
}

// 路由注册时使用
api.POST("/topic", middleware.RequireVerified(userSvc), topicHandler.Create)
```

---

## 6. 依赖注入

**不用 DI 框架**。构造函数注入，两层结构：

```go
// internal/user/service.go — service 直接持有 DB 连接，没有 repo 层
type Service struct {
    db      *gorm.DB
    mongoDB *mongo.Database
    redis   *redis.Client
    cfg     *config.Config
    logger  *zap.Logger
}

func NewService(db *gorm.DB, mongo *mongo.Database, rds *redis.Client,
    cfg *config.Config, l *zap.Logger) *Service {
    return &Service{db: db, mongoDB: mongo, redis: rds, cfg: cfg, logger: l}
}

// internal/user/handler.go — 用户端 handler 持有 service
type Handler struct{ svc *Service }
func NewHandler(s *Service) *Handler { return &Handler{svc: s} }

// internal/user/admin.go — 管理端 handler 也持有同一个 service
type AdminHandler struct{ svc *Service }
func NewAdminHandler(s *Service) *AdminHandler { return &AdminHandler{svc: s} }
```

### 两个服务的组装

```go
// cmd/ecampus/main.go — 用户端服务
func main() {
    cfg := config.Load("configs/ecampus")
    db, mongoDB, rds, amqpConn, logger := initInfra(cfg)

    // 共享 service
    userSvc   := user.NewService(db, mongoDB, rds, cfg, logger)
    topicSvc  := topic.NewService(mongoDB, rds, mqProducers, logger)
    commentSvc := comment.NewService(mongoDB, rds, mqProducers, logger)
    // ... 其余域 service

    // 只注册用户端 handler
    userH    := user.NewHandler(userSvc)
    topicH   := topic.NewHandler(topicSvc)
    commentH := comment.NewHandler(commentSvc)
    // ...

    r := gin.New()
    r.Use(gin.Recovery(), middleware.CORS())
    registerUserRoutes(r, cfg, rds, userH, topicH, commentH, ...)

    // 定时任务（ecampus 负责运行）
    cr := initCron(topicSvc, eventSvc, rds, logger)
    cr.Start()

    // MQ 消费者
    mqConsumers := mq.NewConsumers(amqpConn, rds, mongoDB, logger, topicSvc, commentSvc)
    mqConsumers.Start()

    // 启动 + 优雅关闭
    gracefulServe(cfg.Server.Port, r, cr, mqConsumers, logger)
}

// cmd/ecampus-crm/main.go — 管理后台服务
func main() {
    cfg := config.Load("configs/ecampus-crm")
    db, mongoDB, rds, _, logger := initInfra(cfg)

    // 共享 service（同一套业务逻辑）
    userSvc   := user.NewService(db, mongoDB, rds, cfg, logger)
    topicSvc  := topic.NewService(mongoDB, rds, nil, logger) // CRM 不需要 MQ producer
    // ...

    // 只注册管理端 handler
    userAdmin    := user.NewAdminHandler(userSvc)
    topicAdmin   := topic.NewAdminHandler(topicSvc)
    // ...

    r := gin.New()
    r.Use(gin.Recovery(), middleware.CORS())
    registerAdminRoutes(r, cfg, rds, userAdmin, topicAdmin, ...)

    gracefulServe(cfg.Server.Port, r, nil, nil, logger)
}
```

### 为什么没有 repo 层

- 数据访问就是 `db.Where(...).First(&u)` 或 `coll.FindOne(ctx, filter)` 一两行代码
- 包一层 repo 只是搬了位置，增加跳转成本
- 复杂查询（如推荐排行）在同包拆文件（`service_suggest.go`），不需要新包

---

## 7. Context 传递

- 所有跨层调用第一个参数是 `context.Context`
- handler 从 `c.Request.Context()` 获取，传入 service
- **不在 context 中存业务数据**。用户信息通过函数参数传入 service

```go
// 正确：用户 ID 作为参数
func (s *Service) Create(ctx context.Context, userID int64, req *CreateTopicReq) error

// 错误：从 context 取
func (s *Service) Create(ctx context.Context, req *CreateTopicReq) error {
    userID := ctx.Value("userID").(int64) // 不要这样
}
```

---

## 8. 数据库操作

### GORM (MySQL)
```go
// internal/user/service.go — 直接在 service 中操作
func (s *Service) GetByOpenID(ctx context.Context, openID string) (*User, error) {
    var u User
    err := s.db.WithContext(ctx).Where("openId = ?", openID).First(&u).Error
    if errors.Is(err, gorm.ErrRecordNotFound) { return nil, nil }
    return &u, err
}

func (s *Service) List(ctx context.Context, page, size int, name string) (*result.PageResult[User], error) {
    var total int64
    var list []User
    q := s.db.WithContext(ctx).Model(&User{})
    if name != "" { q = q.Where("nickname LIKE ?", "%"+name+"%") }
    q.Count(&total)
    q.Offset((page - 1) * size).Limit(size).Find(&list)
    return result.NewPage(list, total, page, size), nil
}
```

### MongoDB
```go
// internal/topic/service.go
func (s *Service) FindByID(ctx context.Context, id string) (*Topic, error) {
    oid, err := primitive.ObjectIDFromHex(id)
    if err != nil { return nil, fmt.Errorf("invalid topic id: %w", err) }
    var t Topic
    err = s.coll().FindOne(ctx, bson.M{"_id": oid, "hasCheck": true}).Decode(&t)
    if errors.Is(err, mongo.ErrNoDocuments) { return nil, nil }
    return &t, err
}

// 常用集合的快捷方法（避免每次写 collection name）
func (s *Service) coll() *mongo.Collection {
    return s.mongoDB.Collection("campus_topic")
}
```

---

## 9. 并发

- `sync.RWMutex` 保护共享状态（如 WebSocket session map）
- `chan` 做 goroutine 通信
- goroutine 必须能通过 context 取消或 done channel 关闭
- `errgroup` 做并发聚合

```go
// WebSocket session 管理
type SessionManager struct {
    mu       sync.RWMutex
    sessions map[string]*Session
}

func (m *SessionManager) Get(userID string) (*Session, bool) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    s, ok := m.sessions[userID]
    return s, ok
}
```

---

## 10. 日志

使用 `zap` 结构化日志：

```go
logger.Info("topic created", zap.String("topicID", id), zap.Int64("userID", userID))
logger.Error("insert failed", zap.Error(err), zap.String("collection", "campus_topic"))
```

| 级别 | 用途 |
|------|------|
| Debug | SQL 语句、请求详情（仅开发环境） |
| Info | 业务事件（登录、发帖、MQ 处理） |
| Warn | 可恢复异常（COS 降级、重试） |
| Error | 不可恢复错误（DB 连接失败、MQ 投递失败） |

**禁止**：日志中打印密码、token 全文、身份证号。

---

## 11. 测试

```go
func TestParseToken(t *testing.T) {
    tests := []struct {
        name    string
        token   string
        wantErr bool
    }{
        {"valid", "xxx.yyy.zzz", false},
        {"empty", "", true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := ParseToken(tt.token)
            if (err != nil) != tt.wantErr {
                t.Errorf("ParseToken() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

- 测试文件与被测文件同目录
- 数据库测试用 `testcontainers-go` 或 mock
- 集成测试放在 `cmd/ecampus/main_test.go`（启动完整 context）

---

## 12. 配置管理

```go
// internal/pkg/config/config.go
type Config struct {
    Server   ServerConfig   `mapstructure:"server"`
    MySQL    MySQLConfig    `mapstructure:"mysql"`
    Mongo    MongoConfig    `mapstructure:"mongo"`
    Redis    RedisConfig    `mapstructure:"redis"`
    JWT      JWTConfig      `mapstructure:"jwt"`
    COS      COSConfig      `mapstructure:"cos"`
    WX       WXConfig       `mapstructure:"wx"`
    RabbitMQ RabbitMQConfig `mapstructure:"rabbitmq"`
    Custom   CustomConfig   `mapstructure:"custom"`
    Admin    AdminConfig    `mapstructure:"admin"`
}

func Load(configDir string) *Config {
    v := viper.New()
    v.SetConfigName("application")
    v.AddConfigPath(configDir)
    v.ReadInConfig()

    // Profile overlay
    profile := os.Getenv("APP_PROFILE")
    if profile == "" { profile = "dev" }
    v.SetConfigName("application-" + profile)
    v.MergeInConfig()

    var cfg Config
    v.Unmarshal(&cfg)
    return &cfg
}
```

两个服务各自加载各自的配置目录：
- `cmd/ecampus/` → `config.Load("configs/ecampus")`
- `cmd/ecampus-crm/` → `config.Load("configs/ecampus-crm")`

---

## 13. 代码复用原则

### 该复用
| 层 | 说明 |
|----|------|
| `service.go` | 两个服务共享同一套 service |
| `model.go` | 所有数据结构共享 |
| `middleware/` | JWT、CORS、Log、BlackList 共享 |
| `pkg/` | 工具包完全共享 |
| `mq/` | MQ 基础设施共享 |

### 不该强行复用
| 层 | 说明 |
|----|------|
| `main.go` | 两个入口各自初始化，不要 flag 切换 |
| 路由注册 | 各服务有自己的 `registerRoutes()` |
| 配置文件 | 各服务可以有不同配置 |
| Dockerfile | 各服务独立构建 |

---

## 14. 代码风格总则

- **简短胜于冗长**：函数不超过 80 行，文件不超过 300 行
- **扁平胜于嵌套**：early return 减少 if-else 深度
- **显式胜于隐式**：不用 `init()`（除非注册 driver），不用全局变量
- **组合胜于继承**：struct 嵌入 + 接口组合
- **标准库优先**：能用标准库不引第三方
- **格式化非协商**：`gofmt` + `go vet` + `golangci-lint` 提交前必过
