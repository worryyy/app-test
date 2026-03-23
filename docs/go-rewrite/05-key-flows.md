# 关键流程伪代码

> 本文档覆盖系统中最复杂的业务流程。简单 CRUD 参考 04-api-endpoints.md 即可实现。
>
> **所有伪代码中的 DB 操作都是直接在 service 方法中执行的 GORM/mongo-driver/go-redis 调用，没有独立的 repo 层。**
>
> 每个流程标注了所属 **服务** 和 **包路径**。

---

## 1. JWT 认证全流程

### 1.1 Token 生成 — `internal/pkg/jwtutil/helper.go`

> 两个服务共享此逻辑

```go
func (h *Helper) GenerateTokenPair(user *user.User) (token, refreshToken string, err error) {
    now := time.Now()

    claims := &Claims{
        UserID:      user.ID,
        OpenID:      user.OpenID,
        Power:       user.Power,
        AccountType: user.AccountType,
        RootUserID:  user.RootUserID,
        RegisteredClaims: jwt.RegisteredClaims{
            Issuer:    h.cfg.Issue,
            ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(h.cfg.TokenMinutes) * time.Minute)),
            ID:        snowflake.Generate().String(), // 防止同一用户生成相同 token
        },
    }
    token, _ = jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(h.cfg.Secret))

    refreshClaims := *claims
    refreshClaims.ExpiresAt = jwt.NewNumericDate(now.Add(time.Duration(h.cfg.RefreshTokenMinutes) * time.Minute))
    refreshClaims.ID = snowflake.Generate().String()
    refreshToken, _ = jwt.NewWithClaims(jwt.SigningMethodHS256, &refreshClaims).SignedString([]byte(h.cfg.Secret))

    // 存入 Redis
    ctx := context.Background()
    tokenKey := rediskey.Token(sha1Hex(token))
    h.rds.Set(ctx, tokenKey, rediskey.TokenStatusOK, time.Duration(h.cfg.TokenMinutes)*time.Minute)

    refreshKey := rediskey.RefreshToken(sha1Hex(refreshToken))
    h.rds.Set(ctx, refreshKey, rediskey.TokenStatusOK, time.Duration(h.cfg.RefreshTokenMinutes)*time.Minute)

    return token, refreshToken, nil
}
```

### 1.2 JWT 中间件 — `internal/middleware/jwt.go`

> 两个服务共享

```go
func JWTAuth(helper *jwtutil.Helper, rds *redis.Client) gin.HandlerFunc {
    return func(c *gin.Context) {
        if c.Request.Method == "OPTIONS" { c.Next(); return }

        token := c.GetHeader("Authorization")
        if token == "" {
            result.Fail(c, result.CodeAuthNotExisted, "authorization 找不到")
            c.Abort(); return
        }

        parts := strings.Split(token, ".")
        if len(parts) != 3 {
            result.Fail(c, result.CodeTokenInvalid, "token invalid")
            c.Abort(); return
        }

        // Redis 存在性
        tokenKey := rediskey.Token(sha1Hex(token))
        if rds.Get(c.Request.Context(), tokenKey).Err() != nil {
            result.Fail(c, result.CodeTokenNotExisted, "token 不存在,或已过期")
            c.Abort(); return
        }

        // JWT 签名
        claims, err := helper.Parse(token)
        if err != nil {
            result.Fail(c, result.CodeTokenInvalid, "token invalid")
            c.Abort(); return
        }

        c.Set("claims", claims)
        c.Next()
    }
}
```

### 1.3 Token 刷新 — `internal/user/service.go`

> ecampus 服务

```go
func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (string, string, error) {
    refreshKey := rediskey.RefreshToken(sha1Hex(refreshToken))
    status, err := s.redis.Get(ctx, refreshKey).Result()

    if err != nil {
        return "", "", fmt.Errorf("refresh token: %w", ErrRTKNotExisted)
    }
    if status == rediskey.TokenStatusUsed {
        return "", "", ErrRTKUsed
    }

    // 标记已使用，保留 3 天防重放
    s.redis.Set(ctx, refreshKey, rediskey.TokenStatusUsed, 3*24*time.Hour)

    // 解析获取用户
    claims, _ := s.jwtHelper.Parse(refreshToken)
    var u User
    s.db.WithContext(ctx).First(&u, claims.UserID)

    return s.jwtHelper.GenerateTokenPair(&u)
}
```

### 1.4 Admin 中间件（双重校验）— `internal/middleware/admin.go`

> 仅 ecampus-crm 使用
> **重要**：最新代码要求**同时满足**：JWT 中 power 有管理员标志 **且** admin 表中存在该 userId 记录。

```go
// AdminCheck 需要注入 admin 表查询能力
func AdminCheck(db *gorm.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        claims := GetClaims(c)
        if claims == nil {
            result.Fail(c, result.CodeForbidden, "权限不足")
            c.Abort(); return
        }

        // 条件 1：JWT power 含管理员标志（power >= 2，按位判断）
        isAdminToken := claims.Power >= 2 // 0b10 = 管理员位

        // 条件 2：admin 表中存在该用户
        var count int64
        db.Model(&user.Admin{}).Where("userId = ?", claims.UserID).Count(&count)
        isAdminUser := count > 0

        if !isAdminToken || !isAdminUser {
            result.Fail(c, result.CodeForbidden, "权限不足")
            c.Abort(); return
        }
        c.Next()
    }
}
```

### 1.5 BlackList 中间件（用 rootUserId + 超时容错）— `internal/middleware/blacklist.go`

> 两个服务共享
> **注意**：最新代码使用 `rootUserId`（基座账号 ID）而非 `userId`，确保封禁基座号时所有子身份也被封。
> Redis 查询有 1 秒超时，超时或异常时**放行**（fail-open），避免 Redis 故障导致全站不可用。

