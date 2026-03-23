# API 接口规范

> 所有接口的路径、HTTP 方法、请求参数、响应格式必须与 Java 版完全一致。
>
> 统一响应：`{"success": bool, "code": int, "msg": string, "data": any}`
>
> 认证方式：请求头 `Authorization: <jwt_token>`
>
> 本文档按服务分组：
> - **Part A** — ecampus（用户端，端口 8080）
> - **Part B** — ecampus-crm（管理后台，端口 8081）

---

## 路由认证规则

### ecampus 中间件链

| 路由组 | 中间件 | 说明 |
|--------|--------|------|
| 公开路径 | 无 | 免登录（login、refresh、公开数据） |
| `/api/**` | JWT → BlackList → Log | 用户级 |
| `/file/upload/**`、`/file/del/**` | JWT | 文件操作 |
| `/chat` (WebSocket) | 连接后发 auth 消息 | WebSocket 认证 |

**ecampus 排除 JWT 的路径**：
- `POST /api/user/login`
- `POST /api/user/refresh`
- `PUT /api/user/pre_authentication`
- `POST /api/user/official/login`
- `POST /api/user/official/certification`
- `GET /api/ad/**`
- `GET /api/user/nickname/random`

### ecampus-crm 中间件链

| 路由组 | 中间件 | 说明 |
|--------|--------|------|
| `/admin/user/login` | 无 | Admin 登录免认证 |
| `/admin/**` | JWT → BlackList → Log → AdminCheck | 管理员级 |

---

# Part A — ecampus（用户端）

> handler 文件：各域包的 `handler.go`
> 端口：8080

## A1. User 模块 — `internal/user/handler.go`

### 公开接口

| 方法 | 路径 | 说明 | 请求 | 响应 |
|------|------|------|------|------|
| POST | `/api/user/login` | 微信登录 | `{"code":"wx_code"}` | `Result<{token, refreshToken, user}>` |
| POST | `/api/user/refresh` | 刷新 token | `{"refreshToken":"xxx"}` | `Result<{token, refreshToken}>` |
| PUT | `/api/user/pre_authentication` | 学生预认证 | `{"stuNum":"xxx","stuPwd":"xxx"}` | `Result<Void>` |
| POST | `/api/user/official/login` | 官方号登录 | `{"username":"xxx","password":"xxx"}` | `Result<{token, refreshToken, user}>` |
| POST | `/api/user/official/certification` | 提交官方认证 | `{"name":"xxx","reason":"xxx"}` | `Result<Void>` |
| GET | `/api/user/nickname/random` | 随机昵称 | - | `Result<string>` |

### 需 JWT 接口

| 方法 | 路径 | 说明 | 请求 | 响应 |
|------|------|------|------|------|
| GET | `/api/user` | 当前用户信息 | - | `Result<User>` |
| PUT | `/api/user` | 编辑用户 | `{nickname, avatar, tag, gender}` | `Result<Void>` |
| POST | `/api/user/authentication` | 绑定教务 | `{stuNum, stuPwd}` | `Result<Void>` |
| POST | `/api/user/re_authentication` | 重新绑定 | `{stuNum, stuPwd}` | `Result<Void>` |
| POST | `/api/user/del_authentication` | 解绑教务 | - | `Result<Void>` |
| POST | `/api/user/check_login` | 教务登录检查 | - | `Result<bool>` |
| POST | `/api/user/get_course_by_weeks` | 多周课表 | `{weeks:[1,2], term:"xxx"}` | `Result<[...]>` |
| POST | `/api/user/get_exam` | 考试安排 | - | `Result<[...]>` |
| POST | `/api/user/get_exam_score` | 考试成绩 | - | `Result<[...]>` |
| GET | `/api/user/user_profile` | 目标用户主页 | `?targetUserId={id}` | `Result<UserProfile>` |
| POST | `/api/user/identity/anonymous` | 创建匿名 | `{nickname}` | `Result<User>` |
| PUT | `/api/user/identity/anonymous/nickname` | 改匿名昵称 | `{nickname}` | `Result<Void>` |
| GET | `/api/user/identity/list` | 身份列表 | - | `Result<[User]>` |
| POST | `/api/user/identity/switch` | 切换身份 | `{targetUserId}` | `Result<{token, refreshToken}>` |
| POST | `/api/user/follow` | 关注 | `{targetUserId}` | `Result<Void>` |
| DELETE | `/api/user/follow` | 取消关注 | `?targetUserId={id}` | `Result<Void>` |
| GET | `/api/user/followers` | 粉丝列表 | `?page&size` | `Result<CusPage<User>>` |
| GET | `/api/user/followings` | 关注列表 | `?page&size` | `Result<CusPage<User>>` |
| GET | `/api/user/stats` | 用户统计 | `?targetUserId={id}` | `Result<{followerCount, followingCount, likeCount}>` |
| GET | `/api/user/is_following` | 是否关注 | `?targetUserId={id}` | `Result<bool>` |

