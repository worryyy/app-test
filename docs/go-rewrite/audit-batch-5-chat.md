# Batch 5 审计报告：Chat 模块

审计日期：2026-03-26

审计前提：
- 本次重构切换后，不存在 Java 与 Go 双版本同时部署。
- 旧 Redis 缓存允许在切换前清空。
- Chat 相关的 MySQL `conversations`、`conversation_members` 与 MongoDB `campus_messages`、`campus_notifications` 会沿用现网 Java 存量数据。
- 主报告统计仅计入 Go 独立部署场景下必然触发的问题；仅在特定历史数据条件下触发的问题放在“条件性问题”章节。

## 模块：Chat

### 活跃 API 端点清单

本表列出 Chat 模块中，Java Controller 和 WebSocket 配置已注册且可被客户端直接调用的活跃端点，并与 Go 路由进行交叉比对。

| # | 端点 | 方法 | Java 入口 | Go 入口 | 状态 |
|---|------|------|----------|---------|------|
| 1 | `/api/conversation` | GET | `chat/src/main/java/com/jb/chat/controller/ConversationController.java:22-26` | `internal/chat/handler.go:33-40` | ✅ |
| 2 | `/api/conversation/conversation_enter` | PUT | `chat/src/main/java/com/jb/chat/controller/ConversationController.java:33-39` | `internal/chat/handler.go:42-52` | ✅ |
| 3 | `/api/conversation/{conversation_id}/unread_count` | GET | `chat/src/main/java/com/jb/chat/controller/ConversationController.java:41-45` | `internal/chat/handler.go:54-66` | ✅ |
| 4 | `/api/conversation/conversation_query` | GET | `chat/src/main/java/com/jb/chat/controller/ConversationController.java:47-51` | `internal/chat/handler.go:68-80` | ✅ |
| 5 | `/api/conversation/profile_by_conversation_id` | GET | `chat/src/main/java/com/jb/chat/controller/ConversationController.java:53-57` | `internal/chat/handler.go:82-102` | ✅ |
| 6 | `/api/conversation/{conversation_id}` | DELETE | `chat/src/main/java/com/jb/chat/controller/ConversationController.java:58-63` | `internal/chat/handler.go:104-115` | ✅ |
| 7 | `/api/message/{last_message_id}` | GET | `chat/src/main/java/com/jb/chat/controller/MessageController.java:24-28` | `internal/chat/handler.go:117-129` | ✅ |
| 8 | `/api/message/history_messages` | GET | `chat/src/main/java/com/jb/chat/controller/MessageController.java:30-45` | `internal/chat/handler.go:131-144` | ✅ |
| 9 | `/api/message/unread_messages` | GET | `chat/src/main/java/com/jb/chat/controller/MessageController.java:47-51` | `internal/chat/handler.go:146-153` | ✅ |
| 10 | `/api/notify` | GET | `chat/src/main/java/com/jb/chat/controller/NotifyController.java:24-38` | `internal/chat/handler.go:155-163` | ✅ |
| 11 | `/api/notify/{type}/haveUnread` | GET | `chat/src/main/java/com/jb/chat/controller/NotifyController.java:41-45` | `internal/chat/handler.go:165-172` | ✅ |
| 12 | `/api/notify/{type}` | GET | `chat/src/main/java/com/jb/chat/controller/NotifyController.java:47-51` | `internal/chat/handler.go:174-181` | ✅ |
| 13 | `/chat` | GET(WebSocket Upgrade) | `chat/src/main/java/com/jb/chat/config/WebSocketConfig.java:18-24` | `cmd/ecampus/routes.go:160`, `internal/chat/ws.go:60-143` | ✅ |

状态说明：
- ✅ 端点存在且路径、方法一致
- ❌ Go 中不存在该端点
- ⚠️ 端点存在但路径或方法不一致

### 差异清单

#### DIFF-CHT-01: Go 仍按 camelCase 列名访问会话表，无法兼容 Java 现网 schema