```go
func BlackListCheck(rds *redis.Client) gin.HandlerFunc {
    return func(c *gin.Context) {
        if c.Request.Method == "OPTIONS" { c.Next(); return }

        claims := GetClaims(c)
        rootUserID := strconv.FormatInt(claims.RootUserID, 10)

        // 1 秒超时，超时放行
        ctx, cancel := context.WithTimeout(c.Request.Context(), 1*time.Second)
        defer cancel()

        blocked, err := rds.SIsMember(ctx, rediskey.GlobalBlacklist, rootUserID).Result()
        if err != nil {
            // Redis 异常，放行（fail-open）
            c.Next(); return
        }
        if blocked {
            result.Fail(c, result.CodeForbidden, "账号已被封禁")
            c.Abort(); return
        }
        c.Next()
    }
}
```

---

## 2. 微信登录 — `internal/user/service.go`

> ecampus 服务

```go
func (s *Service) WechatLogin(ctx context.Context, code string) (string, string, *User, error) {
    // 1. code 换 openid
    resp, err := s.wxClient.Jscode2Session(code)
    if err != nil { return "", "", nil, fmt.Errorf("wx jscode2session: %w", err) }

    // 2. 查找或创建用户
    var u User
    err = s.db.WithContext(ctx).Where("openId = ?", resp.OpenID).First(&u).Error
    if errors.Is(err, gorm.ErrRecordNotFound) {
        avatars := strings.Split(s.cfg.Custom.DefaultAvatar, ",")
        u = User{
            OpenID:      resp.OpenID,
            Nickname:    randomNickname(),
            Avatar:      avatars[rand.Intn(len(avatars))],
            Power:       0,
            AccountType: int(AccountTypeWechat),
        }
        s.db.WithContext(ctx).Create(&u)
        u.RootUserID = u.ID
        s.db.WithContext(ctx).Model(&u).Update("rootUserId", u.ID)
    } else if err != nil {
        return "", "", nil, fmt.Errorf("query user: %w", err)
    }

    // 3. 生成 token
    token, refreshToken, err := s.jwtHelper.GenerateTokenPair(&u)
    if err != nil { return "", "", nil, err }

    // 4. HyperLogLog 记录活跃
    today := time.Now().Format("20060102")
    s.redis.PFAdd(ctx, rediskey.ActiveDay(today), u.ID)

    return token, refreshToken, &u, nil
}
```

---

## 3. 帖子创建与内容审核

### 3.1 创建帖子 — `internal/topic/handler.go` + `internal/topic/service.go`

> ecampus 服务

```go
// handler.go
func (h *Handler) Create(c *gin.Context) {
    var req CreateTopicReq
    if err := c.ShouldBindJSON(&req); err != nil {
        result.Fail(c, result.CodeParamError, "参数错误"); return
    }
    claims := middleware.GetClaims(c)
    id, err := h.svc.Create(c.Request.Context(), claims, &req)
    if err != nil { result.HandleError(c, err); return }
    result.Success(c, id)
}

// service.go
func (s *Service) Create(ctx context.Context, claims *jwtutil.Claims, req *CreateTopicReq) (string, error) {
    topic := Topic{
        ThemeID:     req.ThemeID,
        UserID:      strconv.FormatInt(claims.UserID, 10),
        Title:       req.Title,
        Content:     req.Content,
        Imgs:        result.EnsureSlice(req.Imgs),
        HasCheck:    false, // 初始 false，审核通过后改 true
        AccountType: claims.AccountType,
        NickName:    req.NickName, // 从请求或缓存获取
        Avatar:      req.Avatar,
        Ext:         req.Ext,
    }

    res, err := s.coll().InsertOne(ctx, topic)
    if err != nil { return "", fmt.Errorf("insert topic: %w", err) }

    topicID := res.InsertedID.(primitive.ObjectID).Hex()

    // 发 MQ 异步审核
    s.producer.SendTopicCheck(TopicCheckMsg{TopicID: topicID})

    return topicID, nil
}
```

### 3.2 内容审核消费者 — `internal/mq/consumer.go`

> ecampus 服务运行的 MQ consumer

```go
func (c *Consumers) handleTopicCheck(body []byte) error {
    var msg TopicCheckMsg
    json.Unmarshal(body, &msg)

    // 查帖子（含未审核的）
    var t topic.Topic
    oid, _ := primitive.ObjectIDFromHex(msg.TopicID)
    err := c.mongoDB.Collection("campus_topic").FindOne(ctx, bson.M{"_id": oid}).Decode(&t)
    if err != nil { return nil } // 已删除

    // 微信内容安全检查
    titleResult := c.wxUtil.MsgSecCheck(t.Title, t.UserID)
    contentResult := c.wxUtil.MsgSecCheck(t.Content, t.UserID)

    if titleResult.Suggest == "risky" || contentResult.Suggest == "risky" {
        // 审核不通过 — hasCheck 保持 false
        c.wxUtil.SendSubscribeMsg(t.UserID, "您的帖子未通过审核", t.Title)
        return nil
    }

    // 审核通过：用过滤文本替换 + 设 hasCheck=true
    update := bson.M{"$set": bson.M{
        "hasCheck": true,
        "title":    coalesce(titleResult.FilteredContent, t.Title),
        "content":  coalesce(contentResult.FilteredContent, t.Content),
    }}
    c.mongoDB.Collection("campus_topic").UpdateByID(ctx, oid, update)

    // 发 MQ 添加搜索索引
    c.producer.SendAddTopicSearch(AddTopicSearchMsg{TopicID: msg.TopicID})

    // 记录指标
    metrics.PostPublishTotal.WithLabelValues("success").Inc()

    // 通知用户
    c.wxUtil.SendSubscribeMsg(t.UserID, "您的帖子已发布", t.Title)
    return nil
}
```

### 3.3 微信内容安全 — `internal/pkg/wxutil/security.go`

