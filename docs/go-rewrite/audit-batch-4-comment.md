# Batch 4 审计报告：Comment 模块

审计日期：2026-03-26

审计前提：
- 本次重构切换后，不存在 Java 与 Go 双版本同时部署。
- 旧 Redis 缓存允许在切换前清空。
- 评论、举报、通知相关的 MongoDB/MySQL 数据会沿用现网 Java 存量数据。
- 主报告统计仅计入 Go 独立部署场景下必然触发的问题。

## 模块：Comment

### 活跃 API 端点清单

本表列出 Comment 模块中，Java Controller 已注册且可被客户端直接调用的活跃端点，并与 Go 路由进行交叉比对。

| # | 端点 | 方法 | Java 入口 | Go 入口 | 状态 |
|---|------|------|----------|---------|------|
| 1 | `/api/comment/{topic_id}` | POST | `theme/src/main/java/com/jb/theme/controller/CommentController.java:73-137` | `internal/comment/handler.go:20-45` | ✅ |
| 2 | `/api/comment/{topic_id}/{comment_id}` | DELETE | `theme/src/main/java/com/jb/theme/controller/CommentController.java:144-160` | `internal/comment/handler.go:47-53` | ✅ |
| 3 | `/api/comment/{topic_id}` | GET | `theme/src/main/java/com/jb/theme/controller/CommentController.java:196-238` | `internal/comment/handler.go:55-63` | ✅ |
| 4 | `/api/comment` | GET | `theme/src/main/java/com/jb/theme/controller/CommentController.java:242-278` | `internal/comment/handler.go:65-73` | ✅ |
| 5 | `/api/comment/target_user_comments` | GET | `theme/src/main/java/com/jb/theme/controller/CommentController.java:288-301` | `internal/comment/handler.go:75-88` | ✅ |
| 6 | `/api/comment_like/{comment_id}` | POST | `theme/src/main/java/com/jb/theme/controller/CommentLikeController.java:25-31` | `internal/comment/handler.go:90-96` | ✅ |
| 7 | `/api/comment_like/{comment_id}` | DELETE | `theme/src/main/java/com/jb/theme/controller/CommentLikeController.java:34-40` | `internal/comment/handler.go:98-104` | ✅ |
| 8 | `/api/report_comment` | POST | `theme/src/main/java/com/jb/theme/controller/ReportCommentController.java:31-38` | `internal/other/report.go:12-32` | ✅ |
| 9 | `/admin/comment/{topic_id}/{comment_id}` | DELETE | `theme/src/main/java/com/jb/theme/controller/admin/AdmCommentController.java:24-33` | `internal/comment/admin.go:17-23` | ✅ |
| 10 | `/admin/report_comment/{id}` | PUT | `theme/src/main/java/com/jb/theme/controller/admin/AdmReportCommentController.java:44-61` | `internal/other/report_admin.go:9-18` | ✅ |
| 11 | `/admin/report_comment/list` | GET | `theme/src/main/java/com/jb/theme/controller/admin/AdmReportCommentController.java:64-83` | `internal/other/report_admin.go:21-28` | ✅ |

状态说明：
- ✅ 端点存在且路径、方法一致
- ❌ Go 中不存在该端点
- ⚠️ 端点存在但路径或方法不一致

### 差异清单

#### DIFF-CMT-01: Go 无法读取 Java 已落库评论中的 `user.accountType`

