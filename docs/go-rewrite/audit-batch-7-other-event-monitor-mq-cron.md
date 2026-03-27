# Batch 7 审计报告：Other + Event + Monitor + MQ + Cron/AOP 模块

审计日期：2026-03-26

审计前提：
- 本次重构切换后，不存在 Java 与 Go 双版本同时部署。
- 旧 Redis 缓存允许在切换前清空，但 MySQL / MongoDB 现网数据需要被 Go 直接读取。
- 主报告仅计入 Go 独立部署场景下必然触发的问题；本轮未发现仅在混部、共享旧缓存或互认旧 token 条件下才成立的问题。

补充说明：
- Java 中已注释掉且不会被 Spring 注册的 `@Scheduled` 任务未计入“活跃定时任务”。
- `ExpAndControllerTimeAop` 里的经验值奖励依赖 `aop_permission` 表运行时数据；仓库未提供初始化数据，本轮未把“具体哪条接口应加经验”计入必须修复项。

## 模块：Other

### 活跃 API 端点清单

| # | 端点 | 方法 | Java 入口 | Go 入口 | 状态 |
|---|------|------|----------|---------|------|
| 1 | `/api/ad/list_level` | GET | `other/src/main/java/com/jb/other/controller/AdController.java:23` | `internal/other/ad.go:11` | ✅ |
| 2 | `/api/notice/list` | GET | `other/src/main/java/com/jb/other/controller/NoticeController.java:35` | `internal/other/notice.go:9` | ✅ |
| 3 | `/api/vote/list` | GET | `other/src/main/java/com/jb/other/controller/VoteController.java:52` | `internal/other/vote.go:12` | ✅ |
| 4 | `/api/vote/draft/{info_id}` | GET | `other/src/main/java/com/jb/other/controller/VoteController.java:61` | `internal/other/vote.go:22` | ✅ |
| 5 | `/api/vote/draft/{info_id}` | PUT | `other/src/main/java/com/jb/other/controller/VoteController.java:79` | `internal/other/vote.go:36` | ✅ |
| 6 | `/api/vote` | POST | `other/src/main/java/com/jb/other/controller/VoteController.java:102` | `internal/other/vote.go:53` | ✅ |
| 7 | `/api/vote/{info_id}` | POST | `other/src/main/java/com/jb/other/controller/VoteController.java:133` | `internal/other/vote.go:66` | ✅ |
| 8 | `/api/vote/vote/{info_id}` | POST | `other/src/main/java/com/jb/other/controller/VoteController.java:151` | `internal/other/vote.go:84` | ✅ |
| 9 | `/api/report_comment` | POST | `theme/src/main/java/com/jb/theme/controller/ReportCommentController.java:31` | `internal/other/report.go:12` | ✅ |
| 10 | `/api/support/{key}` | GET | `theme/src/main/java/com/jb/theme/controller/FrontendSupportController.java:34` | `internal/other/support.go:9` | ✅ |
| 11 | `/api/support/list` | GET | `theme/src/main/java/com/jb/theme/controller/FrontendSupportController.java:47` | `internal/other/support.go:18` | ✅ |
| 12 | `/admin/notice` | POST | `other/src/main/java/com/jb/other/controller/admin/AdmNoticeController.java:28` | `internal/other/notice_admin.go:11` | ✅ |
| 13 | `/admin/notice/{id}` | DELETE | `other/src/main/java/com/jb/other/controller/admin/AdmNoticeController.java:34` | `internal/other/notice_admin.go:23` | ✅ |
| 14 | `/admin/notice/{id}` | PUT | `other/src/main/java/com/jb/other/controller/admin/AdmNoticeController.java:42` | `internal/other/notice_admin.go:40` | ✅ |
| 15 | `/admin/notice/{id}` | GET | `other/src/main/java/com/jb/other/controller/admin/AdmNoticeController.java:53` | `internal/other/notice_admin.go:61` | ✅ |
| 16 | `/admin/notice/list` | GET | `other/src/main/java/com/jb/other/controller/admin/AdmNoticeController.java:62` | `internal/other/notice_admin.go:79` | ✅ |
| 17 | `/admin/ad` | POST | `other/src/main/java/com/jb/other/controller/admin/AdmAdController.java:34` | `internal/other/ad_admin.go:11` | ✅ |
| 18 | `/admin/ad/{id}` | DELETE | `other/src/main/java/com/jb/other/controller/admin/AdmAdController.java:41` | `internal/other/ad_admin.go:23` | ✅ |
| 19 | `/admin/ad/{id}` | PUT | `other/src/main/java/com/jb/other/controller/admin/AdmAdController.java:49` | `internal/other/ad_admin.go:40` | ✅ |
| 20 | `/admin/ad/{id}` | GET | `other/src/main/java/com/jb/other/controller/admin/AdmAdController.java:63` | `internal/other/ad_admin.go:61` | ✅ |
| 21 | `/admin/ad/list` | GET | `other/src/main/java/com/jb/other/controller/admin/AdmAdController.java:72` | `internal/other/ad_admin.go:79` | ✅ |
| 22 | `/admin/sensitive/getAllList` | GET | `theme/src/main/java/com/jb/theme/controller/SensitiveWordController.java:22` | `internal/other/sensitive_admin.go:9` | ✅ |
| 23 | `/admin/sensitive/getByWord` | GET | `theme/src/main/java/com/jb/theme/controller/SensitiveWordController.java:28` | `internal/other/sensitive_admin.go:18` | ✅ |
| 24 | `/admin/sensitive/deleteByWord` | DELETE | `theme/src/main/java/com/jb/theme/controller/SensitiveWordController.java:40` | `internal/other/sensitive_admin.go:27` | ✅ |
| 25 | `/admin/sensitive/batchDelete` | DELETE | `theme/src/main/java/com/jb/theme/controller/SensitiveWordController.java:48` | `internal/other/sensitive_admin.go:35` | ✅ |
| 26 | `/admin/sensitive/add` | POST | `theme/src/main/java/com/jb/theme/controller/SensitiveWordController.java:56` | `internal/other/sensitive_admin.go:47` | ✅ |
| 27 | `/admin/sensitive/batchAdd` | POST | `theme/src/main/java/com/jb/theme/controller/SensitiveWordController.java:64` | `internal/other/sensitive_admin.go:60` | ✅ |
| 28 | `/admin/sensitive/page` | GET | `theme/src/main/java/com/jb/theme/controller/SensitiveWordController.java:72` | `internal/other/sensitive_admin.go:72` | ✅ |
| 29 | `/admin/sensitive/search_like` | GET | `theme/src/main/java/com/jb/theme/controller/SensitiveWordController.java:82` | `internal/other/sensitive_admin.go:82` | ✅ |
| 30 | `/admin/sensitive/update` | PUT | `theme/src/main/java/com/jb/theme/controller/SensitiveWordController.java:90` | `internal/other/sensitive_admin.go:91` | ✅ |
| 31 | `/admin/report_comment/{id}` | PUT | `theme/src/main/java/com/jb/theme/controller/admin/AdmReportCommentController.java:44` | `internal/other/report_admin.go:10` | ✅ |
| 32 | `/admin/report_comment/list` | GET | `theme/src/main/java/com/jb/theme/controller/admin/AdmReportCommentController.java:64` | `internal/other/report_admin.go:22` | ✅ |
| 33 | `/admin/support` | POST | `theme/src/main/java/com/jb/theme/controller/admin/AdmFrontendSupportController.java:35` | `internal/other/support_admin.go:9` | ✅ |
| 34 | `/admin/support` | PUT | `theme/src/main/java/com/jb/theme/controller/admin/AdmFrontendSupportController.java:42` | `internal/other/support_admin.go:22` | ✅ |
| 35 | `/admin/support/{id}` | DELETE | `theme/src/main/java/com/jb/theme/controller/admin/AdmFrontendSupportController.java:59` | `internal/other/support_admin.go:34` | ✅ |
| 36 | `/admin/support/list` | GET | `theme/src/main/java/com/jb/theme/controller/admin/AdmFrontendSupportController.java:70` | `internal/other/support_admin.go:42` | ✅ |
| 37 | `/admin/merchant_theme` | POST | `theme/src/main/java/com/jb/theme/controller/admin/AdmMerchantThemeController.java:27` | `internal/other/merchant_admin.go:9` | ✅ |
| 38 | `/admin/merchant_theme/{id}` | DELETE | `theme/src/main/java/com/jb/theme/controller/admin/AdmMerchantThemeController.java:37` | `internal/other/merchant_admin.go:22` | ✅ |
| 39 | `/admin/merchant_theme/get_all` | GET | `theme/src/main/java/com/jb/theme/controller/admin/AdmMerchantThemeController.java:45` | `internal/other/merchant_admin.go:30` | ✅ |
| 40 | `/admin/task` | POST | `mq/src/main/java/com/jb/mq/controller/admin/TaskController.java:27` | `internal/other/task_admin.go:11` | ✅ |
| 41 | `/admin/task/{id}` | DELETE | `mq/src/main/java/com/jb/mq/controller/admin/TaskController.java:33` | `internal/other/task_admin.go:23` | ✅ |
| 42 | `/admin/task/{id}` | PUT | `mq/src/main/java/com/jb/mq/controller/admin/TaskController.java:41` | `internal/other/task_admin.go:40` | ✅ |
| 43 | `/admin/task/{id}` | GET | `mq/src/main/java/com/jb/mq/controller/admin/TaskController.java:52` | `internal/other/task_admin.go:61` | ✅ |
| 44 | `/admin/task/list` | GET | `mq/src/main/java/com/jb/mq/controller/admin/TaskController.java:61` | `internal/other/task_admin.go:79` | ✅ |