```go
func (w *Client) MsgSecCheck(content, openID string) *CheckResult {
    token := w.GetAccessToken()
    resp := httpPost("https://api.weixin.qq.com/wxa/msg_sec_check?access_token="+token, map[string]interface{}{
        "content": content, "version": 2, "scene": 1, "openid": openID,
    })
    return &CheckResult{
        Suggest:         resp.Result.Suggest,
        Label:           resp.Result.Label,
        FilteredContent: resp.Detail[0].FilteredContent,
    }
}
```

### 3.4 Access Token 管理 — `internal/pkg/wxutil/token.go`

```go
var (
    accessToken     string
    tokenExpireTime time.Time
    tokenMu         sync.Mutex
)

func (w *Client) GetAccessToken() string {
    if time.Now().Before(tokenExpireTime) && accessToken != "" {
        return accessToken
    }
    tokenMu.Lock()
    defer tokenMu.Unlock()
    if time.Now().Before(tokenExpireTime) && accessToken != "" {
        return accessToken
    }

    resp := httpPost("https://api.weixin.qq.com/cgi-bin/stable_token", map[string]interface{}{
        "grant_type": "client_credential",
        "appid":      w.cfg.AppID,
        "secret":     w.cfg.Secret,
    })

    accessToken = resp.AccessToken
    tokenExpireTime = time.Now().Add(time.Duration(resp.ExpiresIn-200) * time.Second)
    return accessToken
}
```

---

## 4. 评论创建与通知 — `internal/mq/consumer.go`

> ecampus 服务 MQ consumer

```go
func (c *Consumers) handleCommentAdd(body []byte) error {
    var msg AddCommentMsg
    json.Unmarshal(body, &msg)
    cmt := msg.Comment

    // 1. 内容安全
    result := c.wxUtil.MsgSecCheck(cmt.Comment, cmt.User.UserID)
    if result.Suggest == "risky" {
        c.wxUtil.SendSubscribeMsg(cmt.User.UserID, "您的评论未通过审核", cmt.Comment)
        return nil
    }

    // 2. 保存
    cmt.Comment = coalesce(result.FilteredContent, cmt.Comment)
    cmt.HasCheck = true
    cmt.CreatedTime = time.Now()
    c.mongoDB.Collection("campus_comment").InsertOne(ctx, cmt)

    // 3. 帖子 commentNum +1
    topicOID, _ := primitive.ObjectIDFromHex(cmt.TopicID)
    c.mongoDB.Collection("campus_topic").UpdateByID(ctx, topicOID,
        bson.M{"$inc": bson.M{"commentNum": 1}})

    // 4. 如果回复，父评论 commentNum +1
    if cmt.RootCmtID != "" {
        rootOID, _ := primitive.ObjectIDFromHex(cmt.RootCmtID)
        c.mongoDB.Collection("campus_comment").UpdateByID(ctx, rootOID,
            bson.M{"$inc": bson.M{"commentNum": 1}})
    }

    // 5. 通知帖子作者
    var topic topic.Topic
    c.mongoDB.Collection("campus_topic").FindOne(ctx, bson.M{"_id": topicOID}).Decode(&topic)
    if topic.UserID != cmt.User.UserID {
        c.producer.SendNotify(NotifyMsg{
            TargetUserID: topic.UserID, Type: "comment",
            Content: map[string]string{"topicId": cmt.TopicID, "comment": cmt.Comment},
        })
    }

    // 6. 通知被回复人
    if cmt.Parent != nil && cmt.Parent.UserID != cmt.User.UserID {
        c.producer.SendNotify(NotifyMsg{
            TargetUserID: cmt.Parent.UserID, Type: "comment",
            Content: map[string]string{"topicId": cmt.TopicID, "comment": cmt.Comment},
        })
    }

    metrics.CommentPublishTotal.WithLabelValues("success").Inc()
    return nil
}
```

---

## 5. RabbitMQ 基础架构 — `internal/mq/`

### 5.1 BaseProducer — `internal/mq/producer.go`

```go
type BaseProducer struct {
    ch       *amqp.Channel
    exchange string
    routeKey string
    rds      *redis.Client
}

func (p *BaseProducer) Send(ctx context.Context, data interface{}) error {
    uniqueID, _ := p.rds.Incr(ctx, rediskey.MQUUIDKey).Result()
    msg := MQMessage{UniqueID: uniqueID, Data: data}
    body, _ := json.Marshal(msg)

    return p.ch.PublishWithContext(ctx, p.exchange, p.routeKey, true, false,
        amqp.Publishing{
            ContentType:  "application/json",
            Body:         body,
            DeliveryMode: amqp.Persistent,
        })
}
```

### 5.2 BaseConsumer（去重+重试）— `internal/mq/base.go`

```go
func HandleWithDedup(
    rds *redis.Client,
    prefixKey string,
    delivery amqp.Delivery,
    handler func(body []byte) error,
    logger *zap.Logger,
) {
    var msg MQMessage
    json.Unmarshal(delivery.Body, &msg)
    rdsKey := prefixKey + strconv.FormatInt(msg.UniqueID, 10)
    ctx := context.Background()

    // 去重
    status, _ := rds.Get(ctx, rdsKey).Int()
    if status == MsgPost {
        delivery.Ack(false)
        return
    }

    // 重试 3 次
    var lastErr error
    for i := 0; i < 3; i++ {
        if err := handler(delivery.Body); err == nil {
            delivery.Ack(false)
            rds.Set(ctx, rdsKey, MsgPost, 3*24*time.Hour)
            return
        } else {
            lastErr = err
            logger.Error("consumer retry", zap.Int("attempt", i+1), zap.Error(err))
        }
    }

    logger.Error("consumer failed", zap.Error(lastErr))
    delivery.Nack(false, false)
}
```

### 5.3 Confirm 回调 — `internal/mq/producer.go`

