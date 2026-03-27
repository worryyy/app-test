# Batch 3 审计报告：Topic + Theme 模块

审计日期：2026-03-26

审计前提：
- 本次重构切换后，不存在 Java 与 Go 双版本同时部署。
- 旧 Redis 缓存允许在切换前清空。
- 主报告统计仅计入 Go 独立部署场景下必然触发的问题。
- 仅在 Java/Go 混部、共享 Mongo/Redis、或共享旧脚本时才成立的问题，单独放在文末“条件性问题”章节，不计入主统计。

## 模块：Topic

### 活跃 API 端点清单

| # | 端点 | 方法 | Java 入口 | Go 入口 | 状态 |
|---|------|------|----------|---------|------|
| 1 | `/api/topic` | POST | `theme/src/main/java/com/jb/theme/controller/TopicController.java:95-128` | `internal/topic/handler.go:20-31` | ✅ |
| 2 | `/api/topic/{id}` | DELETE | `theme/src/main/java/com/jb/theme/controller/TopicController.java:134-144` | `internal/topic/handler.go:33-39` | ✅ |
| 3 | `/api/topic/{topic_id}` | GET | `theme/src/main/java/com/jb/theme/controller/TopicController.java:172-179` | `internal/topic/handler.go:41-49` | ✅ |
| 4 | `/api/topic/{topic_id}` | PUT | `theme/src/main/java/com/jb/theme/controller/TopicController.java:181-213` | `internal/topic/handler.go:51-61` | ✅ |
| 5 | `/api/topic/search` | GET | `theme/src/main/java/com/jb/theme/controller/TopicController.java:224-254` | `internal/topic/handler.go:63-74` | ✅ |
| 6 | `/api/topic` | GET | `theme/src/main/java/com/jb/theme/controller/TopicController.java:262-276` | `internal/topic/handler.go:76-84` | ✅ |
| 7 | `/api/topic/theme` | GET | `theme/src/main/java/com/jb/theme/controller/TopicController.java:285-301` | `internal/topic/handler.go:86-94` | ✅ |
| 8 | `/api/topic/target_user_topics` | GET | `theme/src/main/java/com/jb/theme/controller/TopicController.java:310-325` | `internal/topic/handler.go:96-109` | ✅ |
| 9 | `/api/topic/follow_topics` | GET | `theme/src/main/java/com/jb/theme/controller/TopicController.java:331-345` | `internal/topic/handler.go:111-119` | ✅ |
| 10 | `/api/like/topic/{topic_id}` | POST | `theme/src/main/java/com/jb/theme/controller/TopicLikeController.java:32-52` | `internal/topic/handler.go:121-127` | ✅ |
| 11 | `/api/like/topic/{topic_id}` | DELETE | `theme/src/main/java/com/jb/theme/controller/TopicLikeController.java:54-63` | `internal/topic/handler.go:129-135` | ✅ |
| 12 | `/api/like/topic` | GET | `theme/src/main/java/com/jb/theme/controller/TopicLikeController.java:72-86` | `internal/topic/handler.go:137-145` | ✅ |
| 13 | `/api/collection/topic/{topic_id}` | POST | `theme/src/main/java/com/jb/theme/controller/CollectionController.java:49-59` | `internal/topic/handler.go:147-153` | ✅ |
| 14 | `/api/collection/topic/{topic_id}` | DELETE | `theme/src/main/java/com/jb/theme/controller/CollectionController.java:61-70` | `internal/topic/handler.go:155-161` | ✅ |
| 15 | `/api/collection/collection_topics` | GET | `theme/src/main/java/com/jb/theme/controller/CollectionController.java:80-94` | `internal/topic/handler.go:163-171` | ✅ |
| 16 | `/admin/topic/{topic_id}` | DELETE | `theme/src/main/java/com/jb/theme/controller/admin/AdmTopicController.java:31-37` | `internal/topic/admin.go:17-23` | ✅ |
| 17 | `/admin/topic/refresh_suggest` | GET | `theme/src/main/java/com/jb/theme/controller/admin/AdmTopicController.java:39-44` | `internal/topic/admin.go:25-32` | ✅ |

状态说明：
- ✅ 端点存在且路径、方法一致
- ❌ Go 中不存在该端点
- ⚠️ 端点存在但路径或方法不一致

### 差异清单

#### DIFF-TOP-01: Go 无法读取现网 Java 帖子文档中的 `accountType`