状态说明：
- ✅ 端点存在且路径、方法一致
- ❌ Go 中不存在该端点
- ⚠️ 端点存在但路径或方法不一致

### 差异清单

#### DIFF-OTH-01: `/api/ad/list_level` 把 `size` 排行接口改成了 `level` 过滤接口

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `other/src/main/java/com/jb/other/controller/AdController.java:23-28`
```java
@GetMapping("/list_level")
public Result<?> getListWithLevel(@RequestParam(value = "size", defaultValue = "0") Integer size) {
    if (size == 0) {
        size = customConf.getPageSize();
    }
    return adService.getListSortByLevel(size);
}
```
- **Go 证据**: `internal/other/ad.go:11-18`，`internal/other/service_ad.go:51-58`
```go
func (h *Handler) AdListByLevel(c *gin.Context) {
    level, _ := strconv.Atoi(c.DefaultQuery("level", "0"))
    data, err := h.svc.ListAdByLevel(c.Request.Context(), level)
```
```go
q := s.db.WithContext(ctx).Where("isOk = ?", true)
if level > 0 {
    q = q.Where("level = ?", level)
}
```
- **模拟场景**:
  - 输入: `GET /api/ad/list_level?size=2`，库中存在 3 条有效广告：`[{id:1,level:10},{id:2,level:8},{id:3,level:1}]`
  - Java 行为: `{"success":true,"code":0,"msg":"","data":[{"id":1,"level":10},{"id":2,"level":8}]}`
  - Go 行为: `{"success":true,"code":200,"msg":"成功","data":[{"id":1,"level":10},{"id":2,"level":8},{"id":3,"level":1}]}`