```go
func SetupConfirmAndReturn(ch *amqp.Channel, mongoDB *mongo.Database) {
    ch.Confirm(false)
    confirms := ch.NotifyPublish(make(chan amqp.Confirmation, 100))
    returns := ch.NotifyReturn(make(chan amqp.Return, 100))

    go func() {
        for confirm := range confirms {
            if !confirm.Ack {
                mongoDB.Collection("campus_mq").InsertOne(ctx, mq.MQLog{
                    CreatedTime: time.Now(), Type: "to_broker_fail", Data: confirm,
                })
            }
        }
    }()

    go func() {
        for ret := range returns {
            mongoDB.Collection("campus_mq").InsertOne(ctx, mq.MQLog{
                CreatedTime: time.Now(), Type: "to_queue_fail", Data: string(ret.Body),
            })
        }
    }()
}
```

### 5.4 Consumer 启动 — `internal/mq/consumer.go`

```go
func (c *Consumers) Start() {
    handlers := map[string]func([]byte) error{
        QueueTopicCheck:        c.handleTopicCheck,
        QueueCommentAdd:        c.handleCommentAdd,
        QueueTopicSearchAdd:    c.handleTopicSearchAdd,
        QueueTopicSearchUpdate: c.handleTopicSearchUpdate,
        QueueTopicSearchDel:    c.handleTopicSearchDel,
        QueueTopicUpdate:       c.handleTopicUpdate,
        QueueTopicDelete:       c.handleTopicDelete,
        QueueCommentUpdate:     c.handleCommentUpdate,
        QueueCommentDelete:     c.handleCommentDelete,
        QueueGetCourse:         c.handleGetCourse,
        QueueNotifyUser:        c.handleNotifyUser,
    }

    for queue, handler := range handlers {
        msgs, _ := c.ch.Consume(queue, "", false, false, false, false, nil)
        h := handler // capture
        q := queue
        go func() {
            for msg := range msgs {
                HandleWithDedup(c.rds, dedupPrefix(q), msg, h, c.logger)
            }
        }()
    }
}
```

---

## 6. WebSocket 聊天 — `internal/chat/ws.go`

> 仅 ecampus 服务

### 6.1 连接与认证

```go
var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool { return true },
}

func (h *WSHandler) HandleUpgrade(c *gin.Context) {
    conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil { return }

    session := &Session{Conn: conn, LastActive: time.Now()}
    h.mgr.AddAnonymous(conn.RemoteAddr().String(), session)
    go h.readPump(session)
}

func (h *WSHandler) readPump(session *Session) {
    defer func() {
        h.mgr.Remove(session)
        session.Conn.Close()
    }()

    for {
        _, message, err := session.Conn.ReadMessage()
        if err != nil { break }

        var msg map[string]interface{}
        json.Unmarshal(message, &msg)

        if msg["type"] == "auth" {
            h.handleAuth(session, msg["token"].(string))
            continue
        }
        if session.UserID == "" {
            session.Conn.Close(); return
        }
        session.LastActive = time.Now()
        h.chatSvc.HandleMessage(session.UserID, message)
    }
}

func (h *WSHandler) handleAuth(session *Session, token string) {
    tokenKey := rediskey.Token(sha1Hex(token))
    if h.rds.Get(context.Background(), tokenKey).Err() != nil {
        session.Conn.WriteJSON(map[string]string{"type": "auth_fail", "reason": "token expired"})
        session.Conn.Close(); return
    }
    claims, err := h.jwtHelper.Parse(token)
    if err != nil {
        session.Conn.WriteJSON(map[string]string{"type": "auth_fail"})
        session.Conn.Close(); return
    }

    userID := strconv.FormatInt(claims.UserID, 10)
    session.UserID = userID
    h.mgr.BindUser(userID, session)
    session.Conn.WriteJSON(map[string]string{"type": "auth_success", "userId": userID})
}
```

### 6.2 心跳与超时

```go
// cron 每 30s
func (h *WSHandler) SendPing() {
    h.mgr.ForEachUser(func(userID string, s *Session) {
        if err := s.Conn.WriteMessage(websocket.PingMessage, []byte("heartbeat")); err != nil {
            h.mgr.RemoveUser(userID)
            s.Conn.Close()
        }
    })
}

// cron 每 10s
func (h *WSHandler) CheckTimeout() {
    now := time.Now()
    h.mgr.ForEach(func(key string, s *Session) {
        if now.Sub(s.LastActive) > 60*time.Second {
            s.Conn.WriteMessage(websocket.CloseMessage,
                websocket.FormatCloseMessage(websocket.CloseGoingAway, "heartbeat timeout"))
            s.Conn.Close()
            h.mgr.Remove(s)
        }
    })
}
```

### 6.3 通知推送 — `internal/mq/consumer.go`

```go
func (c *Consumers) handleNotifyUser(body []byte) error {
    var msg NotifyMsg
    json.Unmarshal(body, &msg)

    // 存 MongoDB
    notification := chat.Notification{
        UserID: msg.TargetUserID, Type: msg.Type,
        Content: msg.Content, CreatedAt: time.Now(), IsRead: false,
    }
    c.mongoDB.Collection("campus_notifications").InsertOne(ctx, notification)

    // 尝试 WebSocket 推送（仅 ecampus 有 session manager）
    if s, ok := c.sessionMgr.GetUser(msg.TargetUserID); ok {
        s.Conn.WriteJSON(map[string]interface{}{"type": "notification", "data": notification})
    }
    return nil
}
```

---

## 7. 文件上传 — `internal/file/service.go`

> ecampus 服务