- **等级**: P1
- **分类**: 数据兼容
- **Java 证据**: `theme-entity/src/main/java/com/jb/themeentity/entity/Topic.java:46-46`，`theme/src/main/java/com/jb/theme/service/impl/TopicServiceImpl.java:420-431`
```java
private String accountType;
topic.setHasLike(topicLikeExists(userId,topicId));
topic.setHasCollection(hasCollectionOrNot(topicId,userId));
topic.setCreatedTime(new ObjectId(topic.getId()).getDate());
return R.data(topic);
```
- **Go 证据**: `internal/topic/model.go:18-18`，`internal/topic/service.go:128-135`
```go
AccountType int `bson:"accountType" json:"accountType"`
err = s.topicColl().FindOne(ctx, bson.M{"_id": oid, "hasCheck": true}).Decode(&topic)
if err == mongo.ErrNoDocuments {
    return nil, nil
}
```
- **模拟场景**:
  - 输入: `campus_topic` 中已有 Java 写入文档 `{"_id":"660000000000000000000001","themeId":"10001","userId":"42","accountType":"anonymous","title":"旧帖","content":"旧内容","hasCheck":true}`
  - Java 行为: `GET /api/topic/660000000000000000000001` 可正常返回，`data.accountType` 为 `"anonymous"`
  - Go 行为: 本地 BSON 复现实验结果为 `error decoding key accountType: cannot decode string into an integer type`；接口会进入错误处理，返回 `{"success":false,"code":-1,"msg":"系统错误","data":null}`
- **预期行为**: Go 应能直接读取 Java 已落库的帖子文档，并继续对外返回字符串类型的 `accountType`
- **影响面**: `/api/topic/{topic_id}`、`/api/topic`、`/api/topic/search`、`/api/topic/theme`、`/api/topic/target_user_topics`、`/api/topic/follow_topics`、`/api/like/topic`、`/api/collection/collection_topics`

#### DIFF-TOP-02: 创建帖子返回体从完整 Topic 变成字符串 ID，且作者信息不再自动补齐

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `theme/src/main/java/com/jb/theme/controller/TopicController.java:100-127`
```java
ThemeId theme = themeService.getThemeByThemeId(topic.getThemeId());
if (theme == null) {
    return R.fail(RC.ERROR_NOT_EXISTED).msg("themeId is not existed");
}
topic.setUserId(user.getId().toString());
topic.setAvatar(user.getAvatar());
topic.setNickName(user.getNickname());
topic.setAccountType(user.getAccountType());
return R.data(save);
```
- **Go 证据**: `internal/topic/service.go:56-71`，`internal/topic/handler.go:25-30`
```go
topic := Topic{
    ThemeID: req.ThemeID,
    UserID:  strconv.FormatInt(claims.UserID, 10),
    AccountType: mapAccountType(claims.AccountType),
    NickName: req.NickName,
    Avatar:   req.Avatar,
}
result.Success(c, id)
```
- **模拟场景**:
  - 输入: 已登录用户 `42` 调用 `POST /api/topic`，请求体 `{"themeId":"10001","title":"午饭","content":"食堂二楼","imgs":["a.jpg"],"ext":""}`
  - Java 行为: `{"success":true,"code":0,"msg":"","data":{"id":"660...","themeId":"10001","userId":"42","avatar":"u.png","nickName":"张三","accountType":"base","title":"午饭","content":"食堂二楼","imgs":["a.jpg"],"hasCheck":false,"ext":"","visitedNum":0,"likeNum":0,"commentNum":0,"collectionNum":0,"hasLike":false,"hasCollection":false}}`
  - Go 行为: `{"success":true,"code":200,"msg":"成功","data":"660..."}`；并且按 Java 契约请求时不会传 `avatar`/`nickName`，落库值会是空字符串
- **预期行为**: 创建成功后应返回完整 Topic 对象，且作者 `avatar`、`nickName`、`accountType` 应取自当前身份；`themeId` 也应先校验存在性
- **影响面**: `/api/topic`

#### DIFF-TOP-03: 帖子详情缺少 `createdTime`，且当前用户的点赞/收藏状态始终不准

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `theme/src/main/java/com/jb/theme/service/impl/TopicServiceImpl.java:420-431`，`theme-entity/src/main/java/com/jb/themeentity/entity/Topic.java:85-98`
```java
incVisitedNum(topicId);
topic.setHasLike(topicLikeExists(userId,topicId));
topic.setHasCollection(hasCollectionOrNot(topicId,userId));
topic.setCreatedTime(new ObjectId(topic.getId()).getDate());
return R.data(topic);
```
- **Go 证据**: `internal/topic/service.go:142-148`，`internal/topic/model.go:18-23`
```go
if queryUserID != "" {
    if fillErr := s.fillLikeAndCollection(ctx, queryUserID, []Topic{topic}); fillErr != nil {
        ...
    }
}
return &topic, nil
```
- **模拟场景**:
  - 输入: 当前用户 `42` 已点赞并收藏帖子 `660...`，请求 `GET /api/topic/660...`
  - Java 行为: `{"success":true,"code":0,"msg":"","data":{"id":"660...","themeId":"10001","userId":"99","avatar":"u.png","nickName":"作者","accountType":"base","title":"午饭","content":"食堂二楼","imgs":["a.jpg"],"hasCheck":true,"visitedNum":10,"likeNum":1,"commentNum":0,"collectionNum":1,"createdTime":"2026-03-20 12:00:00","hasLike":true,"hasCollection":true}}`
  - Go 行为: `{"success":true,"code":200,"msg":"成功","data":{"id":"660...","themeId":"10001","userId":"99","title":"午饭","content":"食堂二楼","imgs":["a.jpg"],"hasCheck":true,"visitedNum":10,"likeNum":1,"commentNum":0,"collectionNum":1,"accountType":1,"nickName":"","avatar":"","hasLike":false,"hasCollection":false}}`