- **预期行为**: 广告位接口应继续按 `size` 返回“按优先级倒序的前 N 条有效广告”。
- **影响面**: `/api/ad/list_level`

#### DIFF-OTH-02: FrontendSupport 读写从 `val/keyDesc` 变成了 `value`，现网 Mongo 文档直接失配

- **等级**: P1
- **分类**: 数据兼容
- **Java 证据**: `theme-entity/src/main/java/com/jb/themeentity/entity/FrontendSupport.java:30-44`，`theme/src/main/java/com/jb/theme/controller/admin/AdmFrontendSupportController.java:43-55`
```java
@Document(collection = "campus_frontend_support")
public class FrontendSupport implements Serializable {
    private String id;
    private String key;
    private String val;
    private String keyDesc;
}
```
```java
one.setVal(dto.getVal());
if(StringUtils.isNotBlank(dto.getKeyDesc())) {
    one.setKeyDesc(dto.getKeyDesc());
}
```
- **Go 证据**: `internal/other/model.go:128-132`，`internal/other/service_merchant_support_task.go:70-84,120-126`
```go
type FrontendSupport struct {
    ID    primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    Key   string             `bson:"key" json:"key"`
    Value interface{}        `bson:"value" json:"value"`
}
```
```go
_, err := s.mongoDB.Collection("campus_frontend_support").UpdateOne(
    ctx, bson.M{"key": support.Key},
    bson.M{"$set": bson.M{"value": support.Value}},
)
```
- **模拟场景**:
  - 输入: 现网已有 Java 写入文档 `{"_id":"660000000000000000000001","key":"checkVersion","val":"1.2.3","keyDesc":"版本号"}`
  - Java 行为: `GET /api/support/checkVersion` 返回 `{"success":true,"code":0,"msg":"","data":{"id":"660000000000000000000001","key":"checkVersion","val":"1.2.3","keyDesc":"版本号"}}`
  - Go 行为: 同一请求会把旧文档解码成 `{"success":true,"code":0,"msg":"","data":{"id":"660000000000000000000001","key":"checkVersion","value":null}}`，并且后续 `PUT /admin/support` 会把新值写到 `value` 字段而不是更新 `val`
- **预期行为**: 前端支持配置的读写应继续兼容现网 `campus_frontend_support` 文档，并保持公开字段 `val`、`keyDesc`。
- **影响面**: `/api/support/{key}`、`/api/support/list`、`/admin/support`

#### DIFF-OTH-03: `/admin/task` 仍由 Java 客户端提交 `{name}` 时，Go 会写出完全不同的任务记录

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `mq/src/main/java/com/jb/mq/dto/admin/AddTaskDTO.java:11-12`，`mq/src/main/java/com/jb/mq/controller/admin/TaskController.java:27-29`
```java
public class AddTaskDTO {
    private String name;
}
```
```java
@PostMapping
public Result<?> add(@RequestBody @Validated AddTaskDTO addTaskDTO) {
    return taskService.add(addTaskDTO);
}
```
- **Go 证据**: `internal/other/task_admin.go:11-20`，`internal/other/model.go:96-104`
```go
func (h *AdminHandler) TaskAdd(c *gin.Context) {
    var req Task
    if !result.BindJSON(c, &req) {
        return
    }
```
```go
type Task struct {
    Status int    `json:"status"`
    Detail string `json:"detail"`
    Parent int64  `json:"parent"`
    Func   string `json:"func"`
}
```
- **模拟场景**:
  - 输入: `POST /admin/task`，请求体 `{"name":"帖子搜索消费者"}`
  - Java 行为: 请求体会正常绑定到 `AddTaskDTO.name`，接口继续沿 `TaskController.add -> TaskServiceImpl.add(AddTaskDTO)` 链路处理该 `name`
  - Go 行为: `name` 字段不会绑定到 `Task`，最终写入的是零值任务，例如 `campus_task {status:0, detail:"", parent:0, func:""}`
- **预期行为**: 管理端任务接口应继续接受现网公开的 `{name}` 请求体，并保持对应的数据写入语义。
- **影响面**: `/admin/task` `POST`，以及同类 `PUT /admin/task/{id}` 调用

#### DIFF-OTH-04: 投票列表与选项返回体把 `0/1` 字段改成了布尔值，并把 `createdBy` 改成了 `userId`

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `other/src/main/java/com/jb/other/controller/VoteController.java:52-58`，`other/src/main/java/com/jb/other/entity/VoteInfo.java:47-78`，`other/src/main/java/com/jb/other/entity/VoteOption.java:57-59`
```java
return R.data(voteInfoPageUtil.page(voteInfoDao, page, size, queryWrapper));
```
```java
private Integer accessDraft;
private Long createdBy;
```
```java
private Integer isOk;
```
- **Go 证据**: `internal/other/service_vote.go:11-20`，`internal/other/model.go:49-77`
```go
func (s *Service) ListVotes(ctx context.Context, page, size int) (*result.PageResult[VoteInfo], error) {
    ...
    return result.NewPage(list, total, page, size), nil
}
```
```go
AccessDraft bool  `json:"accessDraft"`
UserID      int64 `json:"userId"`
IsOk        bool  `json:"isOk"`
```
- **模拟场景**:
  - 输入: `GET /api/vote/list?page=1&size=1`，数据库存在 `campus_vote_info {id:7,title:"午饭",access_draft:1,created_by:101}`
  - Java 行为: `{"success":true,"code":0,"msg":"","data":{"records":[{"id":7,"title":"午饭","accessDraft":1,"createdBy":101}]}}`
  - Go 行为: `{"success":true,"code":200,"msg":"成功","data":{"records":[{"id":7,"title":"午饭","accessDraft":true,"userId":101}]}}`