```go
func (s *Service) Upload(ctx context.Context, file multipart.File, header *multipart.FileHeader, userID string) (string, string, error) {
    data, _ := io.ReadAll(file)
    md5Hash := md5.Sum(data)
    md5Str := hex.EncodeToString(md5Hash[:])

    // 已存在则 refCount++
    coll := s.mongoDB.Collection("campus_file")
    res := coll.FindOneAndUpdate(ctx,
        bson.M{"md5": md5Str},
        bson.M{"$inc": bson.M{"refCount": 1}})
    if res.Err() == nil {
        return md5Str, s.cfg.COS.BaseCDN + md5Str, nil
    }

    // 上传 COS（优先万象 CI 压缩）
    url, err := s.cosClient.PutWithImageProcess(ctx, md5Str, data, s.cfg.COS.Compress)
    if err != nil {
        // 降级：普通上传
        url, err = s.cosClient.Put(ctx, md5Str, data, header.Header.Get("Content-Type"))
        if err != nil { return "", "", fmt.Errorf("cos upload: %w", err) }
    }

    // 保存记录
    coll.InsertOne(ctx, File{MD5: md5Str, IsPublic: false, UserID: userID, RefCount: 1})
    return md5Str, url, nil
}
```

---

## 8. 推荐排行算法 — `internal/cron/suggest.go`

> ecampus 服务 cron，每天 02:01

### 8.1 生成排行

```go
func (c *SuggestJob) Generate() {
    ctx := context.Background()

    // 1. 获取参与推荐的主题
    var themes []theme.Theme
    cursor, _ := c.mongoDB.Collection("campus_theme").Find(ctx, bson.M{"needSuggest": true})
    cursor.All(ctx, &themes)

    // 2. 为每个主题生成排行
    for _, th := range themes {
        var topics []topic.Topic
        cur, _ := c.mongoDB.Collection("campus_topic").Find(ctx,
            bson.M{"themeId": th.ID.Hex(), "hasCheck": true})
        cur.All(ctx, &topics)

        rankKey := rediskey.SuggestRank(th.SuggestSetName)
        today := time.Now()

        for _, t := range topics {
            score := float64(th.SuggestBasicScore)
            days := today.Sub(t.ID.Timestamp()).Hours() / 24
            score -= days * 5  // 时间衰减
            score += float64(t.CommentNum) * 10
            score += float64(t.LikeNum) * 10
            score += float64(t.VisitedNum) * 3

            c.rds.ZAdd(ctx, rankKey, redis.Z{Score: score, Member: t.ID.Hex()})
        }
        // 只保留 top N
        c.rds.ZRemRangeByRank(ctx, rankKey, 0, int64(-(th.SuggestNumber + 1)))
    }

    // 3. 合并排行
    version, _ := c.rds.Incr(ctx, rediskey.SuggestCountKey).Result()
    newKey := fmt.Sprintf("rank:all_%d_%s", version, time.Now().Format("2006-01-02"))

    var allKeys []string
    for _, th := range themes {
        allKeys = append(allKeys, rediskey.SuggestRank(th.SuggestSetName))
    }
    c.rds.ZUnionStore(ctx, newKey, &redis.ZStore{Keys: allKeys})

    // 4. 轮换 key
    prevKey, _ := c.rds.Get(ctx, rediskey.SuggestCurKey).Result()
    c.rds.Set(ctx, rediskey.SuggestPrevKey, prevKey, 0)
    c.rds.Set(ctx, rediskey.SuggestCurKey, newKey, 0)

    // 5. 清分页缓存
    c.rds.Del(ctx, rediskey.SuggestTopicListKey)
}
```

### 8.2 获取推荐列表 — `internal/topic/service.go`

```go
func (s *Service) GetSuggestList(ctx context.Context, userID string, page, size int) (*SuggestListVO, error) {
    cacheKey := fmt.Sprintf("%d_%d", page, size)
    cached, err := s.redis.HGet(ctx, rediskey.SuggestTopicListKey, cacheKey).Result()
    if err == nil {
        var vo SuggestListVO
        json.Unmarshal([]byte(cached), &vo)
        return &vo, nil
    }

    curKey, _ := s.redis.Get(ctx, rediskey.SuggestCurKey).Result()
    if curKey == "" { curKey, _ = s.redis.Get(ctx, rediskey.SuggestPrevKey).Result() }
    if curKey == "" { return nil, nil }

    total, _ := s.redis.ZCard(ctx, curKey).Result()
    start := int64((page - 1) * size)
    topicIDs, _ := s.redis.ZRevRange(ctx, curKey, start, start+int64(size)-1).Result()

    topics := s.findByIDs(ctx, topicIDs)
    s.fillLikeAndCollection(ctx, userID, topics)

    vo := &SuggestListVO{Total: total, CurPage: page, Size: size, Data: topics}
    data, _ := json.Marshal(vo)
    s.redis.HSet(ctx, rediskey.SuggestTopicListKey, cacheKey, string(data))
    return vo, nil
}
```

---

## 9. 教务系统对接 — `internal/school/jw.go`

> ecampus 服务（通过 MQ consumer 异步执行）

### 9.1 登录教务

```go
func (j *JWClient) Login(stuNum, stuPwd string) ([]*http.Cookie, error) {
    // DES/ECB/PKCS5 加密密码
    key := []byte("PassB01I")[:8]
    encrypted := desECBEncrypt([]byte(stuPwd), key)
    encoded := base64.StdEncoding.EncodeToString(encrypted)

    jar, _ := cookiejar.New(nil)
    client := &http.Client{
        Jar:           jar,
        CheckRedirect: func(req *http.Request, via []*http.Request) error {
            return http.ErrUseLastResponse
        },
    }

    form := url.Values{
        "username": {stuNum},
        "password": {encoded},
    }

    req, _ := http.NewRequest("POST",
        "https://auth.sztu.edu.cn/idp/authcenter/ActionAuthChain", strings.NewReader(form.Encode()))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    req.Header.Set("X-Requested-With", "XMLHttpRequest")

    resp, err := client.Do(req)
    if err != nil { return nil, fmt.Errorf("jw login: %w", err) }
    defer resp.Body.Close()

    // 跟随重定向链获取教务 cookie
    // ...
    return jar.Cookies(jwURL), nil
}
```

### 9.2 课程抓取（MQ consumer）— `internal/mq/consumer.go`