- **预期行为**: 帖子详情应继续返回 `createdTime`，并准确反映当前用户的 `hasLike` / `hasCollection`
- **影响面**: `/api/topic/{topic_id}`

#### DIFF-TOP-04: Topic 过滤查询参数名变更，Java 客户端请求会直接走错分支

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `theme/src/main/java/com/jb/theme/controller/TopicController.java:233-237`，`theme/src/main/java/com/jb/theme/controller/TopicController.java:288-290`，`theme/src/main/java/com/jb/theme/controller/TopicController.java:313-315`
```java
@RequestParam(value = "themeIds", required = false) String themeIds,
@RequestParam(value = "content", required = false) String content,
@RequestParam(value = "theme_id") String themeId
@RequestParam(value = "target_user_id") String targetUserId
```
- **Go 证据**: `internal/topic/handler.go:65-68`，`internal/topic/handler.go:88-100`
```go
themeID := c.Query("themeId")
keyword := c.Query("keyword")
orderBy := c.Query("orderBy")
data, err := h.svc.ListByTheme(..., c.Query("themeId"), ...)
targetUserID, err := strconv.ParseInt(c.Query("targetUserId"), 10, 64)
```
- **模拟场景**:
  - 输入: `GET /api/topic/target_user_topics?target_user_id=7&page=1&size=10`
  - Java 行为: `{"success":true,"code":0,"msg":"","data":{"data":[...],"current":1,"total":1,"size":10}}`
  - Go 行为: `{"success":false,"code":1,"msg":"参数错误","data":null}`
- **预期行为**: Go 应继续接受 `themeIds`、`content`、`ord_created`、`theme_id`、`target_user_id`
- **影响面**: `/api/topic/search`、`/api/topic/theme`、`/api/topic/target_user_topics`

#### DIFF-TOP-05: 更新帖子在 Go 中变成必须提交完整创建体，Java 的部分更新请求会失败

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `theme/src/main/java/com/jb/theme/dto/UpdateTopicDTO.java:17-33`，`theme/src/main/java/com/jb/theme/controller/TopicController.java:183-197`
```java
private String title;
private String content;
private List<String> imgs;
private String ext;
private Boolean hasCheck;
if (!(updateTitle || updateContent || updateExt || updateImgs || updateHasCheck)) {
    return R.success();
}
```
- **Go 证据**: `internal/topic/handler.go:51-56`，`internal/topic/model.go:49-57`
```go
var req CreateTopicReq
if !result.BindJSON(c, &req) {
    return
}
ThemeID string `json:"themeId" binding:"required"`
Title   string `json:"title" binding:"required"`
Content string `json:"content" binding:"required"`
```
- **模拟场景**:
  - 输入: `PUT /api/topic/660...`，请求体 `{"title":"新标题"}`
  - Java 行为: `{"success":true,"code":200,"msg":"成功","data":null}`
  - Go 行为: HTTP 400，`{"success":false,"code":1,"msg":"参数错误","data":null}`
- **预期行为**: 更新接口应继续支持只提交被修改字段，并保留 `ext`、`hasCheck`
- **影响面**: `/api/topic/{topic_id}`

#### DIFF-TOP-06: 多个空列表场景在 Java 返回 `[]`，Go 返回分页对象

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `theme/src/main/java/com/jb/theme/service/impl/TopicServiceImpl.java:371-373`，`theme/src/main/java/com/jb/theme/service/impl/TopicLikeServiceImpl.java:229-232`，`theme/src/main/java/com/jb/theme/service/impl/TopicCollectionImpl.java:152-154`
```java
if(CollectionUtils.isEmpty(topicList)) {
    return R.data(Collections.emptyList());
}
if(topicLikeList.isEmpty()){
    return R.data(Collections.emptyList());
}
```
- **Go 证据**: `internal/topic/service_social.go:207-224`
```go
if len(allIDs) == 0 {
    return result.NewCusPage([]Topic{}, 0, page, size), nil
}
```
- **模拟场景**:
  - 输入: 当前用户没有任何点赞记录，调用 `GET /api/like/topic?page=1&size=10`
  - Java 行为: `{"success":true,"code":0,"msg":"","data":[]}`
  - Go 行为: `{"success":true,"code":200,"msg":"成功","data":{"data":[],"current":1,"total":0,"size":10}}`
- **预期行为**: 这些空结果场景应继续保持 Java 返回结构，而不是统一切成分页对象
- **影响面**: `/api/topic/theme`、`/api/topic/target_user_topics`、`/api/like/topic`、`/api/collection/collection_topics`

#### DIFF-TOP-07: Java 审核通过前会剔除二维码图片，Go 不会