- **预期行为**: 投票相关接口应继续返回现网已经公开的字段名与数值语义，包括 `accessDraft` / `isOk` 的 `0/1` 表示以及 `createdBy` 字段。
- **影响面**: `/api/vote/list`、`/api/vote/draft/{info_id}`

#### DIFF-OTH-05: `/api/vote` 创建接口不再接受 Java 现网的 `{info, options}` 请求体

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `other/src/main/java/com/jb/other/dto/VoteInfoDto.java:19-24`，`other/src/main/java/com/jb/other/controller/VoteController.java:102-129`
```java
public class VoteInfoDto implements Serializable {
  private VoteInfo info;
  private List<VoteOption> options;
}
```
```java
VoteInfo info = voteInfoDto.getInfo();
List<VoteOption> options = voteInfoDto.getOptions();
voteInfoDao.save(info);
voteOptionDao.saveBatch(options);
```
- **Go 证据**: `internal/other/vote.go:53-63`，`internal/other/service_vote.go:46-50`
```go
func (h *Handler) VoteCreate(c *gin.Context) {
    var req VoteInfo
    if !result.BindJSON(c, &req) {
        return
    }
```
```go
func (s *Service) CreateVoteInfo(ctx context.Context, info *VoteInfo) error {
    return s.db.WithContext(ctx).Create(info).Error
}
```
- **模拟场景**:
  - 输入: `POST /api/vote`，请求体 `{"info":{"title":"午饭","accessDraft":0,"accessEndTime":"2026-03-27 12:00","voteStart":"2026-03-27 12:30","voteEnd":"2026-03-27 13:30","mode":1,"optionType":1},"options":[{"voteText":"米饭"},{"voteText":"面"}]}`
  - Java 行为: 成功写入 1 条 `campus_vote_info` 与 2 条 `campus_vote_option`
  - Go 行为: `info/options` 顶层字段不会绑定到 `VoteInfo`；插入载荷退化成零值 `VoteInfo`，最终返回数据库错误对应的通用失败响应
- **预期行为**: 投票创建接口应继续接受现网的 `{info, options}` 结构，并在同一次请求中落库投票信息和初始选项。
- **影响面**: `/api/vote`

#### DIFF-OTH-06: `/api/vote/draft/{info_id}` 丢失了“仅创建者可见”和 `is_ok` 过滤

- **等级**: P2
- **分类**: 业务逻辑
- **Java 证据**: `other/src/main/java/com/jb/other/controller/VoteController.java:61-76`
```java
VoteInfo info = voteInfoDao.getById(infoId);
Long userId = ThreadLocalUtil.getUserId();
if(ObjectUtils.notEqual(info.getCreatedBy(), userId)) {
    return R.fail(RC.ERROR_NOT_EXISTED);
}
...
.eq(VoteOption::getIsOk, isOk);
```
- **Go 证据**: `internal/other/vote.go:22-33`，`internal/other/service_vote.go:23-28`
```go
data, err := h.svc.GetVoteOptions(c.Request.Context(), infoID)
```
```go
if err := s.db.WithContext(ctx).
    Where("voteInfoId = ?", voteInfoID).
    Order("id ASC").Find(&options).Error; err != nil {
```
- **模拟场景**:
  - 输入: 投票 `info_id=7` 的创建者是用户 `101`；选项有 `[{id:1,isOk:1},{id:2,isOk:0}]`；用户 `202` 请求 `GET /api/vote/draft/7?is_ok=0`
  - Java 行为: `{"success":false,"code":3,"msg":"资源不存在","data":null}`
  - Go 行为: `{"success":true,"code":200,"msg":"成功","data":[{"id":1,"isOk":true},{"id":2,"isOk":false}]}`
- **预期行为**: 投稿选项列表应继续只对投票创建者开放，并按 `is_ok` 返回对应状态的选项。
- **影响面**: `/api/vote/draft/{info_id}` `GET`

#### DIFF-OTH-07: `/api/vote/vote/{info_id}` 从“追加当天投票记录”变成了“覆盖旧投票记录”

- **等级**: P1
- **分类**: 数据兼容
- **Java 证据**: `other/src/main/java/com/jb/other/controller/VoteController.java:179-190`，`other/src/main/resources/mapper/VoteAnsMapper.xml:4-10`
```java
VoteAns ans = VoteAns.builder()
    .voteUserId(userId)
    .voteInfoId(infoId)
    .voteDate(today)
    .voteOptionId(r)
    .build();
```
```xml
insert ignore into `campus_vote_ans`
(vote_info_id, vote_option_id, vote_date, vote_user_id)
```
- **Go 证据**: `internal/other/service_vote.go:60-88`
```go
if err := tx.Where("voteInfoId = ? AND voteUserId = ?", voteInfoID, voteUserID).
    Delete(&VoteAns{}).Error; err != nil {
    ...
}
```
- **模拟场景**:
  - 输入: 单选投票 `info_id=7`，用户 `101` 先 `POST /api/vote/vote/7` 提交 `[11]`，同一天再次提交 `[22]`
  - Java 行为: `campus_vote_ans` 最终保留两条记录：`(7,11,2026-03-26,101)` 与 `(7,22,2026-03-26,101)`
  - Go 行为: 第二次提交前先删除 `voteInfoId=7 AND voteUserId=101` 的旧记录，最终只剩 `(7,22,2026-03-26,101)`
- **预期行为**: 重复投票时的落库结果应保持与现网一致，避免把已有投票历史改成覆盖语义。
- **影响面**: `/api/vote/vote/{info_id}`

#### DIFF-OTH-08: `/admin/merchant_theme` 新增操作不再幂等，重复提交会生成重复文档