- **等级**: P1
- **分类**: 数据兼容
- **Java 证据**: `chat/src/main/java/com/jb/chat/conversations.sql:1-20`
```sql
CREATE TABLE conversations (
  id VARCHAR(255) PRIMARY KEY,
  last_message_content TEXT,
  last_message_sender_id VARCHAR(255),
  last_message_sent_at TIMESTAMP,
  created_at TIMESTAMP,
  updated_at TIMESTAMP
);
CREATE TABLE conversation_members (
  conversation_id VARCHAR(255),
  user_id VARCHAR(255),
```
- **Go 证据**: `internal/chat/model.go:9-33`，`internal/chat/service.go:41-77`
```go
type Conversation struct {
	ID int64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	LastMessageContent string `gorm:"column:lastMessageContent" json:"lastMessageContent"`
}
type ConversationMember struct {
	ConversationID int64 `gorm:"column:conversationId" json:"conversationId"`
	UserID int64 `gorm:"column:userId" json:"userId"`
}
```
```go
Where("userId = ?", userID).Pluck("conversationId", &ids)
Where("userId = ? AND conversationId = ?", userID, conversationID)
```
- **模拟场景**:
  - 输入: 现网沿用 Java 建表后的库结构，请求 `GET /api/conversation`
  - Java 行为: 查询 `conversation_members.user_id` 与 `conversations.updated_at`，正常返回用户会话列表
  - Go 行为: SQL 会落到 `userId`、`conversationId`、`lastMessageContent`、`lastMessageSentAt` 等不存在的列；接口进入通用错误处理，返回 `{"success":false,"code":-1,"msg":"系统错误","data":null}`
- **预期行为**: Go 对 `conversations`、`conversation_members` 的读写列名必须继续兼容现网 snake_case schema
- **影响面**: `/api/conversation`、`/api/conversation/conversation_enter`、`/api/conversation/{id}/unread_count`、`/api/conversation/conversation_query`、`/api/conversation/profile_by_conversation_id`、`/api/conversation/{id}`

#### DIFF-CHT-02: `conversation_enter` 从 query 参数改成 JSON body，并丢失 `last_message_id` 回执写入

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `chat/src/main/java/com/jb/chat/controller/ConversationController.java:33-39`，`chat/src/main/java/com/jb/chat/service/impl/ConversationServiceImpl.java:97-123`
```java
@PutMapping("/conversation_enter")
public Result<?> enterConversation(
        @RequestParam("conversation_id") String conversationId,
        @RequestParam(value = "last_message_id",required = false) String lastMessageId) {
    return conversationService.enterConversation(conversationId,lastMessageId);
}
```
```java
.set("unread_count", 0)
.set("last_read_message_id", lastMessageId);
```
- **Go 证据**: `internal/chat/model_req.go:3-5`，`internal/chat/handler.go:42-52`，`internal/chat/service.go:57-65`
```go
type ConversationEnterReq struct {
	ConversationID int64 `json:"conversationId" binding:"required"`
}
```
```go
if !result.BindJSON(c, &req) { return }
err := h.svc.EnterConversation(ctx, userID, req.ConversationID)
```
```go
Updates(map[string]interface{}{"unreadCount": 0})
```
- **模拟场景**:
  - 输入: `PUT /api/conversation/conversation_enter?conversation_id=9001&last_message_id=120`
  - Java 行为: `{"success":true,"code":200,"msg":"成功","data":null}`，并把 `conversation_members.last_read_message_id` 更新为 `120`
  - Go 行为: 因请求体为空直接返回 `{"success":false,"code":7,"msg":"请求体不能为空","data":null}`；即便客户端改成 Go 私有格式 `{"conversationId":9001}`，也只会把未读数改成 0，不会写入 `last_read_message_id`