- **等级**: P1
- **分类**: 数据兼容
- **Java 证据**: `theme-entity/src/main/java/com/jb/themeentity/entity/inner/CmtUser.java:14-26`
```java
private String userId;
private String avatar;
private String nickName;
private String gender;
private String accountType;
private String signature;
```
- **Go 证据**: `internal/comment/model.go:25-30`
```go
type CommentUser struct {
	UserID      string `bson:"userId" json:"userId"`
	NickName    string `bson:"nickName" json:"nickName"`
	Avatar      string `bson:"avatar" json:"avatar"`
	AccountType int    `bson:"accountType" json:"accountType"`
}
```
- **模拟场景**:
  - 输入: `campus_comment` 中已有 Java 写入文档 `{"_id":"660000000000000000000101","topicId":"t1","comment":"旧评论","user":{"userId":"42","nickName":"Alice","avatar":"a.png","accountType":"base"}}`
  - Java 行为: `GET /api/comment/t1?page=1&size=15&root_id=0` 可正常返回该评论，`data[0].user.accountType` 为 `"base"`
  - Go 行为: 本地 BSON 复现实验结果为 `error decoding key user.accountType: cannot decode string into an integer type`；接口会进入错误处理，返回 `{"success":false,"code":-1,"msg":"系统错误","data":null}`
- **预期行为**: Go 应能直接读取 Java 现网评论文档，并继续对外返回字符串类型的 `user.accountType`
- **影响面**: `/api/comment/{topic_id}`、`/api/comment`、`/api/comment/target_user_comments`

#### DIFF-CMT-02: Go 只认 `hasCheck=true`，会把 Java 侧原本可见的评论过滤掉

- **等级**: P1
- **分类**: 数据兼容
- **Java 证据**: `theme/src/main/java/com/jb/theme/controller/CommentController.java:217-219`，`theme/src/main/java/com/jb/theme/controller/CommentController.java:259-260`，`theme/src/main/java/com/jb/theme/service/impl/CommentServiceImpl.java:198-199`
```java
Query query = Query.query(Criteria.where("topicId").is(topicId)
        .and("rootCmtId").is(rootId)
        .and("hasCheck").ne(false));
```
- **Go 证据**: `internal/comment/service.go:116-125`
```go
return s.listByFilter(ctx, bson.M{"topicId": topicID, "hasCheck": true}, page, size)
return s.listByFilter(ctx, bson.M{"user.userId": strconv.FormatInt(userID, 10), "hasCheck": true}, page, size)
return s.listByFilter(ctx, bson.M{"user.userId": strconv.FormatInt(targetUserID, 10), "hasCheck": true}, page, size)
```
- **模拟场景**:
  - 输入: Mongo 中已有 Java 写入评论文档 `{"_id":"660000000000000000000102","topicId":"t1","rootCmtId":"0","comment":"旧评论"}`，没有 `hasCheck` 字段
  - Java 行为: `GET /api/comment/t1?page=1&size=15&root_id=0` 返回 `{"success":true,"code":0,"msg":"","data":{"data":[{"id":"660000000000000000000102","topicId":"t1","comment":"旧评论","rootCmtId":"0"}],"current":1,"total":1,"size":15}}`
  - Go 行为: 返回 `{"success":true,"code":200,"msg":"成功","data":{"data":[],"current":1,"total":0,"size":15}}`
- **预期行为**: Go 应继续把 Java 侧原本可见的评论视为可见评论
- **影响面**: `/api/comment/{topic_id}`、`/api/comment`、`/api/comment/target_user_comments`

#### DIFF-CMT-03: Go 缺少 Java 的评论权限限制，匿名与自评匿名帖都会被放行