- **等级**: P2
- **分类**: 业务逻辑
- **Java 证据**: `theme/src/main/java/com/jb/theme/controller/admin/AdmMerchantThemeController.java:27-34`
```java
themeService.existed(dto.getThemeId());
MerchantTheme one = merchantThemeService.getOneWithThemeId(dto.getThemeId());
if( one != null ) {
    return R.data(one);
}
return R.data(merchantThemeService.saveOne(dto));
```
- **Go 证据**: `internal/other/merchant_admin.go:9-19`，`internal/other/service_merchant_support_task.go:16-26`
```go
id, err := h.svc.AddMerchantTheme(c.Request.Context(), req.ThemeID)
```
```go
doc := MerchantTheme{ThemeID: themeID}
res, err := s.mongoDB.Collection("campus_merchant_theme").InsertOne(ctx, doc)
```
- **模拟场景**:
  - 输入: `POST /admin/merchant_theme`，请求体 `{"themeId":"theme_merchant_1"}`，集合里已存在 `{"_id":"660...001","themeId":"theme_merchant_1"}`
  - Java 行为: `{"success":true,"code":0,"msg":"","data":{"id":"660...001","themeId":"theme_merchant_1"}}`
  - Go 行为: `{"success":true,"code":200,"msg":"成功","data":"660...002"}`，并新增第二条同 `themeId` 文档
- **预期行为**: 商家主题新增操作应继续对相同 `themeId` 保持幂等。
- **影响面**: `/admin/merchant_theme` `POST`

### 模块总结

- 活跃端点: 44 个
- Go 已覆盖: 44 个
- P0 差异: 4 个
- P1 差异: 2 个
- P2 差异: 2 个

## 模块：Event

### 活跃 API 端点清单

| # | 端点 | 方法 | Java 入口 | Go 入口 | 状态 |
|---|------|------|----------|---------|------|
| 1 | `/api/event/` | POST | `front-event/src/main/java/com/jb/event/controller/EventController.java:35` | `internal/event/handler.go:15` | ⚠️ |
| 2 | `/admin/event/{id}` | DELETE | `front-event/src/main/java/com/jb/event/controller/admin/AdmEventDataController.java:41` | `internal/event/admin.go:19` | ✅ |
| 3 | `/admin/event/{id}` | PUT | `front-event/src/main/java/com/jb/event/controller/admin/AdmEventDataController.java:52` | `internal/event/admin.go:36` | ✅ |
| 4 | `/admin/event/{id}` | GET | `front-event/src/main/java/com/jb/event/controller/admin/AdmEventDataController.java:66` | `internal/event/admin.go:57` | ✅ |
| 5 | `/admin/event/list` | GET | `front-event/src/main/java/com/jb/event/controller/admin/AdmEventDataController.java:75` | `internal/event/admin.go:75` | ✅ |

状态说明：
- ✅ 端点存在且路径、方法一致
- ❌ Go 中不存在该端点
- ⚠️ 端点存在但路径或方法不一致

### 差异清单

#### DIFF-EVT-01: `/api/event/` 在 Go 中注册成了 `/api/event`，POST 需要额外重定向

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `front-event/src/main/java/com/jb/event/controller/EventController.java:20-36`
```java
@RequestMapping("/api/event")
...
@PostMapping("/")
public Result<?> add(@RequestBody @Validated Event event) {
```
- **Go 证据**: `cmd/ecampus/routes.go:150`，`internal/event/handler.go:15-24`
```go
api.POST("/event", handlers.Event.Add)
```
- **模拟场景**:
  - 输入: `POST /api/event/`，请求体 `{"eventType":"open","eventInfo":"home","eventContent":"{}"}`
  - Java 行为: 直接返回 `200` 成功响应
  - Go 行为: Gin 默认先返回 `307 Temporary Redirect` 到 `/api/event`
- **预期行为**: 埋点接口的路径应与现网一致，客户端不应因为尾斜杠差异额外经历重定向。
- **影响面**: `/api/event/`

#### DIFF-EVT-02: `/api/event/` 不再由服务端注入当前登录用户，`userId` 写入结果从字符串变成了客户端可控的整型

- **等级**: P1
- **分类**: 数据兼容
- **Java 证据**: `front-event/src/main/java/com/jb/event/controller/EventController.java:36-41`，`front-event/src/main/java/com/jb/event/entity/Event.java:42-46`
```java
event.setUserId(ThreadLocalUtil.getUserId().toString());
event.setTriggerTime(new Timestamp(System.currentTimeMillis()));
if (eventDao.save(event)) {
```
```java
private String userId;
private Date triggerTime;
```
- **Go 证据**: `internal/event/handler.go:15-24`，`internal/event/service.go:41-50`，`internal/event/model.go:5-11`
```go
var req Event
if !result.BindJSON(c, &req) {
    return
}
if err := h.svc.AddEvent(c.Request.Context(), &req); err != nil {
```
```go
type Event struct {
    ...
    UserID      int64     `json:"userId"`
    TriggerTime time.Time `json:"triggerTime"`
}
```
- **模拟场景**:
  - 输入: token 用户 `101` 调用 `POST /api/event/`，请求体 `{"eventType":"open","eventInfo":"home","eventContent":"{\"tab\":\"hot\"}"}`
  - Java 行为: 写入 `event_data (event_type,event_info,event_content,user_id,trigger_time) = ('open','home','{\"tab\":\"hot\"}','101','2026-03-26 12:00:00')`
  - Go 行为: 写入 Redis `campus:event:EVENT_KEY` 的 JSON 负载里 `userId` 为 `0`，除非客户端自行伪造该字段
- **预期行为**: 埋点记录的用户身份与触发时间应由服务端从请求上下文生成，而不是依赖客户端传值。
- **影响面**: `/api/event/`，以及后续所有事件查询和统计