- **等级**: P2
- **分类**: 业务逻辑
- **Java 证据**: `theme/src/main/java/com/jb/theme/mq/consumer/TopicCheckConsumer.java:138-158`
```java
filteredImgs = qrCodeDetectionService.filterImagesWithQRCode(originalImgs);
if (filteredImgs != null && filteredImgs.size() != originalImgs.size()) {
    one.setImgs(filteredImgs);
}
one.setHasCheck(true);
topicService.saveOne(one);
```
- **Go 证据**: `internal/mq/consumer_topic_comment.go:71-100`
```go
titleResult, err := c.wxClient.MsgSecCheck(ctx, topic.Title, topic.UserID)
contentResult, err := c.wxClient.MsgSecCheck(ctx, topic.Content, topic.UserID)
...
"$set": bson.M{
    "hasCheck": true,
    "title":    filteredTitle,
    "content":  filteredContent,
},
```
- **模拟场景**:
  - 输入: 用户创建帖子，`imgs=["https://cdn.example/qr.png","https://cdn.example/normal.png"]`，文本审核都通过
  - Java 行为: 审核通过后 `campus_topic.imgs` 变为 `["https://cdn.example/normal.png"]`
  - Go 行为: 审核通过后 `campus_topic.imgs` 仍保留两张图
- **预期行为**: 审核链路应继续剔除包含二维码的图片，再对外发布帖子
- **影响面**: `/api/topic` 创建后的所有详情/列表读取场景

#### DIFF-TOP-08: 管理端刷新推荐榜在 Go 中只自增版本号，没有真正重建排行榜

- **等级**: P1
- **分类**: 业务逻辑
- **Java 证据**: `theme/src/main/java/com/jb/theme/controller/admin/AdmTopicController.java:39-44`，`theme/src/main/java/com/jb/theme/cron/GenerateSuggestTopic.java:166-195`
```java
Long l = generateSuggestTopic.generateAll();
return R.success().msg("刷新"+l+"条排行榜数据");
...
Long l = redisTemplate.opsForZSet().unionAndStore(keys.get(0), keys.subList(1, keys.size()), nex);
changeKey(nex);
topicService.refreshSuggestListCache();
```
- **Go 证据**: `internal/topic/admin.go:25-31`，`internal/topic/service_search.go:228-236`
```go
version, err := h.svc.RefreshSuggest(c.Request.Context())
result.Success(c, version)

v, err := s.redis.Incr(ctx, rediskey.SuggestCountKey).Result()
return v, nil
```
- **模拟场景**:
  - 输入: 调用 `GET /admin/topic/refresh_suggest`
  - Java 行为: `{"success":true,"code":200,"msg":"刷新12条排行榜数据","data":null}`，并刷新 `rank:all_*`、切换 `rank:prevKey/rank:curKey`
  - Go 行为: `{"success":true,"code":200,"msg":"成功","data":13}`，只做 `INCR suggest:cnt`，不刷新任何排行榜数据
- **预期行为**: 管理端刷新操作应真正重建推荐榜数据，而不是只返回版本号
- **影响面**: `/admin/topic/refresh_suggest` 及依赖推荐榜 Redis 数据的场景

#### DIFF-TOP-09: 点赞/收藏别人的帖子时，Go 不再发送通知 MQ

- **等级**: P1
- **分类**: 业务逻辑
- **Java 证据**: `theme/src/main/java/com/jb/theme/service/impl/TopicServiceImpl.java:173-199`，`theme/src/main/java/com/jb/theme/mq/producer/TopicLikeNotifyProducer.java:17-30`
```java
topicLikeNotifyProducer.notify(topic.getUserId(), userId, id, "点赞了你的帖子", NotifyType.TOPIC_LIKE.name());
topicLikeNotifyProducer.notify(topic.getUserId(), userId, id, "收藏了你的帖子",NotifyType.TOPIC_COLLECTION.name());
```
- **Go 证据**: `internal/topic/service_social.go:16-44`，`internal/topic/service_social.go:67-95`
```go
if res.ModifiedCount > 0 || res.UpsertedCount > 0 {
    _, incErr := s.topicColl().UpdateByID(ctx, mustObjectID(topicID), bson.M{"$inc": bson.M{"likeNum": 1}})
}
...
if res.ModifiedCount > 0 || res.UpsertedCount > 0 {
    _, incErr := s.topicColl().UpdateByID(ctx, mustObjectID(topicID), bson.M{"$inc": bson.M{"collectionNum": 1}})
}
```
- **模拟场景**:
  - 输入: 用户 `42` 点赞用户 `99` 的帖子 `660...`
  - Java 行为: 除更新 `campus_topic_like` 和 `topic.likeNum` 外，还发送通知载荷 `{"receiverId":"99","senderId":"42","notifyType":"TOPIC_LIKE","topicId":"660...","content":"点赞了你的帖子"}`
  - Go 行为: 只更新 Mongo 点赞记录和 `likeNum`，不会产生通知 MQ
- **预期行为**: 点赞/收藏他人帖子后，应继续产生与 Java 一致的通知消息
- **影响面**: `/api/like/topic/{topic_id}`、`/api/collection/topic/{topic_id}`，以及通知列表/未读提醒

### 模块总结