- **预期行为**: 进入会话接口应继续接受 Java 现有请求参数，并在清零未读数时同步写入最后已读消息 ID
- **影响面**: `/api/conversation/conversation_enter`，已读回执、未读数清零后的后续同步逻辑

#### DIFF-CHT-03: `unread_count` 返回值从 `ConversationMember[]` 变成了裸整数

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `chat/src/main/java/com/jb/chat/service/impl/ConversationServiceImpl.java:79-88`
```java
QueryWrapper<ConversationMember> queryWrapper = new QueryWrapper<ConversationMember>()
        .eq("conversation_id", conversationId)
        .eq("user_id",userId)
        .select("unread_count");
List<ConversationMember> unreadCountList = conversationMbrMapper.selectList(queryWrapper);
return R.success(unreadCountList);
```
- **Go 证据**: `internal/chat/handler.go:54-66`，`internal/chat/service.go:67-76`
```go
count, err := h.svc.GetUnreadCount(ctx, middleware.GetUserID(c), id)
result.Success(c, count)
```
- **模拟场景**:
  - 输入: `GET /api/conversation/9001/unread_count`
  - Java 行为: `{"success":true,"code":200,"msg":"成功","data":[{"unreadCount":3}]}`
  - Go 行为: `{"success":true,"code":200,"msg":"成功","data":3}`
- **预期行为**: 未读数接口应继续返回与 Java 一致的数据结构，而不是改成裸整数
- **影响面**: `/api/conversation/{conversation_id}/unread_count`

#### DIFF-CHT-04: `conversation_query` 的入参与出参都不再兼容 Java 客户端

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `chat/src/main/java/com/jb/chat/controller/ConversationController.java:47-50`，`chat/src/main/java/com/jb/chat/service/impl/ConversationServiceImpl.java:137-156`
```java
@GetMapping("/conversation_query")
public Result<?> getConversationByTargetUserId(@RequestParam("target_user_id") String targetUserId) {
    return conversationService.getCommonConversation(targetUserId);
}
```
```java
if(list.isEmpty()){
    return R.data(Collections.emptyList());
}
return R.data(list);
```
- **Go 证据**: `internal/chat/handler.go:68-80`，`internal/chat/service.go:79-100`
```go
targetUserID, err := strconv.ParseInt(c.Query("targetUserId"), 10, 64)
data, err := h.svc.QueryConversation(ctx, middleware.GetUserID(c), targetUserID)
result.Success(c, data)
```
- **模拟场景**:
  - 输入: `GET /api/conversation/conversation_query?target_user_id=42`
  - Java 行为: 如果双方已有会话，返回 `{"success":true,"code":0,"msg":"","data":["9001"]}`；如果没有，返回 `{"success":true,"code":0,"msg":"","data":[]}`
  - Go 行为: 因为只读取 `targetUserId`，同样的 Java 请求会返回 `{"success":false,"code":1,"msg":"参数错误","data":null}`；即便改成 Go 私有参数，也会返回整个 `Conversation` 对象或 `null`，而不是会话 ID 数组
- **预期行为**: 查询共同会话接口应继续接受 `target_user_id`，并返回会话 ID 列表
- **影响面**: `/api/conversation/conversation_query`，1v1 会话去重流程