## A2. Topic 模块 — `internal/topic/handler.go`

| 方法 | 路径 | 说明 | 请求 | 响应 |
|------|------|------|------|------|
| POST | `/api/topic` | 创建帖子 | `{themeId, title, content, imgs, ext}` | `Result<string>` |
| DELETE | `/api/topic/:id` | 删除帖子 | - | `Result<Void>` |
| GET | `/api/topic/:topic_id` | 帖子详情 | - | `Result<Topic>` |
| PUT | `/api/topic/:topic_id` | 更新帖子 | `{title, content, imgs}` | `Result<Void>` |
| GET | `/api/topic/search` | 搜索帖子 | `?themeId&keyword&page&size&orderBy` | `Result<CusPage<Topic>>` |
| GET | `/api/topic` | 我的帖子 | `?page&size` | `Result<CusPage<Topic>>` |
| GET | `/api/topic/theme` | 主题下我的帖子 | `?themeId&page&size` | `Result<CusPage<Topic>>` |
| GET | `/api/topic/target_user_topics` | 目标用户帖子 | `?targetUserId&page&size` | `Result<CusPage<Topic>>` |
| GET | `/api/topic/follow_topics` | 关注人帖子流 | `?page&size` | `Result<CusPage<Topic>>` |

**创建帖子特殊逻辑**：
- 需要 `middleware.RequireVerified` 检查（认证+商户权限）
- 创建后 `hasCheck=false`，通过 MQ `campus.topic_check` 异步审核
- 搜索 `orderBy=hot` 用聚合管道：`hotScore = commentNum*9 + likeNum*6 + visitedNum*1`

## A3. Comment 模块 — `internal/comment/handler.go`

| 方法 | 路径 | 说明 | 请求 | 响应 |
|------|------|------|------|------|
| POST | `/api/comment/:topic_id` | 发表评论 | `{comment, parentCmtId?, rootCmtId?}` | `Result<Void>` |
| DELETE | `/api/comment/:topic_id/:comment_id` | 删除评论 | - | `Result<Void>` |
| GET | `/api/comment/:topic_id` | 帖子评论列表 | `?page&size` | `Result<CusPage<Comment>>` |
| GET | `/api/comment` | 我的评论 | `?page&size` | `Result<CusPage<Comment>>` |
| GET | `/api/comment/target_user_comments` | 目标用户评论 | `?targetUserId&page&size` | `Result<CusPage<Comment>>` |

**发表评论**：通过 MQ `campus.comment_add` 异步审核。

## A4. Like 模块 — `internal/topic/handler.go`（帖子点赞）+ `internal/comment/handler.go`（评论点赞）

| 方法 | 路径 | Handler 位置 | 说明 | 响应 |
|------|------|-------------|------|------|
| POST | `/api/like/topic/:topic_id` | topic/handler.go | 点赞帖子 | `Result<Void>` |
| DELETE | `/api/like/topic/:topic_id` | topic/handler.go | 取消点赞 | `Result<Void>` |
| GET | `/api/like/topic` | topic/handler.go | 我点赞的帖子 `?page&size` | `Result<CusPage<Topic>>` |
| POST | `/api/comment_like/:comment_id` | comment/handler.go | 点赞评论 | `Result<Void>` |
| DELETE | `/api/comment_like/:comment_id` | comment/handler.go | 取消评论点赞 | `Result<Void>` |

## A5. Collection 模块 — `internal/topic/handler.go`

| 方法 | 路径 | 说明 | 响应 |
|------|------|------|------|
| POST | `/api/collection/topic/:topic_id` | 收藏帖子 | `Result<Void>` |
| DELETE | `/api/collection/topic/:topic_id` | 取消收藏 | `Result<Void>` |
| GET | `/api/collection/collection_topics` | 我收藏的帖子 `?page&size` | `Result<CusPage<Topic>>` |