```go
func (c *Consumers) handleGetCourse(body []byte) error {
    var msg CourseMsg
    json.Unmarshal(body, &msg)

    // 更新状态 → 抓取中
    c.db.Model(&school.UserCourse{}).
        Where("userId = ? AND term = ? AND week = ?", msg.UserID, msg.Term, msg.Week).
        Update("status", 1)

    cookies, err := c.jwClient.Login(msg.StuNum, msg.StuPwd)
    if err != nil {
        c.db.Model(&school.UserCourse{}).
            Where("userId = ? AND term = ? AND week = ?", msg.UserID, msg.Term, msg.Week).
            Update("status", 3) // 失败
        return err
    }

    courseData := c.jwClient.GetCourse(cookies, msg.Term, msg.Week)
    courseJSON, _ := json.Marshal(courseData)

    c.db.Where("userId = ? AND term = ? AND week = ?", msg.UserID, msg.Term, msg.Week).
        Assign(school.UserCourse{Course: string(courseJSON), Status: 2}).
        FirstOrCreate(&school.UserCourse{UserID: msg.UserID, Term: msg.Term, Week: msg.Week})

    return nil
}
```

---

## 10. 签到与经验 — `internal/level/service.go`

> ecampus 服务

```go
func (s *Service) SignIn(ctx context.Context, userID int64) error {
    yearMonth := time.Now().Format("200601")
    day := int64(time.Now().Day())
    key := rediskey.UserSign(yearMonth)
    offset := userID*31 + day

    already, _ := s.redis.GetBit(ctx, key, offset).Result()
    if already == 1 { return ErrAlreadySigned }

    s.redis.SetBit(ctx, key, offset, 1)

    // 经验写入 Redis List（cron 批量入库）
    detail := ExpDetail{UserID: userID, GetExpDate: time.Now(), GetExp: 10}
    data, _ := json.Marshal(detail)
    s.redis.LPush(ctx, rediskey.ExpDetailKey, string(data))

    // 清经验缓存
    s.redis.Del(ctx, rediskey.UserExp(userID))

    return nil
}

func (s *Service) GetSignDetail(ctx context.Context, userID int64) (*SignDetailVO, error) {
    yearMonth := time.Now().Format("200601")
    key := rediskey.UserSign(yearMonth)
    day := time.Now().Day()

    todaySigned, _ := s.redis.GetBit(ctx, key, userID*31+int64(day)).Result()

    signDays := 0
    for d := 1; d <= day; d++ {
        bit, _ := s.redis.GetBit(ctx, key, userID*31+int64(d)).Result()
        if bit == 1 { signDays++ }
    }

    exp := s.getExpWithCache(ctx, userID)
    return &SignDetailVO{Exp: exp, SignDays: signDays, TodaySigned: todaySigned == 1}, nil
}
```

---

## 11. 搜索索引维护 — `internal/mq/consumer.go`

> ecampus 服务 MQ consumer

```go
func (c *Consumers) handleTopicSearchAdd(body []byte) error {
    var msg AddTopicSearchMsg
    json.Unmarshal(body, &msg)

    // 查帖子和主题
    oid, _ := primitive.ObjectIDFromHex(msg.TopicID)
    var t topic.Topic
    c.mongoDB.Collection("campus_topic").FindOne(ctx, bson.M{"_id": oid}).Decode(&t)

    var th theme.Theme
    thOID, _ := primitive.ObjectIDFromHex(t.ThemeID)
    c.mongoDB.Collection("campus_theme").FindOne(ctx, bson.M{"_id": thOID}).Decode(&th)

    if !th.NeedSearch { return nil }

    // gse 分词
    titleTokens := c.segmenter.CutSearch(t.Title)
    contentTokens := c.segmenter.CutSearch(t.Content)

    search := topic.TopicSearch{
        TopicID:   t.ID.Hex(),
        ThemeName: th.Name,
        Title:     strings.Join(titleTokens, " "),
        Content:   strings.Join(contentTokens, " "),
    }
    c.mongoDB.Collection("campus_topic_search").InsertOne(ctx, search)
    return nil
}
```

### 文本搜索 — `internal/topic/search.go`

```go
func (s *Service) SearchByKeyword(ctx context.Context, keyword, themeName string, page, size int) ([]string, int64, error) {
    tokenized := s.segmenter.CutSearch(keyword)
    searchStr := strings.Join(tokenized, " ")

    filter := bson.M{"$text": bson.M{"$search": searchStr}}
    if themeName != "" { filter["themeName"] = themeName }

    coll := s.mongoDB.Collection("campus_topic_search")
    total, _ := coll.CountDocuments(ctx, filter)

    opts := options.Find().
        SetProjection(bson.M{"score": bson.M{"$meta": "textScore"}, "topicId": 1}).
        SetSort(bson.M{"score": bson.M{"$meta": "textScore"}}).
        SetSkip(int64((page - 1) * size)).SetLimit(int64(size))

    cursor, _ := coll.Find(ctx, filter, opts)
    var results []topic.TopicSearch
    cursor.All(ctx, &results)

    ids := make([]string, len(results))
    for i, r := range results { ids[i] = r.TopicID }
    return ids, total, nil
}
```

---

## 12. Prometheus 指标 — `internal/cron/metrics.go`

> ecampus 服务 cron，每 60s

```go
var (
    postPublishTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{Name: "campus_post_publish_total"}, []string{"result"})
    commentPublishTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{Name: "campus_comment_publish_total"}, []string{"result"})
    activeUsersGauge = promauto.NewGaugeVec(
        prometheus.GaugeOpts{Name: "campus_active_users"}, []string{"window"})
)

func RefreshActiveUserGauges(rds *redis.Client) {
    ctx := context.Background()
    now := time.Now()

    // DAU
    dau, _ := rds.PFCount(ctx, rediskey.ActiveDay(now.Format("20060102"))).Result()
    activeUsersGauge.WithLabelValues("dau").Set(float64(dau))

    // WAU
    var weekKeys []string
    for i := 0; i < 7; i++ {
        weekKeys = append(weekKeys, rediskey.ActiveDay(now.AddDate(0, 0, -i).Format("20060102")))
    }
    wau, _ := rds.PFCount(ctx, weekKeys...).Result()
    activeUsersGauge.WithLabelValues("wau").Set(float64(wau))

    // MAU
    var monthKeys []string
    for i := 0; i < 30; i++ {
        monthKeys = append(monthKeys, rediskey.ActiveDay(now.AddDate(0, 0, -i).Format("20060102")))
    }
    mau, _ := rds.PFCount(ctx, monthKeys...).Result()
    activeUsersGauge.WithLabelValues("mau").Set(float64(mau))
}
```