#### DIFF-CHT-05: `profile_by_conversation_id` 的参数名改变，返回对象还扩大成了完整用户实体

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `chat/src/main/java/com/jb/chat/controller/ConversationController.java:53-56`，`chat/src/main/java/com/jb/chat/service/impl/ConversationServiceImpl.java:165-187`
```java
@GetMapping("/profile_by_conversation_id")
public Result<?> getUserProfileByConversationId(@RequestParam("conversation_id") String conversationId){
    return conversationService.getUserProfile(conversationId);
}
```
```java
UserVO userVO = UserVO.builder()
        .nickname(user.getNickname())
        .avatar(user.getAvatar())
        .userId(String.valueOf(user.getId()))
        .build();
return R.data(userVO);
```
- **Go 证据**: `internal/chat/handler.go:82-101`，`internal/user/model.go:10-33`
```go
conversationID, err := strconv.ParseInt(c.Query("conversationId"), 10, 64)
data, err = h.userSvc.GetByID(c.Request.Context(), peerID)
result.Success(c, data)
```
```go
type User struct {
	ID int64 `json:"id"`
	Nickname string `json:"nickname"`
	Avatar string `json:"avatar"`
	Power int `json:"power"`
	StuPwd string `json:"stuPwd"`
```
- **模拟场景**:
  - 输入: `GET /api/conversation/profile_by_conversation_id?conversation_id=9001`
  - Java 行为: `{"success":true,"code":0,"msg":"","data":{"avatar":"/a.png","nickname":"Alice","userId":"42","gender":"","stuCla":"","signature":""}}`
  - Go 行为: 同样请求直接返回 `{"success":false,"code":1,"msg":"参数错误","data":null}`；即便改成 Go 私有参数 `conversationId=9001`，返回的也是完整用户实体，包含 `id`、`power`、`accountType`、`stuNum`、`stuPwd` 等额外字段
- **预期行为**: 该接口应继续接受 `conversation_id`，并只返回会话对方的轻量资料对象
- **影响面**: `/api/conversation/profile_by_conversation_id`

#### DIFF-CHT-06: 删除会话后，Go 不再清理消息和空会话，也不再拦截越权/不存在场景

- **等级**: P2
- **分类**: 业务逻辑
- **Java 证据**: `chat/src/main/java/com/jb/chat/service/impl/ConversationServiceImpl.java:202-252`
```java
Conversation conversation = conversationMapper.selectById(conversationId);
if (conversation == null) { return R.fail(RC.ERROR_NOT_EXISTED).msg("会话不存在"); }
ConversationMember member = conversationMbrDao.getOne(memberQuery);
if (member == null) { return R.fail().msg("无权限删除该会话"); }
conversationMbrMapper.delete(memberQuery);
mongoTemplate.remove(new Query(Criteria.where("conversation_id").is(conversationId)), Message.class);
if (remainingMemberCount == 0) { conversationMapper.deleteById(conversationId); }
```
- **Go 证据**: `internal/chat/service.go:103-108`
```go
func (s *Service) DeleteConversation(ctx context.Context, userID, conversationID int64) error {
	if err := s.db.WithContext(ctx).Where("userId = ? AND conversationId = ?", userID, conversationID).Delete(&ConversationMember{}).Error; err != nil {
		return fmt.Errorf("delete conversation member: %w", err)
	}
	return nil
}
```
- **模拟场景**:
  - 输入: 用户 7 删除仍有两名成员的会话 `DELETE /api/conversation/9001`
  - Java 行为: `{"success":true,"code":200,"msg":"删除会话成功","data":null}`，并删除 `conversation_id=9001` 的全部消息；如果会话不存在或用户不是成员，则返回失败
  - Go 行为: `{"success":true,"code":200,"msg":"成功","data":null}`，只删除当前用户自己的成员行，不删 Mongo 消息，也不检查是否真的删到记录
- **预期行为**: 删除会话接口应继续保持 Java 的权限校验与数据清理副作用
- **影响面**: `/api/conversation/{conversation_id}`，会话删除后的历史消息可见性

#### DIFF-CHT-07: Go 无法直接读取 Java 已落库消息文档中的字符串 ID 字段