## A6. Theme 模块 — `internal/theme/handler.go`

公开接口（无需 JWT）：

| 方法 | 路径 | 说明 | 响应 |
|------|------|------|------|
| POST | `/api/theme/campus/init` | 初始化校园主题 | `Result<Void>` |
| GET | `/api/theme/campus` | 所有校园主题 | `Result<[Theme]>` |

## A7. File 模块 — `internal/file/handler.go`

| 方法 | 路径 | Auth | 说明 | 请求 | 响应 |
|------|------|------|------|------|------|
| GET | `/file/:md5` | 无 | 下载文件 | - | 文件内容/重定向 COS |
| GET | `/file` | 无 | 公开图片列表 | `?page&size` | `Result<CusPage<File>>` |
| POST | `/file/upload` | JWT | 上传文件 | `multipart/form-data` (field: file) | `Result<{md5, url}>` |
| DELETE | `/file/del/:md5` | JWT | 删除文件 | - | `Result<Void>` |

单文件最大 15MB。

## A8. Chat 模块

### REST 接口 — `internal/chat/handler.go`

| 方法 | 路径 | 说明 | 请求 | 响应 |
|------|------|------|------|------|
| GET | `/api/conversation` | 会话列表 | - | `Result<[Conversation]>` |
| PUT | `/api/conversation/conversation_enter` | 进入会话 | `{conversationId}` | `Result<Void>` |
| GET | `/api/conversation/:id/unread_count` | 未读消息数 | - | `Result<int>` |
| GET | `/api/conversation/conversation_query` | 查询会话 | `?targetUserId` | `Result<Conversation>` |
| GET | `/api/conversation/profile_by_conversation_id` | 对方信息 | `?conversationId` | `Result<User>` |
| DELETE | `/api/conversation/:id` | 删除会话 | - | `Result<Void>` |
| GET | `/api/message/:last_message_id` | 拉取离线消息 | - | `Result<[Message]>` |
| GET | `/api/message/history_messages` | 历史消息 | `?conversationId&page&size` | `Result<CusPage<Message>>` |
| GET | `/api/message/unread_messages` | 未读消息 | - | `Result<[Message]>` |
| GET | `/api/notify` | 通知列表 | `?type&page&size` | `Result<CusPage<Notification>>` |
| GET | `/api/notify/:type/haveUnread` | 有未读通知 | - | `Result<bool>` |
| GET | `/api/notify/:type` | 最新通知 | - | `Result<Notification>` |

### WebSocket — `internal/chat/ws.go`

| 路径 | 协议 | 说明 |
|------|------|------|
| `/chat` | WebSocket | 实时聊天 |

**连接流程**：
1. 客户端连接 `ws://host/chat`
2. 服务端记录匿名 session
3. 客户端发送 `{"type":"auth","token":"jwt_token"}`
4. 服务端验证 JWT → 绑定 userID → 返回 `{"type":"auth_success","userId":"xxx"}`
5. 后续消息由 `chatService.HandleMessage` 处理

**心跳**：服务端每 30s 发 Ping，客户端回 Pong，60s 无活动关闭。

## A9. Level 模块 — `internal/level/handler.go`

| 方法 | 路径 | 说明 | 响应 |
|------|------|------|------|
| GET | `/api/getUserSignDetail` | 签到详情 | `Result<{exp, signDays, todaySigned}>` |
| POST | `/api/sign_in` | 签到 | `Result<Void>` |
| GET | `/api/UserExp` | 批量经验 `?userIds=1,2,3` | `Result<[{userId, exp}]>` |

## A10. School 模块 — `internal/school/handler.go`

公开接口：

| 方法 | 路径 | 说明 | 响应 |
|------|------|------|------|
| GET | `/api/term/list` | 学期列表 | `Result<[Term]>` |
| GET | `/api/term` | 当前学期 | `Result<{term, currentWeek, date}>` |

需 JWT 接口：

| 方法 | 路径 | 说明 | 请求 | 响应 |
|------|------|------|------|------|
| POST | `/api/course_color` | 课程颜色 | `{colors:{name:hex}}` | `Result<Void>` |

## A11. Other 子模块

### 公告 — `internal/other/notice.go`

| 方法 | 路径 | Auth | 说明 | 响应 |
|------|------|------|------|------|
| GET | `/api/notice/list` | 无 | 公告列表 `?page&size` | `Result<PageResult<Notice>>` |