#### DIFF-EVT-03: `/api/event/` 从“成功即入库”变成了“先写 Redis，最多延迟 15 分钟才入库”

- **等级**: P2
- **分类**: 业务逻辑
- **Java 证据**: `front-event/src/main/java/com/jb/event/controller/EventController.java:36-44`
```java
event.setUserId(ThreadLocalUtil.getUserId().toString());
event.setTriggerTime(new Timestamp(System.currentTimeMillis()));
if (eventDao.save(event)) {
    return R.success().msg("添加成功");
}
```
- **Go 证据**: `internal/event/service.go:41-50`，`internal/cron/scheduler.go:60-67`，`internal/cron/event_flush.go:34-68`
```go
if err := s.redis.LPush(ctx, rediskey.EventKey, string(data)).Err(); err != nil {
    return fmt.Errorf("push event to redis: %w", err)
}
```
```go
_, err = s.cron.AddFunc("0 */15 * * * *", func() {
    if runErr := s.event.Run(context.Background()); runErr != nil {
```
- **模拟场景**:
  - 输入: 用户 `101` 于 `2026-03-26 10:01:00` 成功调用 `POST /api/event/`，管理员立刻调用 `GET /admin/event/list?prev_id=1&size=10&start_time=2026-03-26%2000:00:00`
  - Java 行为: 新事件已在 `event_data` 中，可被管理端列表查询到
  - Go 行为: 事件仍停留在 Redis `campus:event:EVENT_KEY`，在下一次 `0 */15 * * * *` flush 前不会出现在 `event_data`
- **预期行为**: 埋点接口成功返回后，数据应立即进入管理端查询口径。
- **影响面**: `/api/event/`、`/admin/event/{id}`、`/admin/event/list`

#### DIFF-EVT-04: `/admin/event/list` 的查询参数和返回结构整体换了协议

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `front-event/src/main/java/com/jb/event/controller/admin/AdmEventDataController.java:75-112`
```java
@RequestParam(value = "prev_id", defaultValue = "1" ) Integer prevId,
@RequestParam(value = "start_time", defaultValue = "2023-09-18 12:00:00") Date triggerTime,
@RequestParam(value = "user_id", defaultValue = "") String userId,
@RequestParam(value = "event_type", defaultValue = "") String eventType,
@RequestParam(value = "key_word", defaultValue = "") String keyWord
```
```java
res.put("data", list);
res.put("total", count);
return R.data(res);
```
- **Go 证据**: `internal/event/admin.go:75-83`，`internal/event/service.go:77-98`
```go
page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
size, _ := strconv.Atoi(c.DefaultQuery("size", "15"))
data, err := h.svc.ListEvents(c.Request.Context(), page, size, c.Query("eventType"))
```
```go
return result.NewPage(list, total, page, size), nil
```
- **模拟场景**:
  - 输入: `GET /admin/event/list?prev_id=100&size=2&start_time=2026-03-26%2000:00:00&user_id=101&event_type=open&key_word=首页`
  - Java 行为: 按 `prev_id/start_time/user_id/event_type/key_word` 过滤，返回 `{"success":true,"code":0,"msg":"","data":{"data":[...],"total":1}}`
  - Go 行为: `prev_id/start_time/user_id/event_type/key_word` 全部失效，只按 `page/size/eventType` 工作；本请求会返回 `{"success":true,"code":200,"msg":"成功","data":{"records":[...],"current":1,"size":2,"total":N,"pages":M}}`
- **预期行为**: 事件管理列表应继续支持现网已公开的筛选参数与 `{data,total}` 返回结构。
- **影响面**: `/admin/event/list`

### 模块总结

- 活跃端点: 5 个
- Go 已覆盖: 5 个
- P0 差异: 2 个
- P1 差异: 1 个
- P2 差异: 1 个

## 模块：Monitor

### 活跃 API 端点清单

| # | 端点 | 方法 | Java 入口 | Go 入口 | 状态 |
|---|------|------|----------|---------|------|
| 1 | `/admin/local_cache/all_key` | GET | `monitor/src/main/java/com/jb/monitor/controller/admin/LocalCacheController.java:31` | `internal/monitor/admin.go:15` | ✅ |
| 2 | `/admin/local_cache/stats` | GET | `monitor/src/main/java/com/jb/monitor/controller/admin/LocalCacheController.java:38` | `internal/monitor/admin.go:24` | ✅ |

状态说明：
- ✅ 端点存在且路径、方法一致
- ❌ Go 中不存在该端点
- ⚠️ 端点存在但路径或方法不一致

### 差异清单

按当前口径，本模块无可报告差异。

说明：
- `local_cache` 现已明确改为 Redis 视角，不再按 Java 的 Caffeine 本地缓存语义判定兼容性问题。

### 模块总结

- 活跃端点: 2 个
- Go 已覆盖: 2 个
- P0 差异: 0 个
- P1 差异: 0 个
- P2 差异: 0 个

## 模块：MQ

### 活跃 API 端点清单

该模块无自有 Controller。以下列出本轮实际用于验证 MQ 链路的活跃触发端点。

| # | 端点 | 方法 | Java 入口 | Go 入口 | 状态 |
|---|------|------|----------|---------|------|
| 1 | `/api/user` | PUT | `user/src/main/java/com/jb/user/controller/UserController.java:78` | `internal/user/handler.go:134` | ✅ |
| 2 | `/api/topic/{id}` | DELETE | `theme/src/main/java/com/jb/theme/controller/TopicController.java:135` | `internal/topic/handler.go:34` | ✅ |

状态说明：
- ✅ 端点存在且路径、方法一致
- ❌ Go 中不存在该端点
- ⚠️ 端点存在但路径或方法不一致

### 差异清单