- **等级**: P2
- **分类**: 业务逻辑
- **Java 证据**: `theme/src/main/java/com/jb/theme/controller/CommentController.java:80-97`
```java
if (Objects.equals(accountType, AccountType.ANONYMOUS.getCode())
        && !Objects.equals(one.getAccountType(), AccountType.ANONYMOUS.getCode())) {
    forbiddenMsg = "匿名用户禁止评论非匿名帖";
} else if (Objects.equals(accountType, AccountType.BASE.getCode())
        && Objects.equals(one.getAccountType(), AccountType.ANONYMOUS.getCode())
        && Objects.equals(rootUserId, topicRootUserId)) {
    forbiddenMsg = "禁止左右脑互搏和自导自演";
}
if (forbiddenMsg != null) {
    return R.fail(RC.ERROR_FORBIDDEN).msg(forbiddenMsg);
}
```
- **Go 证据**: `internal/comment/handler.go:25-44`，`internal/comment/service.go:55-87`
```go
userID := getUserID(c)
claims := getClaims(c)
_, err := h.svc.AddComment(c.Request.Context(), c.Param("topic_id"), CommentUser{
	UserID:      strconv.FormatInt(userID, 10),
	AccountType: accountType,
}, req.Comment, req.ParentCmtID, req.RootCmtID, false)
```
- **模拟场景**:
  - 输入: 匿名身份 token 调用 `POST /api/comment/t1`，请求体 `{"comment":"匿名评论","parentCmtId":"0"}`；其中 `t1` 是普通帖子
  - Java 行为: `{"success":false,"code":5,"msg":"匿名用户禁止评论非匿名帖","data":null}`
  - Go 行为: `{"success":true,"code":200,"msg":"成功","data":null}`
- **预期行为**: Go 应继续拦截 Java 已禁止的评论场景
- **影响面**: `/api/comment/{topic_id}`

#### DIFF-CMT-04: Go 直接信任客户端的 `rootCmtId`，回复评论的树结构会写错

- **等级**: P2
- **分类**: 业务逻辑
- **Java 证据**: `theme/src/main/java/com/jb/theme/controller/CommentController.java:110-128`
```java
Comment parent = commentService.getOne(parentCmtId);
comment.setParent(parent.getUser());
if(!parent.getRootCmtId().equals(Comment.DEFAULT_ROOT)) {
    comment.setRootCmtId(parent.getRootCmtId());
} else {
    comment.setRootCmtId(parent.getId());
}
```
- **Go 证据**: `internal/comment/model_req.go:3-7`，`internal/comment/handler.go:36-39`，`internal/comment/service.go:56-67`
```go
type CreateCommentReq struct {
	Comment     string `json:"comment" binding:"required"`
	ParentCmtID string `json:"parentCmtId"`
	RootCmtID   string `json:"rootCmtId"`
}
```
```go
RootCmtID:   rootCmtID,
HasCheck:    false,
```
- **模拟场景**:
  - 输入: 已存在根评论 `r1(rootCmtId="0")` 和它的子评论 `c1(rootCmtId="r1")`；客户端按 Java 现有契约只提交 `POST /api/comment/t1`，body 为 `{"comment":"reply","parentCmtId":"c1"}`
  - Java 行为: 新评论落库为 `{"parentCmtId":"c1","rootCmtId":"r1","parent":{"userId":"42",...}}`，根评论 `r1.commentNum` 加 1
  - Go 行为: 新评论落库为 `{"parentCmtId":"c1","rootCmtId":"","parent":null}`，后续 MQ 也不会给 `r1.commentNum` 加 1
- **预期行为**: 回复评论时，Go 应自动建立正确的 `parent/root` 关系，而不是依赖客户端额外传 `rootCmtId`
- **影响面**: `/api/comment/{topic_id}`、`/api/comment/{topic_id}` 列表、回复数显示

#### DIFF-CMT-05: `/api/comment/{topic_id}` 忽略 `root_id`，并且排序与 `hasLike` 语义一起偏离