### 广告 — `internal/other/ad.go`

| 方法 | 路径 | Auth | 说明 | 响应 |
|------|------|------|------|------|
| GET | `/api/ad/list_level` | 无 | 广告列表 | `Result<[Ad]>` |

### 投票 — `internal/other/vote.go`

| 方法 | 路径 | Auth | 说明 | 请求 | 响应 |
|------|------|------|------|------|------|
| GET | `/api/vote/list` | JWT | 投票列表 | `?page&size` | `Result<PageResult<VoteInfo>>` |
| GET | `/api/vote/draft/:info_id` | JWT | 投票选项 | - | `Result<[VoteOption]>` |
| PUT | `/api/vote/draft/:info_id` | JWT | 采纳选项 | `{optionIds:[1,2]}` | `Result<Void>` |
| POST | `/api/vote` | JWT | 创建投票 | `VoteInfo` | `Result<Void>` |
| POST | `/api/vote/:info_id` | JWT | 提交选项 | `VoteOption` | `Result<Void>` |
| POST | `/api/vote/vote/:info_id` | JWT | 投票 | `{optionIds:[1]}` | `Result<Void>` |

### 举报 — `internal/other/report.go`

| 方法 | 路径 | Auth | 说明 | 请求 | 响应 |
|------|------|------|------|------|------|
| POST | `/api/report_comment` | JWT | 举报评论 | `{commentId, topicId, reason}` | `Result<Void>` |

### 前端支持 — `internal/other/support.go`

| 方法 | 路径 | Auth | 说明 | 响应 |
|------|------|------|------|------|
| GET | `/api/support/:key` | 无 | 按 key 获取 | `Result<FrontendSupport>` |
| GET | `/api/support/list` | 无 | 所有支持数据 | `Result<[FrontendSupport]>` |

## A12. WX 模块 — `internal/user/handler.go`

| 方法 | 路径 | Auth | 说明 | 请求 | 响应 |
|------|------|------|------|------|------|
| POST | `/api/wx/unlimited/wxa_code` | 无 | 小程序码 | `{scene, page}` | 二进制图片 (image/png) |

## A13. Event 模块 — `internal/event/handler.go`

| 方法 | 路径 | 说明 | 请求 | 响应 |
|------|------|------|------|------|
| POST | `/api/event` | 上报事件 | `Event` | `Result<Void>` |

写入 Redis List 缓冲，cron 定时批量入库。

## A14. Health & Metrics

| 方法 | 路径 | 说明 | 响应 |
|------|------|------|------|
| GET | `/health` | 健康检查 | `{"status":"UP"}` |
| GET | `/metrics` | Prometheus 指标 | text format |

Prometheus 指标：
- `campus_post_publish_total{result="success|failure"}`
- `campus_comment_publish_total{result="success|failure"}`
- `campus_active_users{window="dau|wau|mau"}` (Gauge)

---

# Part B — ecampus-crm（管理后台）

> handler 文件：各域包的 `admin.go`
> 端口：8081
> 所有 `/admin/**` 接口（除 login）需要 JWT + Admin 权限

## B1. Admin User — `internal/user/admin.go`

| 方法 | 路径 | 说明 | 请求 | 响应 |
|------|------|------|------|------|
| POST | `/admin/user/login` | 管理员登录（免JWT） | `{username, password}` | `Result<{token, refreshToken, user}>` |
| POST | `/admin/user` | 添加用户 | `User` | `Result<Void>` |
| POST | `/admin/user/add` | 添加管理员 | `{userId, username, password}` | `Result<Void>` |
| DELETE | `/admin/user/:id` | 删除用户 | - | `Result<Void>` |
| PUT | `/admin/user/:id` | 编辑用户 | `User`(部分) | `Result<Void>` |
| GET | `/admin/user/:id` | 用户详情 | - | `Result<User>` |
| GET | `/admin/user/list` | 用户列表 | `?page&size&name` | `Result<PageResult<User>>` |
| POST | `/admin/user/clear` | 清除教务认证 | `{userId}` | `Result<Void>` |
| POST | `/admin/user/course` | 异步获取课程 | `{key}` | `Result<Void>` |
| POST | `/admin/user/add_black_list` | 添加黑名单 | `{userIds:[1,2]}` | `Result<Void>` |
| DELETE | `/admin/user/del_black_list` | 移除黑名单 | `{userIds:[1,2]}` | `Result<Void>` |
| GET | `/admin/user/black_list` | 黑名单列表 | - | `Result<[User]>` |
| GET | `/admin/user/certification/list` | 认证申请列表 | `?page&size` | `Result<[OfficialCertification]>` |
| POST | `/admin/user/certification/review` | 审核认证 | `{certId, approved}` | `Result<Void>` |