---

## 13. 埋点批量入库 — `internal/cron/event_flush.go`

> ecampus 服务 cron，每 15 分钟

```go
func (j *EventFlushJob) Run() {
    ctx := context.Background()
    length, _ := j.rds.LLen(ctx, rediskey.EventKey).Result()
    if length == 0 { return }

    items, _ := j.rds.LRange(ctx, rediskey.EventKey, 0, -1).Result()

    // 批量入库（每 1000 条）
    for i := 0; i < len(items); i += 1000 {
        end := i + 1000
        if end > len(items) { end = len(items) }

        var events []event.Event
        for _, raw := range items[i:end] {
            var e event.Event
            json.Unmarshal([]byte(raw), &e)
            events = append(events, e)
        }
        j.db.WithContext(ctx).CreateInBatches(events, 1000)
    }

    j.rds.Del(ctx, rediskey.EventKey)
}
```

---

## 14. 零值序列化兼容

Go 默认零值与 FastJSON 一致。唯一注意 slice：

```go
// internal/pkg/result/util.go
func EnsureSlice[T any](s []T) []T {
    if s == nil { return []T{} }
    return s
}

// 所有返回 slice 的地方必须使用：
topics := result.EnsureSlice(topicList)
```

---

## 15. 管理员登录（含二级密码+锁定+旧管理员迁移）— `internal/user/service.go`

> ecampus-crm 服务
> **最新逻辑**：管理员登录比普通登录复杂，包含三层校验和旧数据迁移。

```go
// admin.go — handler
func (h *AdminHandler) Login(c *gin.Context) {
    var req AdminLoginReq
    if err := c.ShouldBindJSON(&req); err != nil {
        result.Fail(c, result.CodeParamError, "参数错误"); return
    }
    token, refreshToken, u, err := h.svc.AdminLogin(c.Request.Context(), &req)
    if err != nil { result.HandleError(c, err); return }
    result.Success(c, gin.H{"token": token, "refreshToken": refreshToken, "user": u})
}

// service.go — 完整管理员登录流程
func (s *Service) AdminLogin(ctx context.Context, req *AdminLoginReq) (string, string, *User, error) {
    lockKey := "admin:login:lock:" + req.Username
    failCountKey := "admin:login:fail:count:" + req.Username

    // 1. 检查账号是否被锁定（10 次失败后锁 24 小时）
    if s.redis.Exists(ctx, lockKey).Val() > 0 {
        return "", "", nil, errors.New("账号已锁定，请明天后再试")
    }

    // 2. 校验二级密码（硬编码安全码，防止接口被扫）
    if req.SecondaryPassword != s.cfg.Admin.SecondaryPassword {
        remaining := s.handleLoginFail(ctx, failCountKey, lockKey)
        return "", "", nil, fmt.Errorf("二级密码错误，今日还有 %d 次机会", remaining)
    }

    // 3. 查 admin 表
    var admin Admin
    err := s.db.WithContext(ctx).Where("username = ?", req.Username).First(&admin).Error

    if errors.Is(err, gorm.ErrRecordNotFound) {
        // 3a. 旧管理员迁移：查 user 表中 power >= 8 且学号密码匹配的用户
        legacyUser, err := s.loadLegacyAdmin(ctx, req.Username, req.Password)
        if err != nil || legacyUser == nil {
            remaining := s.handleLoginFail(ctx, failCountKey, lockKey)
            return "", "", nil, fmt.Errorf("账号或密码错误，今日还有 %d 次机会", remaining)
        }
        // 自动迁移到 admin 表
        admin = s.migrateLegacyAdmin(ctx, legacyUser, req.Username, req.Password)
    }

    // 4. 校验密码（MD5）
    if admin.Password != md5Hex(req.Password) {
        remaining := s.handleLoginFail(ctx, failCountKey, lockKey)
        return "", "", nil, fmt.Errorf("账号或密码错误，今日还有 %d 次机会", remaining)
    }

    // 5. 加载关联用户
    var u User
    if err := s.db.WithContext(ctx).First(&u, admin.UserID).Error; err != nil {
        return "", "", nil, errors.New("管理员关联用户不存在")
    }

    // 6. 清除失败计数，生成 token
    s.redis.Del(ctx, failCountKey)
    u.Power = admin.Power // 使用 admin 表的 power
    u.StuPwd = ""         // 脱敏
    token, refreshToken, err := s.jwtHelper.GenerateTokenPair(&u)
    return token, refreshToken, &u, err
}

// 登录失败计数 + 锁定
func (s *Service) handleLoginFail(ctx context.Context, failCountKey, lockKey string) int {
    count, _ := s.redis.Incr(ctx, failCountKey).Result()
    if count == 1 {
        s.redis.Expire(ctx, failCountKey, 24*time.Hour)
    }
    if count >= 10 {
        s.redis.Set(ctx, lockKey, "locked", 24*time.Hour)
    }
    return int(10 - count)
}

// 旧管理员查询（user 表中 power >= 8 的用户）
func (s *Service) loadLegacyAdmin(ctx context.Context, stuNum, rawPwd string) (*User, error) {
    encPwd, _ := encrypt.AESEncrypt(rawPwd, s.cfg.Encryption.Key)
    var u User
    err := s.db.WithContext(ctx).
        Where("stuNum = ? AND stuPwd = ? AND power >= 8", stuNum, encPwd).
        First(&u).Error
    if errors.Is(err, gorm.ErrRecordNotFound) { return nil, nil }
    return &u, err
}

// 迁移旧管理员到 admin 表
func (s *Service) migrateLegacyAdmin(ctx context.Context, u *User, username, rawPwd string) Admin {
    admin := Admin{
        UserID:   u.ID,
        Username: username,
        Password: md5Hex(rawPwd),
        Power:    resolveAdminPower(u.Power),
    }
    s.db.WithContext(ctx).Create(&admin)
    return admin
}
```