#### DIFF-MQ-01: 用户资料变更后，Go 不再同步更新回复场景中的 `parent.*` 快照

- **等级**: P1
- **分类**: 数据兼容
- **Java 证据**: `user/src/main/java/com/jb/user/service/impl/UserServiceImpl.java:321-357`，`mq/src/main/java/com/jb/mq/consumer/UpdateCommentUserConsumer.java:41-70`
```java
Query query1 = Query.query(Criteria.where("parent.userId").is(user.getUserId()));
Update update1 = new Update();
update1.set("parent.nickName", user.getNickName());
update1.set("parent.avatar", user.getAvatar());
...
mongoTemplate.updateMulti(query1, update1, Comment.class);
```
- **Go 证据**: `internal/user/service.go:138-153`，`internal/mq/consumer_user_cleanup.go:53-89`
```go
commentMsg := mq.CommentUserUpdateMsg(msg)
if err := s.producer.SendUpdateCommentUser(ctx, commentMsg); err != nil {
```
```go
if _, err := c.mongoDB.Collection("campus_comment").UpdateMany(
    ctx,
    bson.M{"user.userId": msg.UserID},
    bson.M{"$set": userSet},
); err != nil {
```
- **模拟场景**:
  - 输入: 用户 `101` 先被别人回复过，因此某条 `campus_comment` 文档里存在 `parent.userId="101", parent.nickName="旧名"`；随后用户 `101` 调用 `PUT /api/user` 把昵称改为 `新名`
  - Java 行为: `campus_comment` 中 `user.userId="101"` 和 `parent.userId="101"` 的两类文档都会被更新，回复列表里的父评论作者昵称变成 `新名`
  - Go 行为: 只更新 `user.userId="101"` 的文档；已有回复里的 `parent.nickName` 继续保留 `旧名`
- **预期行为**: 用户资料同步链路应同时更新评论作者快照和被回复父评论快照。
- **影响面**: `/api/user`，以及所有评论列表/回复展示场景

#### DIFF-MQ-02: 删帖后的 MQ 清理从“保留软删除记录”变成了“直接硬删帖子和评论”

- **等级**: P1
- **分类**: 数据兼容
- **Java 证据**: `theme/src/main/java/com/jb/theme/service/impl/TopicServiceImpl.java:103-114`，`theme/src/main/java/com/jb/theme/mq/consumer/DeleteTopicConsumer.java:51-62`
```java
Update update = new Update().set("hasCheck", false);
mongoTemplate.updateFirst(new Query(Criteria.where("_id").is(id)), update, Topic.class);
```
```java
update.pull("topicIds", topicId);
mongoTemplate.updateMulti(query, update, TopicCollection.class);
mongoTemplate.updateMulti(query, update, TopicLike.class);
mongoTemplate.updateMulti(Query.query(Criteria.where("topicId").is(topicId)), updateCheck, Comment.class);
mongoTemplate.remove(Query.query(Criteria.where("topicId").is(topicId)), TopicSearch.class);
```
- **Go 证据**: `internal/topic/service.go:99-124`，`internal/mq/consumer_user_cleanup.go:92-137`
```go
res, err := s.topicColl().UpdateOne(ctx, filter, bson.M{"$set": bson.M{"hasCheck": false}})
...
if err := s.producer.SendDeleteTopic(ctx, mq.TopicDeleteMsg{TopicID: topicID}); err != nil {
```
```go
if _, err := c.mongoDB.Collection("campus_topic").DeleteOne(ctx, bson.M{"_id": topicOID}); err != nil {
    return fmt.Errorf("delete topic: %w", err)
}
...
if _, err := c.mongoDB.Collection("campus_comment").DeleteMany(ctx, bson.M{"topicId": msg.TopicID}); err != nil {
```
- **模拟场景**:
  - 输入: 作者删除帖子 `topicId=660000000000000000000001`，该帖子下有 2 条评论和 1 条搜索索引
  - Java 行为:
    - `campus_topic` 保留原文档，仅把 `hasCheck` 置为 `false`
    - `campus_comment` 保留原评论文档，仅把 `hasCheck` 置为 `false`
    - `campus_topic_search` 删除对应索引，点赞/收藏数组中移除该 `topicId`
  - Go 行为:
    - `campus_topic` 直接删除 `_id=660000000000000000000001`
    - `campus_comment` 直接删除 `topicId=660000000000000000000001` 的全部评论
    - `campus_comment_like` 也被一并物理删除
- **预期行为**: 删帖后的异步清理应保持现网的软删除结果，不应把仍需保留的帖子/评论记录直接物理删除。
- **影响面**: `/api/topic/{id}`、`/admin/topic/{topic_id}`，以及所有依赖软删除记录的后台排查与数据一致性场景

### 模块总结

- 活跃端点: 2 个
- Go 已覆盖: 2 个
- P0 差异: 0 个
- P1 差异: 2 个
- P2 差异: 0 个

## 模块：Cron + AOP

### 活跃 API 端点清单

该模块无自有 Controller。以下列出本轮用于验证 AOP/定时链路的代表性活跃端点；其中控制器耗时写入实际覆盖全部 Controller 方法。

| # | 端点 | 方法 | Java 入口 | Go 入口 | 状态 |
|---|------|------|----------|---------|------|
| 1 | `/api/report_comment` | POST | `theme/src/main/java/com/jb/theme/controller/ReportCommentController.java:31` | `internal/other/report.go:12` | ✅ |
| 2 | `/api/vote` | POST | `other/src/main/java/com/jb/other/controller/VoteController.java:102` | `internal/other/vote.go:53` | ✅ |
| 3 | `/api/topic` | POST | `theme/src/main/java/com/jb/theme/controller/TopicController.java:96` | `internal/topic/handler.go:21` | ✅ |