- **等级**: P1
- **分类**: 数据兼容
- **Java 证据**: `chat/src/main/java/com/jb/chat/entity/Message.java:25-45`，`chat/src/main/java/com/jb/chat/entity/dto/ChatMessage.java:34-61`
```java
private Long messageId;
private String conversationId;
private String receiverId;
private String senderId;
private String content;
private Date sentAt;
```
- **Go 证据**: `internal/chat/model.go:35-43`
```go
type Message struct {
	MessageID int64 `bson:"message_id" json:"messageId"`
	ConversationID int64 `bson:"conversation_id" json:"conversationId"`
	ReceiverID int64 `bson:"receiver_id" json:"receiverId"`
	SenderID int64 `bson:"sender_id" json:"senderId"`
```
- **模拟场景**:
  - 输入: Mongo 中已有 Java 写入文档 `{"message_id":123,"conversation_id":"9001","receiver_id":"42","sender_id":"7","content":"hi","sentAt":"2024-03-09T16:00:00Z"}`
  - Java 行为: `/api/message/0`、`/api/message/history_messages` 可以正常读出该消息
  - Go 行为: 本地 BSON 复现实验结果为 `error decoding key conversation_id: cannot decode string into an integer type`；即便查询命中，解码也会失败
- **预期行为**: Go 必须能够直接读取 Java 现网 `campus_messages` 中的旧文档
- **影响面**: `/api/message/{last_message_id}`、`/api/message/history_messages`、`/api/message/unread_messages`、WebSocket 离线补偿链路

#### DIFF-CHT-08: `history_messages` 的参数、排序、游标和鉴权语义都偏离了 Java

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `chat/src/main/java/com/jb/chat/controller/MessageController.java:30-45`，`chat/src/main/java/com/jb/chat/service/impl/MessageServiceImpl.java:82-112`
```java
@GetMapping("/history_messages")
public Result<?> getHistoryMessages(@RequestParam("conversation_id") String conversationId,
                                    @RequestParam(value = "oldest_message_id",required = false) Long oldestMessageId,
                                    @RequestParam(value = "page",defaultValue = "1") Integer page,
                                    @RequestParam(value = "size",defaultValue = "0") Integer size)
```
```java
ConversationMember isMember = conversationMbrMapper.selectOne(...)
if (Objects.isNull(isMember)) { return R.fail().msg("无权访问该会话历史"); }
Criteria criteria = Criteria.where("conversation_id").is(conversationId);
if (Objects.nonNull(oldestMessageId)) { criteria.and("message_id").lt(oldestMessageId); }
Query query = new Query(criteria).with(Sort.by(Sort.Direction.ASC, "message_id")).limit(50);
```
- **Go 证据**: `internal/chat/handler.go:131-144`，`internal/chat/service.go:146-178`
```go
conversationID, err := strconv.ParseInt(c.Query("conversationId"), 10, 64)
data, err := h.svc.GetHistoryMessages(c.Request.Context(), conversationID, page, size)
```
```go
filter := bson.M{"conversation_id": conversationID}
cur, err := s.messageColl().Find(ctx, filter, options.Find().
	SetSort(bson.M{"message_id": -1}).
	SetSkip(int64((page-1)*size)).
	SetLimit(int64(size)))
```
- **模拟场景**:
  - 输入: 当前用户不是会话 `9001` 成员，但请求 `GET /api/message/history_messages?conversation_id=9001&oldest_message_id=500&page=1&size=15`
  - Java 行为: `{"success":false,"code":400,"msg":"无权访问该会话历史","data":null}`；对合法成员则按 `message_id ASC` 和 `oldest_message_id` 游标返回最多 50 条
  - Go 行为: 同样的 Java 请求会先因缺少 `conversationId` 返回 `{"success":false,"code":1,"msg":"参数错误","data":null}`；如果把参数改成 Go 私有格式 `conversationId=9001`，接口会直接按页码倒序返回该会话消息，不再校验当前用户是否是会话成员，也完全忽略 `oldest_message_id`
- **预期行为**: 历史消息接口应继续接受 Java 现有参数、执行会话成员校验，并保持基于 `oldest_message_id` 的升序游标语义
- **影响面**: `/api/message/history_messages`