- 活跃端点: 17 个
- Go 已覆盖: 17 个
- P0 差异: 5 个
- P1 差异: 3 个
- P2 差异: 1 个
- 热度排序未发现差异：Java 与 Go 的活跃热度公式都为 `commentNum*9 + likeNum*6 + visitedNum*1`
- 用例 `views=100, likes=50, comments=20, collects=10` 下，两端热度分值都为 `580`
- Java 活跃 `/api/topic/search` 实际直接查询 `campus_topic`，未走 `campus_topic_search`；后者只出现在 MQ 索引链路

## 模块：Theme

### 活跃 API 端点清单

| # | 端点 | 方法 | Java 入口 | Go 入口 | 状态 |
|---|------|------|----------|---------|------|
| 1 | `/api/theme/campus/init` | POST | `theme/src/main/java/com/jb/theme/controller/ThemeController.java:33-37` | `internal/theme/handler.go:15-22` | ✅ |
| 2 | `/api/theme/campus` | GET | `theme/src/main/java/com/jb/theme/controller/ThemeController.java:39-43` | `internal/theme/handler.go:24-31` | ✅ |
| 3 | `/admin/theme/{id}` | PUT | `theme/src/main/java/com/jb/theme/controller/admin/AdmThemeController.java:36-49` | `internal/theme/admin.go:15-25` | ✅ |
| 4 | `/admin/theme` | GET | `theme/src/main/java/com/jb/theme/controller/admin/AdmThemeController.java:51-60` | `internal/theme/admin.go:27-34` | ✅ |
| 5 | `/admin/theme/search` | PUT | `theme/src/main/java/com/jb/theme/controller/admin/AdmThemeController.java:62-69` | `internal/theme/admin.go:36-46` | ✅ |
| 6 | `/admin/theme/suggest` | POST | `theme/src/main/java/com/jb/theme/controller/admin/AdmThemeController.java:71-75` | `internal/theme/admin.go:48-64` | ✅ |
| 7 | `/admin/theme/campus` | POST | `theme/src/main/java/com/jb/theme/controller/admin/AdmThemeController.java:77-85` | `internal/theme/admin.go:66-77` | ✅ |
| 8 | `/admin/theme/campus/{themeId}` | DELETE | `theme/src/main/java/com/jb/theme/controller/admin/AdmThemeController.java:87-92` | `internal/theme/admin.go:79-90` | ✅ |

状态说明：
- ✅ 端点存在且路径、方法一致
- ❌ Go 中不存在该端点
- ⚠️ 端点存在但路径或方法不一致

### 差异清单

#### DIFF-THM-01: Go 无法读取现网 Java 主题文档中的 `suggestType`

- **等级**: P1
- **分类**: 数据兼容
- **Java 证据**: `theme-entity/src/main/java/com/jb/themeentity/entity/Theme.java:43-44`
```java
@ApiModelProperty("推荐帖子的展现形式，前端决定")
private Integer suggestType;
```
- **Go 证据**: `internal/theme/model.go:13-14`，`internal/theme/service.go:90-103`
```go
SuggestType string `bson:"suggestType" json:"suggestType"`
var themes []Theme
if err := cur.All(ctx, &themes); err != nil {
    return nil, fmt.Errorf("decode themes: %w", err)
}
```
- **模拟场景**:
  - 输入: `campus_theme` 中已有 Java 文档 `{"_id":"65f...","name":"日常","category_name":"生活","needSearch":true,"suggestType":1}`
  - Java 行为: `GET /admin/theme` 正常返回，`suggestType` 为数字 `1`
  - Go 行为: 本地 BSON 复现实验结果为 `error decoding key suggestType: cannot decode 32-bit integer into a string type`；接口返回 `{"success":false,"code":-1,"msg":"系统错误","data":null}`
- **预期行为**: Go 应能直接读取 Java 已落库的主题配置，并继续保持数值型 `suggestType`
- **影响面**: `/admin/theme` 及所有依赖 `campus_theme` 解码的后台主题操作

#### DIFF-THM-02: 编辑主题时，Java 的 `category_name` 请求体会在 Go 中被写成空字符串，且返回体从 Theme 对象变成 `null`

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `theme/src/main/java/com/jb/theme/dto/admin/ThemeDTO.java:18-29`，`theme/src/main/java/com/jb/theme/controller/admin/AdmThemeController.java:37-49`
```java
private String name;
private String category_name;
private Boolean needSearch;
one.setCategory_name(dto.getCategory_name());
return R.data(themeService.edit(oldName, one));
```
- **Go 证据**: `internal/theme/model.go:7-14`，`internal/theme/admin.go:15-24`，`internal/theme/service.go:147-156`
```go
CategoryName string `bson:"category_name" json:"categoryName"`
if !result.BindJSON(c, &req) { return }
result.Success(c, nil)
"name":          theme.Name,
"category_name": theme.CategoryName,
```
- **模拟场景**:
  - 输入: `PUT /admin/theme/65f...`，请求体 `{"name":"日常","category_name":"生活","needSearch":true}`
  - Java 行为: `{"success":true,"code":0,"msg":"","data":{"id":"65f...","name":"日常","category_name":"生活","needSearch":true}}`
  - Go 行为: `{"success":true,"code":200,"msg":"成功","data":null}`；`category_name` 无法绑定到 `categoryName`，持久化后变成空字符串