- **等级**: P2
- **分类**: 业务逻辑
- **Java 证据**: `theme/src/main/java/com/jb/theme/controller/CommentController.java:201-238`，`theme/src/main/java/com/jb/theme/service/impl/CommentServiceImpl.java:168-192`
```java
@RequestParam(value = "root_id", defaultValue = Comment.DEFAULT_ROOT) String rootId
.and("rootCmtId").is(rootId)
request = PageRequest.of(page - 1, size, Sort.by(Sort.Direction.DESC, "commentNum","_id"));
o.setHasLike(hasLikeBatch.contains(o.getId()))
```
- **Go 证据**: `internal/comment/handler.go:55-63`，`internal/comment/service.go:163-194`
```go
data, err := h.svc.ListByTopic(c.Request.Context(), c.Param("topic_id"), page, size)
```
```go
opts := options.Find().
	SetSort(bson.M{"createdTime": -1})
```
- **模拟场景**:
  - 输入: `topic=t1` 下有根评论 `r1(rootCmtId="0", commentNum=1)` 和子评论 `c1(rootCmtId="r1")`；当前登录用户已点赞 `r1`；请求 `GET /api/comment/t1?root_id=0&page=1&size=10`
  - Java 行为: `{"success":true,"code":0,"msg":"","data":{"data":[{"id":"r1","topicId":"t1","rootCmtId":"0","commentNum":1,"hasLike":true}],"current":1,"total":1,"size":10}}`
  - Go 行为: `{"success":true,"code":200,"msg":"成功","data":{"data":[{"id":"c1","topicId":"t1","rootCmtId":"r1","hasLike":false},{"id":"r1","topicId":"t1","rootCmtId":"0","hasLike":false}],"current":1,"total":2,"size":10}}`
- **预期行为**: `root_id` 应决定返回哪一层评论，并按基线顺序返回，同时正确回填 `hasLike`
- **影响面**: `/api/comment/{topic_id}`

#### DIFF-CMT-06: 新评论的公开字段与 Java 契约不一致

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `theme/src/main/java/com/jb/theme/controller/CommentController.java:98-132`，`theme-entity/src/main/java/com/jb/themeentity/entity/inner/CmtUser.java:14-26`
```java
CmtUser cmtUser = CmtUser.builder()
        .avatar(user.getAvatar())
        .nickName(user.getNickname())
        .accountType(user.getAccountType())
        .signature(user.getSignature())
        .build();
comment.setIsAuthor(one.getUserId().equals(userId.toString()));
comment.setRootCmtId(Comment.DEFAULT_ROOT);
```
- **Go 证据**: `internal/comment/handler.go:36-39`，`internal/comment/model.go:25-30`，`internal/comment/service.go:56-67`
```go
CommentUser{
	UserID:      strconv.FormatInt(userID, 10),
	AccountType: accountType,
}
```
```go
type CommentUser struct {
	UserID      string `bson:"userId" json:"userId"`
	NickName    string `bson:"nickName" json:"nickName"`
	Avatar      string `bson:"avatar" json:"avatar"`
	AccountType int    `bson:"accountType" json:"accountType"`
}
```
- **模拟场景**:
  - 输入: 话题作者 `userId=100` 发根评论，`POST /api/comment/t1`，body 为 `{"comment":"hi","parentCmtId":"0"}`
  - Java 行为: 后续 `GET /api/comment/t1?root_id=0&page=1&size=15` 返回 `{"success":true,"code":0,"msg":"","data":{"data":[{"comment":"hi","parentCmtId":"0","rootCmtId":"0","isAuthor":true,"user":{"userId":"100","nickName":"Alice","avatar":"/a.png","accountType":"base","signature":"hello"}}],"current":1,"total":1,"size":15}}`
  - Go 行为: 返回 `{"success":true,"code":200,"msg":"成功","data":{"data":[{"comment":"hi","parentCmtId":"0","rootCmtId":"","isAuthor":false,"user":{"userId":"100","nickName":"","avatar":"","accountType":1}}],"current":1,"total":1,"size":15}}`
- **预期行为**: 新评论公开 JSON 中的用户快照、`accountType` 类型、`isAuthor`、`rootCmtId` 等字段应与基线保持一致
- **影响面**: `/api/comment/{topic_id}`、`/api/comment`