#### DIFF-CHT-09: `unread_messages` 在 Java 只是布尔标志，Go 改成了消息列表

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `chat/src/main/java/com/jb/chat/controller/MessageController.java:47-50`，`chat/src/main/java/com/jb/chat/service/impl/MessageServiceImpl.java:115-119`
```java
@GetMapping("/unread_messages")
public Result<?> getUnreadMessages(){
    return messageService.getUnread();
}
```
```java
return R.success(conversationMbrDao.HasUnreadOrNot(userId));
```
- **Go 证据**: `internal/chat/handler.go:146-153`，`internal/chat/service.go:180-195`
```go
func (h *Handler) UnreadMessages(c *gin.Context) {
	data, err := h.svc.GetUnreadMessages(c.Request.Context(), middleware.GetUserID(c))
	result.Success(c, data)
}
```
- **模拟场景**:
  - 输入: `GET /api/message/unread_messages`
  - Java 行为: `{"success":true,"code":200,"msg":"成功","data":true}`
  - Go 行为: `{"success":true,"code":200,"msg":"成功","data":[{"id":"660000000000000000000001","messageId":123,"conversationId":9001,"receiverId":42,"senderId":7,"content":"hi","sentAt":"2024-03-09T16:00:00Z"}]}`
- **预期行为**: 未读消息接口应继续返回布尔值，而不是返回消息列表
- **影响面**: `/api/message/unread_messages`

#### DIFF-CHT-10: 离线消息拉取改成了“只看 receiver_id”，丢失 Java 的会话级增量同步语义

- **等级**: P2
- **分类**: 业务逻辑
- **Java 证据**: `chat/src/main/java/com/jb/chat/service/impl/MessageServiceImpl.java:55-69`
```java
List<String> conversationIds = getConversationIds(userId);
Criteria criteria = Criteria.where("conversation_id").in(conversationIds);
if (Objects.nonNull(lastMessageId)) {
    criteria.and("message_id").gt(lastMessageId);
}
List<Message> messages = mongoTemplate.find(new Query(criteria)
        .with(Sort.by(Sort.Direction.ASC, "message_id")), Message.class);
```
- **Go 证据**: `internal/chat/service.go:124-143`
```go
filter := bson.M{"receiver_id": userID}
if lastMessageID > 0 {
	filter["message_id"] = bson.M{"$gt": lastMessageID}
}
cur, err := s.messageColl().Find(ctx, filter, options.Find().SetSort(bson.M{"message_id": 1}))
```
- **模拟场景**:
  - 输入: 用户 42 参与会话 `9001`，`last_message_id=100` 之后该会话新增了两条消息：`101` 为别人发给 42，`102` 为 42 在另一台设备发出
  - Java 行为: `GET /api/message/100` 返回两条消息 `101`、`102`
  - Go 行为: `GET /api/message/100` 只会返回 `receiver_id=42` 的 `101`，不会返回 `102`
- **预期行为**: 离线增量同步应继续按“用户参与的会话 + 消息游标”工作，而不是只按接收者过滤
- **影响面**: `/api/message/{last_message_id}`，多端同步

#### DIFF-CHT-11: `notify` 列表在空结果形态和已读副作用上都偏离了 Java

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `chat/src/main/java/com/jb/chat/service/impl/NotifyServiceImpl.java:35-59`
```java
List<Notification> notifications = mongoTemplate.find(query.with(request), Notification.class);
if (notifications.isEmpty()) {
    return R.data(Collections.emptyList());
}
Notification updatedNotification = mongoTemplate.findAndModify(
        updateQuery,
        new Update().set("is_read", true),
        Notification.class
);
return R.success(cusPage);
```
- **Go 证据**: `internal/chat/service.go:198-233`
```go
total, err := s.notifyColl().CountDocuments(ctx, filter)
cur, err := s.notifyColl().Find(ctx, filter, options.Find().
	SetSort(bson.M{"created_time": -1}).
	SetSkip(int64((page-1)*size)).
	SetLimit(int64(size)))
return result.NewCusPage(list, total, page, size), nil
```
- **模拟场景**:
  - 输入-A: 当前用户没有 `COMMENT_ADD` 通知，请求 `GET /api/notify?type=COMMENT_ADD&page=1&size=15`
  - Java 行为-A: `{"success":true,"code":0,"msg":"","data":[]}`
  - Go 行为-A: `{"success":true,"code":200,"msg":"成功","data":{"data":[],"current":1,"total":0,"size":15}}`
  - 输入-B: 当前用户最新一条 `COMMENT_ADD` 通知 `is_read=false`，请求 `GET /api/notify?type=COMMENT_ADD&page=1&size=15`
  - Java 行为-B: 返回列表后，会把最新一条通知原子更新为 `is_read=true`
  - Go 行为-B: 返回列表，但不会更新任何通知的 `is_read`