- **预期行为**: 编辑主题应继续接受 `category_name`，并返回更新后的 Theme 对象
- **影响面**: `/admin/theme/{id}`

#### DIFF-THM-03: `PUT /admin/theme/search` 从批量启用搜索变成单主题切换

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `theme/src/main/java/com/jb/theme/dto/admin/ThemeIdsDTO.java:17-21`，`theme/src/main/java/com/jb/theme/controller/admin/AdmThemeController.java:62-68`
```java
private List<String> themeIds;
List<String> ids = themeIds.getThemeIds();
themeService.updateNeedSearch(ids, true);
return R.success();
```
- **Go 证据**: `internal/theme/model_req.go:3-6`，`internal/theme/admin.go:36-45`
```go
type ThemeSearchReq struct {
    ThemeID    string `json:"themeId" binding:"required"`
    NeedSearch bool   `json:"needSearch"`
}
```
- **模拟场景**:
  - 输入: `PUT /admin/theme/search`，请求体 `{"themeIds":["65f1","65f2"]}`
  - Java 行为: `{"success":true,"code":200,"msg":"成功","data":null}`，两个主题都会被设置为 `needSearch=true`
  - Go 行为: HTTP 400，`{"success":false,"code":1,"msg":"参数错误","data":null}`
- **预期行为**: 该接口应继续接受 `themeIds` 数组，并对数组中的主题批量开启搜索
- **影响面**: `/admin/theme/search`

#### DIFF-THM-04: `POST /admin/theme/suggest` 从批量按主题名配置推荐，变成单主题按 ID 配置

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `theme/src/main/java/com/jb/theme/dto/admin/ThemeSuggestDTO.java:19-22`，`theme/src/main/java/com/jb/theme/dto/admin/inner/ThemeSuggest.java:18-47`，`theme/src/main/java/com/jb/theme/controller/admin/AdmThemeController.java:72-74`
```java
private List<@Valid @NotNull ThemeSuggest> list;
private String theme_name;
private Integer suggestType;
return R.data(themeService.editSuggestByList(themeSuggestDTO));
```
- **Go 证据**: `internal/theme/model_req.go:8-14`，`internal/theme/admin.go:48-63`
```go
type ThemeSuggestReq struct {
    ThemeID     string `json:"themeId" binding:"required"`
    SuggestType string `json:"suggestType"`
}
result.Success(c, nil)
```
- **模拟场景**:
  - 输入: `POST /admin/theme/suggest`，请求体 `{"list":[{"theme_name":"日常","needSuggest":true,"suggestBasicScore":80,"suggestNumber":10,"suggestType":1,"suggestSetName":"hot_daily"}]}`
  - Java 行为: `{"success":true,"code":0,"msg":"","data":[{"name":"日常","suggestBasicScore":80,"suggestNumber":10,"suggestSetName":"hot_daily","suggestType":1,"needSuggest":true}]}`
  - Go 行为: HTTP 400，`{"success":false,"code":1,"msg":"参数错误","data":null}`
- **预期行为**: 该接口应继续接受 Java 的批量 `list` 结构，并返回更新后的主题配置列表
- **影响面**: `/admin/theme/suggest`

#### DIFF-THM-05: 主题名称过滤从精确匹配变成正则包含匹配

- **等级**: P2
- **分类**: 业务逻辑
- **Java 证据**: `theme/src/main/java/com/jb/theme/controller/admin/AdmThemeController.java:52-59`，`theme/src/main/java/com/jb/theme/service/impl/ThemeServiceImpl.java:215-218`
```java
if(StringUtils.isBlank(name)) {
    res = themeService.getAll();
} else {
    res = themeService.getAllThemeWithName(name);
}
Criteria.where("name").is(name)
```
- **Go 证据**: `internal/theme/admin.go:27-33`，`internal/theme/service.go:85-90`
```go
if name != "" {
    filter["name"] = bson.M{"$regex": name}
}
result.Success(c, data)
```
- **模拟场景**:
  - 输入: 库中存在主题 `"日常"`、`"日记"`，请求 `GET /admin/theme?name=日`
  - Java 行为: `{"success":true,"code":0,"msg":"","data":[]}`
  - Go 行为: `{"success":true,"code":200,"msg":"成功","data":[{"name":"日常",...},{"name":"日记",...}]}`
- **预期行为**: 主题名称过滤应继续保持 Java 的精确匹配语义
- **影响面**: `/admin/theme`

### 模块总结

- 活跃端点: 8 个
- Go 已覆盖: 8 个
- P0 差异: 3 个
- P1 差异: 1 个
- P2 差异: 1 个

## 模块：WX / Sensitive

### 活跃 API 端点清单