### Redis Keys（管理员登录相关）

| Key | 类型 | TTL | 说明 |
|-----|------|-----|------|
| `admin:login:lock:{username}` | String | 24h | 登录锁定标记 |
| `admin:login:fail:count:{username}` | String(int) | 24h | 失败次数计数器 |

---

## 16. Admin CRUD 模式（通用参考）

> ecampus-crm 的 admin handler 遵循统一模式

```go
// internal/other/notice_admin.go — 典型 admin CRUD handler
type NoticeAdminHandler struct{ svc *Service }

func (h *NoticeAdminHandler) Add(c *gin.Context) {
    var notice Notice
    if err := c.ShouldBindJSON(&notice); err != nil {
        result.Fail(c, result.CodeParamError, "参数错误"); return
    }
    notice.CreatedBy = middleware.GetUserID(c)
    if err := h.svc.CreateNotice(c.Request.Context(), &notice); err != nil {
        result.HandleError(c, err); return
    }
    result.Success(c, nil)
}

func (h *NoticeAdminHandler) Delete(c *gin.Context) {
    id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
    if err := h.svc.DeleteNotice(c.Request.Context(), id); err != nil {
        result.HandleError(c, err); return
    }
    result.Success(c, nil)
}

func (h *NoticeAdminHandler) List(c *gin.Context) {
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    size, _ := strconv.Atoi(c.DefaultQuery("size", "15"))
    pageResult, err := h.svc.ListNotices(c.Request.Context(), page, size)
    if err != nil { result.HandleError(c, err); return }
    result.Success(c, pageResult)
}

// service.go 中对应的方法
func (s *Service) CreateNotice(ctx context.Context, n *Notice) error {
    return s.db.WithContext(ctx).Create(n).Error
}

func (s *Service) DeleteNotice(ctx context.Context, id int64) error {
    return s.db.WithContext(ctx).Delete(&Notice{}, id).Error
}

func (s *Service) ListNotices(ctx context.Context, page, size int) (*result.PageResult[Notice], error) {
    var total int64
    var list []Notice
    s.db.WithContext(ctx).Model(&Notice{}).Count(&total)
    s.db.WithContext(ctx).Offset((page-1)*size).Limit(size).Order("createdAt DESC").Find(&list)
    return result.NewPage(list, total, page, size), nil
}
```

---

## 17. 帖子搜索热度排序 — `internal/topic/service.go`

> ecampus 服务，`GET /api/topic/search?orderBy=hot`

```go
func (s *Service) SearchHot(ctx context.Context, themeID string, page, size int) ([]Topic, int64, error) {
    match := bson.M{"hasCheck": true}
    if themeID != "" { match["themeId"] = themeID }

    // 7 天内的帖子优先
    sevenDaysAgo := primitive.NewObjectIDFromTimestamp(time.Now().AddDate(0, 0, -7))

    pipeline := mongo.Pipeline{
        {{Key: "$match", Value: match}},
        {{Key: "$addFields", Value: bson.M{
            "hotScore": bson.M{"$add": []interface{}{
                bson.M{"$multiply": []interface{}{"$commentNum", 9}},
                bson.M{"$multiply": []interface{}{"$likeNum", 6}},
                bson.M{"$multiply": []interface{}{"$visitedNum", 1}},
            }},
            "isRecent": bson.M{"$gte": []interface{}{"$_id", sevenDaysAgo}},
        }}},
        {{Key: "$sort", Value: bson.D{
            {Key: "isRecent", Value: -1},
            {Key: "hotScore", Value: -1},
            {Key: "_id", Value: -1},
        }}},
        {{Key: "$skip", Value: int64((page - 1) * size)}},
        {{Key: "$limit", Value: int64(size)}},
    }

    cursor, err := s.coll().Aggregate(ctx, pipeline)
    if err != nil { return nil, 0, fmt.Errorf("aggregate hot: %w", err) }

    var topics []Topic
    cursor.All(ctx, &topics)

    // 总数（不含 skip/limit）
    countPipeline := mongo.Pipeline{
        {{Key: "$match", Value: match}},
        {{Key: "$count", Value: "total"}},
    }
    countCursor, _ := s.coll().Aggregate(ctx, countPipeline)
    var countResult []bson.M
    countCursor.All(ctx, &countResult)
    var total int64
    if len(countResult) > 0 { total = countResult[0]["total"].(int64) }

    return topics, total, nil
}
```

---

## Cron 任务汇总

> 全部由 ecampus 服务运行

| 任务 | 位置 | 周期 | 功能 |
|------|------|------|------|
| 推荐排行生成 | `cron/suggest.go` | 每天 02:01 | 计算帖子热度，生成 Redis sorted set |
| 旧排行清理 | `cron/suggest.go` | 每天 02:02 | SCAN 删除 `rank:all_*` 旧 key |
| 埋点批量入库 | `cron/event_flush.go` | 每 15 分钟 | Redis List → MySQL batch insert |
| 活跃指标刷新 | `cron/metrics.go` | 每 60 秒 | HyperLogLog → Prometheus gauge |
| WebSocket Ping | `chat/ws.go` | 每 30 秒 | 发送心跳 |
| WebSocket 超时检查 | `chat/ws.go` | 每 10 秒 | 关闭 60s 无活动连接 |
| 经验明细入库 | `cron/exp_flush.go` | 每 5 分钟 | Redis List → MySQL batch insert |