- **预期行为**: 通知列表接口应继续保持 Java 的空结果返回形态，并在读取列表时更新最新通知的已读状态
- **影响面**: `/api/notify`

#### DIFF-CHT-12: `haveUnread` 从“看最新一条是否未读”变成了“只要历史里有未读就返回 true”

- **等级**: P2
- **分类**: 业务逻辑
- **Java 证据**: `chat/src/main/java/com/jb/chat/service/impl/NotifyServiceImpl.java:71-83`
```java
Query query = buildQuery(userId, type);
Notification notification = mongoTemplate.findOne(query, Notification.class);
return R.success(notification != null && !notification.getIsRead());
```
- **Go 证据**: `internal/chat/service.go:235-245`
```go
filter := bson.M{"receiver_id": fmt.Sprintf("%d", userID), "is_read": false}
if typ != "" {
	filter["type"] = typ
}
count, err := s.notifyColl().CountDocuments(ctx, filter)
return count > 0, nil
```
- **模拟场景**:
  - 输入: 同一 `type=COMMENT_ADD` 下，最新一条通知 `created_time=20, is_read=true`，更老一条通知 `created_time=10, is_read=false`；请求 `GET /api/notify/COMMENT_ADD/haveUnread`
  - Java 行为: `{"success":true,"code":200,"msg":"成功","data":false}`
  - Go 行为: `{"success":true,"code":200,"msg":"成功","data":true}`
- **预期行为**: 未读标志接口应继续与 Java 一致，只反映该类型“最新通知摘要”的未读状态
- **影响面**: `/api/notify/{type}/haveUnread`