状态说明：
- ✅ 端点存在且路径、方法一致
- ❌ Go 中不存在该端点
- ⚠️ 端点存在但路径或方法不一致

### 差异清单

#### DIFF-CRN-01: Go 完全缺失 `controller_time` 采集与 10 分钟批量落库链路

- **等级**: P1
- **分类**: 数据兼容
- **Java 证据**: `aop/src/main/java/com/jb/aop/user/ExpAndControllerTimeAop.java:47-90`，`level/src/main/java/com/jb/level/scheduleTask/InsertControllerTimeTask.java:30-45`
```java
ControllerTime controllerTime = new ControllerTime(
    null, controller, endTime - startTime, success,
    new Timestamp(System.currentTimeMillis())
);
controllerTimeDao.insertControllerTime(controllerTime);
```
```java
@Scheduled(cron = "* 0/10 * * * ?")
public void insertAllControllerTimeFromRedis() {
    List<ControllerTime> controllerTimeList = controllerTimeRedisService.getAndDeleteAll();
    controllerTimeMapper.saveBatch(list);
}
```
- **Go 证据**: `internal/middleware/log.go:15-45`
```go
return func(c *gin.Context) {
    start := time.Now()
    ...
    logger.Info("http request", fields...)
}
```
- **模拟场景**:
  - 输入: `POST /api/report_comment` 成功处理一次
  - Java 行为: 先写 Redis `campus:controllerTime:ReportCommentController.add`；在下一次 10 分钟批量任务后写入 `controller_time {controller:"ReportCommentController.add", success:1, time_cost:...}`
  - Go 行为: 只写 zap 日志；不会产生 `campus:controllerTime:*` Redis key，也不会向 `controller_time` 表插入任何记录
- **预期行为**: 控制器耗时采集数据应继续进入 Redis 与 MySQL，供现有监控/排障链路使用。
- **影响面**: 所有 Controller 端点；本条用 `/api/report_comment` 作为代表性验证场景

#### DIFF-CRN-02: `AuthPermissionAOP` 缺失后，未认证用户可以直接调用原本应被统一拦截的非 GET 接口

- **等级**: P2
- **分类**: 中间件行为
- **Java 证据**: `aop/src/main/java/com/jb/aop/user/AuthPermissionAOP.java:47-68`，`theme/src/main/java/com/jb/theme/controller/ReportCommentController.java:31-38`
```java
if(method.equalsIgnoreCase("GET") || method.equalsIgnoreCase("OPTIONS")) {
    return;
}
...
if(!one.getStuIsCheck()) {
    throw new RuntimeException("当前接口需要进行认证后，方可使用");
}
```
- **Go 证据**: `internal/other/report.go:12-29`，`internal/other/vote.go:53-99`
```go
report, err := h.svc.CreateReportComment(c.Request.Context(), &ReportComment{...})
```
```go
req.UserID = middleware.GetUserID(c)
if err := h.svc.CreateVoteInfo(c.Request.Context(), &req); err != nil {
```
- **模拟场景**:
  - 输入: 已登录但 `stuIsCheck=false` 的用户调用 `POST /api/report_comment`，请求体 `{"commentId":"660000000000000000000010","reportContent":"spam"}`
  - Java 行为: 请求在进入 Controller 前被 AOP 拦截，不会写入 `campus_report_comment`；错误消息为 `当前接口需要进行认证后，方可使用`，HTTP 包装 **未验证，需人工确认**
  - Go 行为: 请求进入 Handler，并向 `campus_report_comment` 插入新举报文档
- **预期行为**: 统一认证拦截应继续作用于非 GET/OPTIONS 的受保护接口，未认证用户不能绕过该限制写业务数据。
- **影响面**: 大量 POST/PUT/DELETE `/api/**` 与 `/admin/**` 端点；本条用 `/api/report_comment` 代表

#### DIFF-CRN-03: `MerchantPermissionAOP` 缺失后，非商家用户可以发布商家专属主题帖子

- **等级**: P2
- **分类**: 中间件行为
- **Java 证据**: `aop/src/main/java/com/jb/aop/user/MerchantPermissionAOP.java:35-59`，`theme/src/main/java/com/jb/theme/controller/TopicController.java:96-127`
```java
if(!themeIds.contains(topic.getThemeId())) {
    return;
}
...
if(!Merchant.isMerchant(power)) {
    throw new RuntimeException("当前帖子类型只有商家可以发布");
}
```
- **Go 证据**: `internal/topic/handler.go:21-30`，`internal/topic/service_helpers.go:56-89`，`internal/topic/service.go:46-78`
```go
data, err := h.svc.Create(c.Request.Context(), middleware.GetClaims(c), &req)
```
```go
author, err := s.resolveTopicAuthor(ctx, claims, accountType)
...
res, err := s.topicColl().InsertOne(ctx, topic)
```
- **模拟场景**:
  - 输入: 非商家用户 `power=0`，`POST /api/topic`，请求体 `{"themeId":"theme_merchant_1","title":"促销","content":"..."}`，且 `theme_merchant_1` 已在商家主题集合中
  - Java 行为: 请求被拦截，`campus_topic` 不会新增记录；错误消息为 `当前帖子类型只有商家可以发布`，HTTP 包装 **未验证，需人工确认**
  - Go 行为: 主题存在即可继续创建，最终向 `campus_topic` 插入新帖子并返回成功
- **预期行为**: 商家专属主题应继续只允许商家身份发布。
- **影响面**: `/api/topic` `POST`，商家主题内容治理

### 模块总结

- 活跃端点: 3 个
- Go 已覆盖: 3 个
- P0 差异: 0 个
- P1 差异: 1 个
- P2 差异: 2 个