#### DIFF-CMT-07: `/api/comment` 返回 `Comment` 分页，Java 返回 `MyCommentVO{comment,topic}`

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `theme/src/main/java/com/jb/theme/controller/CommentController.java:262-277`，`theme/src/main/java/com/jb/theme/vo/MyCommentVO.java:16-18`
```java
return MyCommentVO.builder()
        .comment(o)
        .topic(topicService.getTopicsForComment(o.getTopicId())).build();
```
- **Go 证据**: `internal/comment/handler.go:65-73`，`internal/comment/service.go:120-121`
```go
data, err := h.svc.ListMine(c.Request.Context(), getUserID(c), page, size)
return s.listByFilter(ctx, bson.M{"user.userId": strconv.FormatInt(userID, 10), "hasCheck": true}, page, size)
```
- **模拟场景**:
  - 输入: 当前用户有一条评论 `c1`，所属帖子 `t1(title="帖子")`；请求 `GET /api/comment?page=1&size=15`
  - Java 行为: `{"success":true,"code":0,"msg":"","data":{"data":[{"comment":{"id":"c1","topicId":"t1","comment":"hi"},"topic":{"id":"t1","title":"帖子"}}],"current":1,"total":1,"size":15}}`
  - Go 行为: `{"success":true,"code":200,"msg":"成功","data":{"data":[{"id":"c1","topicId":"t1","comment":"hi"}],"current":1,"total":1,"size":15}}`
- **预期行为**: “我的评论”接口应继续返回 `comment + topic` 的组合结构
- **影响面**: `/api/comment`

#### DIFF-CMT-08: `/api/comment/target_user_comments` 查询参数从 `target_user_id` 变成了 `targetUserId`

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `theme/src/main/java/com/jb/theme/controller/CommentController.java:289-301`，`theme/src/main/java/com/jb/theme/service/impl/CommentServiceImpl.java:202-206`
```java
@RequestParam("target_user_id") String targetUserId
if(comments.isEmpty()) {
    return R.data(Collections.emptyList());
}
```
- **Go 证据**: `internal/comment/handler.go:75-80`
```go
targetUserID, err := strconv.ParseInt(c.Query("targetUserId"), 10, 64)
if err != nil {
	result.Fail(c, result.CodeParamError, "参数错误")
	return
}
```
- **模拟场景**:
  - 输入: `GET /api/comment/target_user_comments?target_user_id=42&page=1&size=15`，目标用户无评论
  - Java 行为: `{"success":true,"code":0,"msg":"","data":[]}`
  - Go 行为: `{"success":false,"code":1,"msg":"参数错误","data":null}`
- **预期行为**: 目标用户评论接口应继续接受 `target_user_id`，并保持基线返回格式
- **影响面**: `/api/comment/target_user_comments`

#### DIFF-CMT-09: 评论点赞接口不再对非法状态返回失败

- **等级**: P2
- **分类**: 业务逻辑
- **Java 证据**: `theme/src/main/java/com/jb/theme/service/impl/CommentServiceImpl.java:124-142`
```java
if(commentLikeExists(commentId, userId)) return;
myIncCommentLike(commentId,1);
Assert.isTrue(updateResult.getModifiedCount() > 0, "点赞失败");

Assert.isTrue(commentLikeExists(commentId,userId), "还没有对该评论进行点赞");
```
- **Go 证据**: `internal/comment/service.go:128-160`
```go
if res.ModifiedCount > 0 || res.UpsertedCount > 0 {
	if err := s.incCommentLike(ctx, commentID, 1); err != nil {
		s.logger.Warn("increase comment like num failed", zap.Error(err), zap.String("commentID", commentID))
	}
}
```
- **模拟场景**:
  - 输入 A: `POST /api/comment_like/507f1f77bcf86cd799439011`，该评论不存在
  - Java 行为: `{"success":false,"code":400,"msg":"点赞失败","data":null}`
  - Go 行为: `{"success":true,"code":200,"msg":"成功","data":null}`
  - 输入 B: `DELETE /api/comment_like/c1`，当前用户从未点赞过 `c1`
  - Java 行为: `{"success":false,"code":400,"msg":"还没有对该评论进行点赞","data":null}`
  - Go 行为: `{"success":true,"code":200,"msg":"成功","data":null}`