#### DIFF-CHT-13: WebSocket 消息与通知帧协议整体不兼容 Java 客户端

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `chat/src/main/java/com/jb/chat/handler/ChatHandler.java:68-90`，`chat/src/main/java/com/jb/chat/service/impl/ChatServiceImpl.java:68-72`，`chat/src/main/java/com/jb/chat/service/impl/ChatServiceImpl.java:179-207`，`chat/src/main/java/com/jb/chat/mq/consumer/NotificationConsumer.java:60-65`
```java
if (AUTH_MESSAGE_TYPE.equals(messageMap.get("type"))) {
    handleAuthentication(session, actSession, messageMap);
    return;
}
chatService.handleMessage(message);
```
```java
if (HANDLE_TYPE_INIT.equals(map.get(HANDLE_TYPE))) {
    handleInitMessage(payload);
} else {
    handleChatMessage(payload);
}
```
```java
receiverSession.sendMessage(new TextMessage(objectMapper.writeValueAsString(chatMessage)));
session.sendMessage(new TextMessage(objectMapper.writeValueAsString(notification)));
```
- **Go 证据**: `internal/chat/ws.go:52-58`，`internal/chat/ws.go:123-141`，`internal/chat/service.go:264-291`，`internal/chat/realtime.go:19-22`
```go
type wsEnvelope struct {
	Type string `json:"type"`
	ConversationID int64 `json:"conversationId"`
	ReceiverID int64 `json:"receiverId"`
	Content string `json:"content"`
}
```
```go
if env.Type != "message" {
	_ = conn.WriteJSON(gin.H{"type": "error", "msg": "unsupported message type"})
	continue
}
_ = conn.WriteJSON(gin.H{"type": "message_ack", "data": data})
_ = peer.Conn.WriteJSON(gin.H{"type": "message", "data": data})
```
```go
session.Conn.WriteJSON(map[string]interface{}{
	"type": "notification",
	"data": payload,
})
```
- **模拟场景**:
  - 输入-A: 用户 7 已鉴权后按 Java 现有协议发送首条建会话消息 `{"handleType":"INIT","id":"9001","senderId":"7","receiverId":"42","content":"hi","sentAt":"2024-03-09T16:00:00"}`
  - Java 行为-A: 创建会话与成员、写入 `campus_messages`，若用户 42 在线则直接推送原始消息帧 `{"messageId":1234567890,"conversationId":"9001","senderId":"7","receiverId":"42","content":"hi","sentAt":"2024-03-09T16:00:00"}`
  - Go 行为-A: 因缺少 `type:"message"` 返回 `{"type":"error","msg":"unsupported message type"}`；不会创建会话，也不会推送消息
  - 输入-B: MQ 产生一条通知，目标用户 42 在线
  - Java 行为-B: 直接推送原始通知对象，字段为 camelCase，例如 `{"id":"660...001","receiverId":"42","senderId":"7","type":"TOPIC_LIKE","content":"x","topicId":"t1","commentId":"c1","createdTime":1710000000000,"isRead":false}`
  - Go 行为-B: 推送包装帧 `{"type":"notification","data":{"receiver_id":"42","sender_id":"7","type":"TOPIC_LIKE","content":"x","topic_id":"t1","comment_id":"c1","created_time":"2024-03-09T16:00:00Z","is_read":false}}`
- **预期行为**: WebSocket 入站消息、实时聊天推送与实时通知推送都应继续兼容 Java 现有帧协议
- **影响面**: `/chat`，场景 A 在线投递、场景 B 离线后补偿前的实时通知

### 条件性问题

#### DIFF-CHT-C01: [条件性] 现网若存在非数字会话 ID，Go 的 Chat 路由参数解析会整体失败

- **等级**: P1
- **分类**: 数据兼容
- **Java 证据**: `chat/src/main/java/com/jb/chat/conversations.sql:1-2`，`chat/src/main/java/com/jb/chat/controller/ConversationController.java:35-37`，`chat/src/main/java/com/jb/chat/controller/MessageController.java:32-33`
```sql
id VARCHAR(255) PRIMARY KEY COMMENT '会话ID (UUID或者雪花算法)'
```
```java
@RequestParam("conversation_id") String conversationId
```
- **Go 证据**: `internal/chat/handler.go:55-56`，`internal/chat/handler.go:83-84`，`internal/chat/handler.go:133-134`
```go
id, err := strconv.ParseInt(c.Param("id"), 10, 64)
conversationID, err := strconv.ParseInt(c.Query("conversationId"), 10, 64)
```
- **模拟场景**:
  - 输入: 现网若存在会话 ID `550e8400-e29b-41d4-a716-446655440000`，请求 `GET /api/conversation/550e8400-e29b-41d4-a716-446655440000/unread_count`
  - Java 行为: 继续按字符串会话 ID 正常处理
  - Go 行为: 直接返回 `{"success":false,"code":1,"msg":"参数错误","data":null}`
- **预期行为**: 只要现网历史里存在字符串会话 ID，Go 就必须继续接受并处理它们
- **影响面**: `/api/conversation/{conversation_id}/unread_count`、`/api/conversation/profile_by_conversation_id`、`/api/message/history_messages`、`/api/conversation/{conversation_id}`

### 模块总结

- 活跃端点: 13 个
- Go 已覆盖: 13 个
- P0 差异: 8 个
- P1 差异: 2 个
- P2 差异: 3 个