## B2. Admin Topic — `internal/topic/admin.go`

| 方法 | 路径 | 说明 | 响应 |
|------|------|------|------|
| DELETE | `/admin/topic/:topic_id` | 删除帖子 | `Result<Void>` |
| GET | `/admin/topic/refresh_suggest` | 刷新推荐排行 | `Result<long>` |

## B3. Admin Comment — `internal/comment/admin.go`

| 方法 | 路径 | 说明 | 响应 |
|------|------|------|------|
| DELETE | `/admin/comment/:topic_id/:comment_id` | 删除评论 | `Result<Void>` |

## B4. Admin Theme — `internal/theme/admin.go`

| 方法 | 路径 | 说明 | 请求 | 响应 |
|------|------|------|------|------|
| PUT | `/admin/theme/:id` | 编辑主题 | `Theme` | `Result<Void>` |
| GET | `/admin/theme` | 主题列表 | `?name` | `Result<[Theme]>` |
| PUT | `/admin/theme/search` | 更新搜索标志 | `{themeId, needSearch}` | `Result<Void>` |
| POST | `/admin/theme/suggest` | 更新推荐配置 | `ThemeSuggest` | `Result<Void>` |
| POST | `/admin/theme/campus` | 添加校园主题 | `Theme` | `Result<Void>` |
| DELETE | `/admin/theme/campus/:themeId` | 删除校园主题 | - | `Result<Void>` |

## B5. Admin File — `internal/file/admin.go`

| 方法 | 路径 | 说明 | 请求 | 响应 |
|------|------|------|------|------|
| POST | `/admin/file` | 设为公开 | `{md5List:["xxx"]}` | `Result<Void>` |
| GET | `/admin/file` | 图片列表 | `?page&size` | `Result<CusPage<File>>` |

## B6. Admin School (Term) — `internal/school/admin.go`

| 方法 | 路径 | 说明 | 请求 | 响应 |
|------|------|------|------|------|
| POST | `/admin/term` | 添加学期 | `Term` | `Result<Void>` |
| DELETE | `/admin/term/:id` | 删除学期 | - | `Result<Void>` |
| POST | `/admin/term/cur` | 设置当前学期 | `{termId}` | `Result<Void>` |

## B7. Admin Notice — `internal/other/notice_admin.go`

| 方法 | 路径 | 说明 | 请求 | 响应 |
|------|------|------|------|------|
| POST | `/admin/notice` | 添加公告 | `Notice` | `Result<Void>` |
| DELETE | `/admin/notice/:id` | 删除公告 | - | `Result<Void>` |
| PUT | `/admin/notice/:id` | 编辑公告 | `Notice` | `Result<Void>` |
| GET | `/admin/notice/:id` | 公告详情 | - | `Result<Notice>` |
| GET | `/admin/notice/list` | 公告列表 | `?page&size` | `Result<PageResult<Notice>>` |

## B8. Admin Ad — `internal/other/ad_admin.go`

| 方法 | 路径 | 说明 | 请求 | 响应 |
|------|------|------|------|------|
| POST | `/admin/ad` | 添加广告 | `Ad` | `Result<Void>` |
| DELETE | `/admin/ad/:id` | 删除广告 | - | `Result<Void>` |
| PUT | `/admin/ad/:id` | 编辑广告 | `Ad` | `Result<Void>` |
| GET | `/admin/ad/:id` | 广告详情 | - | `Result<Ad>` |
| GET | `/admin/ad/list` | 广告列表 | `?page&size` | `Result<PageResult<Ad>>` |

## B9. Admin Sensitive Word — `internal/other/sensitive_admin.go`