- **预期行为**: 点赞/取消点赞在目标评论不存在或状态不合法时，应继续返回失败
- **影响面**: `/api/comment_like/{comment_id}`

#### DIFF-CMT-10: 内容安全不通过时，Java 不落库，Go 会保留一条 `hasCheck=false` 评论

- **等级**: P2
- **分类**: 业务逻辑
- **Java 证据**: `theme/src/main/java/com/jb/theme/mq/consumer/AddCommentConsumer.java:79-93`
```java
MsgSecCheckResp checkResp = wxHelperService.check(comment.getComment(), MsgSecCheckReq.CHECK_COMMENT, operatorOpenId);
if (!checkPass(checkResp)) {
    return true;
}
comment.setComment(checkResp.getFilterContent());
Comment one = commentService.saveOne(comment);
```
- **Go 证据**: `internal/comment/service.go:69-86`，`internal/mq/consumer_topic_comment.go:136-143`
```go
res, err := s.commentColl().InsertOne(ctx, cmt)
```
```go
checkResult, err := c.wxClient.MsgSecCheck(ctx, cmt.Comment, cmt.User.UserID)
if isRisky(checkResult.Suggest) {
	_ = c.wxClient.SendSubscribeMsg(ctx, cmt.User.UserID, "您的评论未通过审核", cmt.Comment)
	return nil
}
```
- **模拟场景**:
  - 输入: `POST /api/comment/t1`，评论内容命中微信内容安全拒绝
  - Java 行为: `campus_comment` 中不会新增该评论文档，`campus_topic.commentNum` 不变
  - Go 行为: `campus_comment` 中会保留一条 `{"comment":"原文","hasCheck":false,...}` 文档，`campus_topic.commentNum` 不变
- **预期行为**: 内容安全未通过时，Go 的持久化结果应与 Java 一致，不能额外残留未通过评论
- **影响面**: `/api/comment/{topic_id}`，以及后续所有依赖 `campus_comment` 原始数据的链路

#### DIFF-CMT-11: `/api/report_comment` 的请求体和成功返回体都变了

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `theme/src/main/java/com/jb/theme/controller/ReportCommentController.java:31-38`，`theme-entity/src/main/java/com/jb/themeentity/entity/ReportComment.java:28-40`
```java
public Result<?> add(@RequestBody @Validated ReportComment reportComment) {
    commentService.existed(reportComment.getCommentId());
    reportComment.setCreatedTime(new Date());
    reportComment.setReportUserId(s);
    ReportComment res = reportCommentDao.save(reportComment);
    return R.data(res);
}
```
- **Go 证据**: `internal/other/report.go:13-17`，`internal/other/report.go:21-32`
```go
var req struct {
	CommentID string `json:"commentId" binding:"required"`
	TopicID   string `json:"topicId" binding:"required"`
	Reason    string `json:"reason" binding:"required"`
}
result.Success(c, nil)
```
- **模拟场景**:
  - 输入: `POST /api/report_comment`，body 为 `{"commentId":"c1","reportContent":"辱骂"}`
  - Java 行为: `{"success":true,"code":0,"msg":"","data":{"id":"r1","commentId":"c1","reportContent":"辱骂","reportUserId":"100","createdTime":"2026-03-26T10:00:00Z","hasHandle":false,"handlerContent":"","handlerUserId":"","handlerTime":""}}`
  - Go 行为: `{"success":false,"code":1,"msg":"参数错误","data":null}`
- **预期行为**: 举报评论接口应继续接受 `reportContent`，并在成功时返回已创建的举报对象
- **影响面**: `/api/report_comment`