| # | 端点 | 方法 | Java 入口 | Go 入口 | 状态 |
|---|------|------|----------|---------|------|
| 1 | `/api/wx/unlimited/wxa_code` | POST | `theme/src/main/java/com/jb/theme/controller/WXaCodeController.java:26-42` | `internal/user/handler.go:327-340` | ✅ |
| 2 | `/admin/sensitive/getAllList` | GET | `theme/src/main/java/com/jb/theme/controller/SensitiveWordController.java:22-26` | `internal/other/sensitive_admin.go:9-16` | ✅ |
| 3 | `/admin/sensitive/getByWord` | GET | `theme/src/main/java/com/jb/theme/controller/SensitiveWordController.java:28-38` | `internal/other/sensitive_admin.go:18-25` | ✅ |
| 4 | `/admin/sensitive/deleteByWord` | DELETE | `theme/src/main/java/com/jb/theme/controller/SensitiveWordController.java:40-46` | `internal/other/sensitive_admin.go:27-33` | ✅ |
| 5 | `/admin/sensitive/batchDelete` | DELETE | `theme/src/main/java/com/jb/theme/controller/SensitiveWordController.java:48-54` | `internal/other/sensitive_admin.go:35-45` | ✅ |
| 6 | `/admin/sensitive/add` | POST | `theme/src/main/java/com/jb/theme/controller/SensitiveWordController.java:56-62` | `internal/other/sensitive_admin.go:47-57` | ✅ |
| 7 | `/admin/sensitive/batchAdd` | POST | `theme/src/main/java/com/jb/theme/controller/SensitiveWordController.java:64-70` | `internal/other/sensitive_admin.go:59-69` | ✅ |
| 8 | `/admin/sensitive/page` | GET | `theme/src/main/java/com/jb/theme/controller/SensitiveWordController.java:72-80` | `internal/other/sensitive_admin.go:71-79` | ✅ |
| 9 | `/admin/sensitive/search_like` | GET | `theme/src/main/java/com/jb/theme/controller/SensitiveWordController.java:82-88` | `internal/other/sensitive_admin.go:81-88` | ✅ |
| 10 | `/admin/sensitive/update` | PUT | `theme/src/main/java/com/jb/theme/controller/SensitiveWordController.java:90-98` | `internal/other/sensitive_admin.go:90-99` | ✅ |

状态说明：
- ✅ 端点存在且路径、方法一致
- ❌ Go 中不存在该端点
- ⚠️ 端点存在但路径或方法不一致

### 差异清单

#### DIFF-WXS-01: 小程序码接口从 Base64 字符串变成 PNG 二进制，且把 `page` 从可选改成必填

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `theme/src/main/java/com/jb/theme/controller/WXaCodeController.java:26-42`，`theme/src/main/java/com/jb/theme/pojo/WxaCodeRequest.java:11-38`，`theme/src/main/java/com/jb/theme/service/impl/WXHelperServiceImpl.java:263-285`
```java
@PostMapping("/unlimited/wxa_code")
public String generateUnlimitedWxaCode(@Validated @RequestBody WxaCodeRequest request, ...)
private String page;
private boolean checkPath = true;
return Base64.getEncoder().encodeToString(response.getBody());
```
- **Go 证据**: `internal/user/handler.go:327-340`，`internal/pkg/wxutil/client.go:175-196`
```go
Scene string `json:"scene" binding:"required"`
Page  string `json:"page" binding:"required"`
c.Data(http.StatusOK, "image/png", data)

"scene": scene,
"page":  page,
"check_path": false,
```
- **模拟场景**:
  - 输入 A: `POST /api/wx/unlimited/wxa_code`，请求体 `{"scene":"topicId=1"}`
  - Java 行为: 接受请求，`page` 为空值，返回 Base64 字符串
  - Go 行为: HTTP 400，`{"success":false,"code":1,"msg":"参数错误","data":null}`
  - 输入 B: `POST /api/wx/unlimited/wxa_code`，请求体 `{"scene":"topicId=1","page":"pages/topic/detail"}`
  - Java 行为: 返回 Base64 字符串响应体，例如 `"iVBORw0KGgo..."`
  - Go 行为: 返回 `image/png` 二进制字节流
- **预期行为**: 该接口应继续保持 Java 的请求体兼容性与响应体格式
- **影响面**: `/api/wx/unlimited/wxa_code`

#### DIFF-WXS-02: 敏感词新增/更新接口从 Query 参数改成 JSON Body

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `theme/src/main/java/com/jb/theme/controller/SensitiveWordController.java:56-62`，`theme/src/main/java/com/jb/theme/controller/SensitiveWordController.java:90-97`
```java
@PostMapping("/add")
public Result<?> addSensitiveWord(@RequestParam("word") String word)

@PutMapping("/update")
public Result<?> updateSensitiveWordByWord(
    @RequestParam("word") String word,
    @RequestParam("updateWord") String updateWord)
```
- **Go 证据**: `internal/other/sensitive_admin.go:47-56`，`internal/other/sensitive_admin.go:90-99`
```go
var req SensitiveWord
if !result.BindJSON(c, &req) { return }
...
var req SensitiveWord
if !result.BindJSON(c, &req) { return }
```
- **模拟场景**:
  - 输入: `POST /admin/sensitive/add?word=广告开户链接`
  - Java 行为: `{"success":true,"code":200,"msg":"关键词：广告开户链接添加成功","data":null}`
  - Go 行为: HTTP 400，`{"success":false,"code":7,"msg":"请求体不能为空","data":null}`