| 方法 | 路径 | 说明 | 请求 | 响应 |
|------|------|------|------|------|
| GET | `/admin/sensitive/getAllList` | 全部敏感词 | - | `Result<[SensitiveWord]>` |
| GET | `/admin/sensitive/getByWord` | 按词查询 | `?word` | `Result<SensitiveWord>` |
| DELETE | `/admin/sensitive/deleteByWord` | 删除 | `?word` | `Result<Void>` |
| DELETE | `/admin/sensitive/batchDelete` | 批量删除 | `{words:["w1"]}` | `Result<Void>` |
| POST | `/admin/sensitive/add` | 添加 | `SensitiveWord` | `Result<Void>` |
| POST | `/admin/sensitive/batchAdd` | 批量添加 | `{words:["w1"]}` | `Result<Void>` |
| GET | `/admin/sensitive/page` | 分页 | `?page&size` | `Result<PageResult<SensitiveWord>>` |
| GET | `/admin/sensitive/search_like` | 模糊搜索 | `?word` | `Result<[SensitiveWord]>` |
| PUT | `/admin/sensitive/update` | 更新 | `SensitiveWord` | `Result<Void>` |

## B10. Admin Report Comment — `internal/other/report_admin.go`

| 方法 | 路径 | 说明 | 请求 | 响应 |
|------|------|------|------|------|
| PUT | `/admin/report_comment/:id` | 处理举报 | `{status:1}` | `Result<Void>` |
| GET | `/admin/report_comment/list` | 举报列表 | `?page&size` | `Result<CusPage<ReportComment>>` |

## B11. Admin Frontend Support — `internal/other/support_admin.go`

| 方法 | 路径 | 说明 | 请求 | 响应 |
|------|------|------|------|------|
| POST | `/admin/support` | 添加 | `FrontendSupport` | `Result<Void>` |
| PUT | `/admin/support` | 更新 | `FrontendSupport` | `Result<Void>` |
| DELETE | `/admin/support/:id` | 删除 | - | `Result<Void>` |
| GET | `/admin/support/list` | 列表 | - | `Result<[FrontendSupport]>` |

## B12. Admin Merchant Theme — `internal/other/merchant_admin.go`

| 方法 | 路径 | 说明 | 请求 | 响应 |
|------|------|------|------|------|
| POST | `/admin/merchant_theme` | 添加 | `{themeId}` | `Result<Void>` |
| DELETE | `/admin/merchant_theme/:id` | 删除 | - | `Result<Void>` |
| GET | `/admin/merchant_theme/get_all` | 全部 | - | `Result<[MerchantTheme]>` |

## B13. Admin Event — `internal/event/admin.go`

| 方法 | 路径 | 说明 | 请求 | 响应 |
|------|------|------|------|------|
| DELETE | `/admin/event/:id` | 删除事件 | - | `Result<Void>` |
| PUT | `/admin/event/:id` | 编辑事件 | `Event` | `Result<Void>` |
| GET | `/admin/event/:id` | 事件详情 | - | `Result<Event>` |
| GET | `/admin/event/list` | 事件列表 | `?page&size&eventType` | `Result<PageResult<Event>>` |

## B14. Admin Task — `internal/other/` (task 相关放在 other 包)

| 方法 | 路径 | 说明 | 请求 | 响应 |
|------|------|------|------|------|
| POST | `/admin/task` | 添加任务 | `Task` | `Result<Void>` |
| DELETE | `/admin/task/:id` | 删除任务 | - | `Result<Void>` |
| PUT | `/admin/task/:id` | 编辑任务 | `Task` | `Result<Void>` |
| GET | `/admin/task/:id` | 任务详情 | - | `Result<Task>` |
| GET | `/admin/task/list` | 任务列表 | `?page&size` | `Result<PageResult<Task>>` |

## B15. Admin Monitor — `internal/monitor/admin.go`

| 方法 | 路径 | 说明 | 响应 |
|------|------|------|------|
| GET | `/admin/local_cache/all_key` | 缓存名称 | `Result<[string]>` |
| GET | `/admin/local_cache/stats` | 缓存统计 `?cacheName` | `Result<CacheStats>` |

## B16. CRM Health

| 方法 | 路径 | 说明 | 响应 |
|------|------|------|------|
| GET | `/health` | 健康检查 | `{"status":"UP"}` |

---

# API 数量统计

| 服务 | 接口数 |
|------|--------|
| ecampus 公开接口 | ~17 |
| ecampus JWT 接口 | ~55 |
| ecampus 文件 JWT | 2 |
| ecampus WebSocket | 1 |
| ecampus Health/Metrics | 2 |
| **ecampus 合计** | **~83** |
| ecampus-crm admin 接口 | ~65 |
| ecampus-crm health | 1 |
| **ecampus-crm 合计** | **~66** |
| **总计** | **~149** |