#### DIFF-CMT-12: `/admin/report_comment/{id}` 处理举报接口改成了另一套协议

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `theme/src/main/java/com/jb/theme/controller/admin/AdmReportCommentController.java:44-57`，`theme/src/main/java/com/jb/theme/dto/admin/ReportCommentHandleDTO.java:15-18`
```java
public Result<?> edit(@Validated @RequestBody ReportCommentHandleDTO dto, @PathVariable("id")String id)
.set("handlerUserId", handler)
.set("handlerTime", new Date())
.set("handlerContent", dto.getHandlerContent())
.set("hasHandle", true);
```
- **Go 证据**: `internal/other/report_admin.go:9-18`，`internal/other/model_req.go:15-17`，`internal/other/service_report.go:30-39`
```go
type ReportReviewReq struct {
	Status int `json:"status" binding:"required"`
}
_, err = s.mongoDB.Collection("campus_report_comment").UpdateByID(ctx, oid, bson.M{"$set": bson.M{"status": status}})
```
- **模拟场景**:
  - 输入: `PUT /admin/report_comment/r1`，body 为 `{"handlerContent":"证据不足，驳回"}`
  - Java 行为: `{"success":true,"code":200,"msg":"成功","data":null}`，并写入 `handlerUserId/handlerTime/handlerContent/hasHandle=true`
  - Go 行为: `{"success":false,"code":1,"msg":"参数错误","data":null}`
- **预期行为**: 管理员处理举报应继续使用 `handlerContent` 协议，并写入处理人、处理时间、处理内容和已处理状态
- **影响面**: `/admin/report_comment/{id}`

#### DIFF-CMT-13: `/admin/report_comment/list` 返回的数据模型与排序都不兼容

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `theme/src/main/java/com/jb/theme/controller/admin/AdmReportCommentController.java:64-83`，`theme-entity/src/main/java/com/jb/themeentity/entity/ReportComment.java:28-56`
```java
PageRequest pageRequest = PageRequest.of(page-1, size, Sort.by(Sort.Direction.ASC, "hasHandle"));
cusPage.setData(reportComments);
```
- **Go 证据**: `internal/other/report_admin.go:21-28`，`internal/other/service_report.go:42-50`，`internal/other/model.go:111-119`
```go
return listMongoPage[ReportComment](
	ctx,
	s.mongoDB.Collection("campus_report_comment"),
	bson.M{},
	bson.M{"createdAt": -1},
	page,
	size,
)
```
```go
type ReportComment struct {
	CommentID  string `bson:"commentId" json:"commentId"`
	TopicID    string `bson:"topicId" json:"topicId"`
	ReporterID string `bson:"reporterId" json:"reporterId"`
	Reason     string `bson:"reason" json:"reason"`
	Status     int    `bson:"status" json:"status"`
}
```
- **模拟场景**:
  - 输入: `GET /admin/report_comment/list?page=1&size=15`，库中有一条未处理举报
  - Java 行为: `{"success":true,"code":0,"msg":"","data":{"data":[{"id":"r1","commentId":"c1","reportContent":"辱骂","reportUserId":"100","hasHandle":false,"handlerContent":"","handlerUserId":"","handlerTime":""}],"current":1,"total":1,"size":15}}`
  - Go 行为: `{"success":true,"code":200,"msg":"成功","data":{"data":[{"id":"r1","commentId":"c1","topicId":"t1","reporterId":"100","reason":"辱骂","status":0,"createdAt":"2026-03-26T10:00:00Z"}],"current":1,"total":1,"size":15}}`
- **预期行为**: 举报列表字段名、字段集合和排序语义都应与基线一致
- **影响面**: `/admin/report_comment/list`

#### DIFF-CMT-14: 删除评论时 Go 会额外删除对应举报记录