- **预期行为**: 新增/更新敏感词应继续使用 Java 约定的 Query 参数格式
- **影响面**: `/admin/sensitive/add`、`/admin/sensitive/update`

#### DIFF-WXS-03: 敏感词批量接口从原始数组改成 `{"words":[...]}` 对象

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `theme/src/main/java/com/jb/theme/controller/SensitiveWordController.java:48-69`
```java
@DeleteMapping("/batchDelete")
public Result<?> deleteSensitiveWordsByList(@RequestBody List<String> words)

@PostMapping("/batchAdd")
public Result<?> addSensitiveWordsByList(@RequestBody List<String> words)
```
- **Go 证据**: `internal/other/model_req.go:11-13`，`internal/other/sensitive_admin.go:35-68`
```go
type WordsReq struct {
    Words []string `json:"words" binding:"required"`
}
var req WordsReq
if !result.BindJSON(c, &req) { return }
```
- **模拟场景**:
  - 输入: `DELETE /admin/sensitive/batchDelete`，请求体 `["广告","兼职"]`
  - Java 行为: `{"success":true,"code":200,"msg":"批量删除关键词成功","data":null}`
  - Go 行为: HTTP 400，`{"success":false,"code":1,"msg":"参数错误","data":null}`
- **预期行为**: 批量新增/删除接口应继续接受 Java 的原始字符串数组
- **影响面**: `/admin/sensitive/batchAdd`、`/admin/sensitive/batchDelete`

### 模块总结

- 活跃端点: 10 个
- Go 已覆盖: 10 个
- P0 差异: 3 个
- P1 差异: 0 个
- P2 差异: 0 个

## 条件性问题

以下问题仅在 Java/Go 混部、共享 Mongo/Redis、或仍有旧脚本依赖旧数据格式时成立，不计入上述统计。

#### [条件性] DIFF-TOP-C1: Go 把 `topic_like` / `topic_collection` 的 `themeName` 语义改成了 `themeId`

- **等级**: P1
- **分类**: 数据兼容
- **Java 证据**: `theme/src/main/java/com/jb/theme/service/impl/TopicLikeServiceImpl.java:61-66`，`theme/src/main/java/com/jb/theme/service/impl/TopicCollectionImpl.java:79-83`
```java
TopicLike.builder()
    .userId(userId)
    .themeName(themeName)
    .accountType(ThreadLocalUtil.getAccountType())
```
- **Go 证据**: `internal/topic/service_social.go:25-33`，`internal/topic/service_social.go:76-83`
```go
filter := bson.M{"userId": ..., "themeName": topic.ThemeID}
"$setOnInsert": bson.M{
    "themeName":   topic.ThemeID,
    "accountType": 1,
}
```
- **模拟场景**:
  - 输入: 用户点赞主题 `themeId=10001`、主题名 `"日常"` 的帖子
  - Java 行为: `campus_topic_like` 写入 `{"userId":"42","themeName":"日常","accountType":"base","topicIds":["660..."]}`
  - Go 行为: `campus_topic_like` 写入 `{"userId":"42","themeName":"10001","accountType":1,"topicIds":["660..."]}`
- **预期行为**: 如果 Java/Go 需要共享 Mongo 数据，`themeName` 和 `accountType` 的落库语义必须一致
- **影响面**: 仅在 Java/Go 共享 Mongo 或仍有 Java 清理/消费逻辑读取这些集合时触发

#### [条件性] DIFF-TOP-C2: 推荐榜 Redis key 空间已改名，旧脚本或混部服务会读不到

- **等级**: P1
- **分类**: 数据兼容
- **Java 证据**: `theme/src/main/java/com/jb/theme/cron/GenerateSuggestTopic.java:30-36`，`theme/src/main/java/com/jb/theme/service/impl/TopicServiceImpl.java:59-64`
```java
private static final String CNT_KEY = "rank:cnt";
public static final String PREV_KEY = "rank:prevKey";
public static final String CUR_KEY = "rank:curKey";
private final String SUGGEST_TOPIC_LIST_KEY = "rank:list";
```
- **Go 证据**: `internal/pkg/rediskey/keys.go:63-75`
```go
SuggestCurKey    = "suggest:cur"
SuggestPrevKey   = "suggest:prev"
SuggestCountKey  = "suggest:cnt"
SuggestTopicListKey = "campus:suggest_topic_list"
```
- **模拟场景**:
  - 输入: 外部脚本或 Java 进程读取当前推荐榜指针
  - Java 行为: 读取 `rank:curKey` / `rank:list`
  - Go 行为: 只写 `suggest:cur` / `campus:suggest_topic_list`
- **预期行为**: 如果存在共享 Redis 或旧运维脚本，键名语义必须一致
- **影响面**: 仅在 Java/Go 共享 Redis、共享推荐榜运维脚本或外部消费者时触发