- **等级**: P2
- **分类**: 业务逻辑
- **Java 证据**: `theme/src/main/java/com/jb/theme/service/impl/CommentServiceImpl.java:85-118`
```java
Update update = new Update()
        .set("hasCheck", false)
        .set("deletedTime", new Date());
topicService.decCommentNum(topicId);
if (one.getRootCmtId() != null && !one.getRootCmtId().equals(Comment.DEFAULT_ROOT)) {
    decCommentNum(one.getRootCmtId());
}
```
- **Go 证据**: `internal/comment/service.go:107-113`，`internal/mq/consumer_user_cleanup.go:169-170`
```go
sendErr := s.producer.SendDeleteComment(ctx, topicID, commentID)
```
```go
_, _ = c.mongoDB.Collection("campus_comment_like").DeleteMany(ctx, bson.M{"commentId": msg.CommentID})
_, _ = c.mongoDB.Collection("campus_report_comment").DeleteMany(ctx, bson.M{"commentId": msg.CommentID})
```
- **模拟场景**:
  - 输入: 评论 `c1` 已被举报一次；随后评论所有者调用 `DELETE /api/comment/t1/c1`；之后管理员请求 `GET /admin/report_comment/list?page=1&size=15`
  - Java 行为: 举报记录仍在列表中
  - Go 行为: 该举报记录被一并删除，列表中消失
- **预期行为**: 删除评论后，已有举报记录应继续保留，不能比基线更早消失
- **影响面**: `/api/comment/{topic_id}/{comment_id}`、`/admin/comment/{topic_id}/{comment_id}`、`/admin/report_comment/list`

#### DIFF-CMT-15: 评论通知写入的类型和值结构与 Java 不同

- **等级**: P0
- **分类**: 中间件行为
- **Java 证据**: `theme/src/main/java/com/jb/theme/service/impl/CommentServiceImpl.java:247-259`，`theme/src/main/java/com/jb/theme/mq/producer/CommentAddNotifyProducer.java:17-31`，`chat/src/main/java/com/jb/chat/entity/Notification.java:32-55`
```java
commentAddNotifyProducer.notify(
        targetUserId, userId, comment.getTopicId(), comment.getId(),
        comment.getComment(), NotifyType.COMMENT_REPLY.name());
```
```java
private String receiverId;
private String senderId;
private String type;
private String content;
private String topicId;
private String commentId;
```
- **Go 证据**: `internal/mq/consumer_topic_comment.go:177-214`，`internal/mq/consumer_course_notify.go:70-76`，`internal/chat/model.go:45-52`
```go
c.sendNotify(ctx, NotifyMsg{
	TargetUserID: parentUserID,
	Type:         "comment",
	Content: map[string]string{
		"topicId": cmt.TopicID, "comment": filteredComment, "commentId": cmt.ID.Hex(),
	},
})
```
```go
notification := bson.M{
	"userId":    msg.TargetUserID,
	"type":      msg.Type,
	"content":   msg.Content,
	"createdAt": time.Now(),
	"isRead":    false,
}
```
- **模拟场景**:
  - 输入: 用户 `100` 回复用户 `200` 在帖子 `t1` 下的评论，内容为 `hi`
  - Java 行为: `campus_notifications` 新增 `{"receiver_id":"200","sender_id":"100","type":"COMMENT_REPLY","content":"hi","topic_id":"t1","comment_id":"c9","created_time":"2026-03-26T10:00:00Z","is_read":false}`
  - Go 行为: `campus_notifications` 新增 `{"userId":"200","type":"comment","content":{"topicId":"t1","comment":"hi","commentId":"c9"},"createdAt":"2026-03-26T10:00:00Z","isRead":false}`
- **预期行为**: 评论触发的通知类型和值结构应继续兼容基线的通知查询接口和 WebSocket 消费方
- **影响面**: `/api/notify`、`/api/notify/{type}`、`/api/notify/{type}/haveUnread`、实时通知推送

### 模块总结

- 活跃端点: 11 个
- Go 已覆盖: 11 个
- P0 差异: 7 个
- P1 差异: 2 个
- P2 差异: 6 个
