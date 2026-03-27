# Batch 2 审计报告：User 模块

审计日期：2026-03-25

审计前提：
- 本次重构切换后，不存在 Java 与 Go 双版本同时部署。
- 旧 Redis 缓存允许在切换前清空。
- 因此前报告中仅在“跨版本互认 token / 共享旧缓存”条件下成立的问题已剔除，不再单列条件性问题。

## 模块：User

### 活跃 API 端点清单

本表列出 User 模块中，Java Controller 已注册且可被客户端直接调用的活跃端点，并与 Go 路由进行交叉比对。

| # | 端点 | 方法 | Java 入口 | Go 入口 | 状态 |
|---|------|------|----------|---------|------|
| 1 | `/api/user/login` | POST | `user/src/main/java/com/jb/user/controller/UserController.java:45-48` | `internal/user/handler.go:21-39` | ✅ |
| 2 | `/api/user` | GET | `user/src/main/java/com/jb/user/controller/UserController.java:51-57` | `internal/user/handler.go:116-124` | ✅ |
| 3 | `/api/user/refresh` | POST | `user/src/main/java/com/jb/user/controller/UserController.java:72-75` | `internal/user/handler.go:41-61` | ✅ |
| 4 | `/api/user` | PUT | `user/src/main/java/com/jb/user/controller/UserController.java:77-82` | `internal/user/handler.go:126-136` | ✅ |
| 5 | `/api/user/authentication` | POST | `user/src/main/java/com/jb/user/controller/UserController.java:87-96` | `internal/user/handler.go:138-148` | ✅ |
| 6 | `/api/user/re_authentication` | POST | `user/src/main/java/com/jb/user/controller/UserController.java:98-108` | `internal/user/handler.go:150-160` | ✅ |
| 7 | `/api/user/del_authentication` | POST | `user/src/main/java/com/jb/user/controller/UserController.java:110-118` | `internal/user/handler.go:162-168` | ✅ |
| 8 | `/api/user/check_login` | POST | `user/src/main/java/com/jb/user/controller/UserController.java:120-129` | `internal/user/handler.go:170-177` | ✅ |
| 9 | `/api/user/get_course_by_weeks` | POST | `user/src/main/java/com/jb/user/controller/UserController.java:132-152` | `internal/user/handler.go:179-190` | ✅ |
| 10 | `/api/user/get_exam` | POST | `user/src/main/java/com/jb/user/controller/UserController.java:154-174` | `internal/user/handler.go:192-199` | ✅ |
| 11 | `/api/user/get_exam_score` | POST | `user/src/main/java/com/jb/user/controller/UserController.java:176-196` | `internal/user/handler.go:201-208` | ✅ |
| 12 | `/api/user/user_profile` | GET | `user/src/main/java/com/jb/user/controller/UserController.java:205-208` | `internal/user/handler.go:210-222` | ✅ |
| 13 | `/api/user/pre_authentication` | PUT | `user/src/main/java/com/jb/user/controller/UserController.java:211-217` | `internal/user/handler.go:63-73` | ✅ |
| 14 | `/api/user/official/certification` | POST | `user/src/main/java/com/jb/user/controller/UserController.java:219-223` | `internal/user/handler.go:98-109` | ✅ |
| 15 | `/api/user/official/login` | POST | `user/src/main/java/com/jb/user/controller/UserController.java:225-229` | `internal/user/handler.go:75-96` | ✅ |
| 16 | `/api/user/identity/anonymous` | POST | `user/src/main/java/com/jb/user/controller/UserController.java:231-235` | `internal/user/handler_identity.go:7-18` | ✅ |
| 17 | `/api/user/identity/anonymous/nickname` | PUT | `user/src/main/java/com/jb/user/controller/UserController.java:237-240` | `internal/user/handler_identity.go:20-30` | ✅ |
| 18 | `/api/user/identity/list` | GET | `user/src/main/java/com/jb/user/controller/UserController.java:243-247` | `internal/user/handler_identity.go:32-39` | ✅ |
| 19 | `/api/user/identity/switch` | POST | `user/src/main/java/com/jb/user/controller/UserController.java:249-253` | `internal/user/handler_identity.go:41-71` | ✅ |
| 20 | `/api/user/nickname/random` | GET | `user/src/main/java/com/jb/user/controller/UserController.java:255-260` | `internal/user/handler.go:111-114` | ✅ |
| 21 | `/api/user/follow` | POST | `user/src/main/java/com/jb/user/controller/UserController.java:268-277` | `internal/user/handler_follow.go:11-21` | ✅ |
| 22 | `/api/user/follow` | DELETE | `user/src/main/java/com/jb/user/controller/UserController.java:285-291` | `internal/user/handler_follow.go:23-34` | ✅ |
| 23 | `/api/user/followers` | GET | `user/src/main/java/com/jb/user/controller/UserController.java:297-312` | `internal/user/handler_follow.go:36-44` | ✅ |
| 24 | `/api/user/followings` | GET | `user/src/main/java/com/jb/user/controller/UserController.java:318-333` | `internal/user/handler_follow.go:46-54` | ✅ |
| 25 | `/api/user/stats` | GET | `user/src/main/java/com/jb/user/controller/UserController.java:340-344` | `internal/user/handler_follow.go:56-68` | ✅ |
| 26 | `/api/user/is_following` | GET | `user/src/main/java/com/jb/user/controller/UserController.java:352-357` | `internal/user/handler_follow.go:70-82` | ✅ |
| 27 | `/admin/user` | POST | `user/src/main/java/com/jb/user/controller/admin/AdminUserController.java:32-35` | `internal/user/admin.go:33-43` | ✅ |
| 28 | `/admin/user/login` | POST | `user/src/main/java/com/jb/user/controller/admin/AdminUserController.java:37-40` | `internal/user/admin.go:20-31` | ✅ |
| 29 | `/admin/user/add` | POST | `user/src/main/java/com/jb/user/controller/admin/AdminUserController.java:42-45` | `internal/user/admin.go:45-55` | ✅ |
| 30 | `/admin/user/{id}` | DELETE | `user/src/main/java/com/jb/user/controller/admin/AdminUserController.java:47-53` | `internal/user/admin.go:57-72` | ✅ |
| 31 | `/admin/user/{id}` | PUT | `user/src/main/java/com/jb/user/controller/admin/AdminUserController.java:55-64` | `internal/user/admin.go:74-93` | ✅ |
| 32 | `/admin/user/{id}` | GET | `user/src/main/java/com/jb/user/controller/admin/AdminUserController.java:66-72` | `internal/user/admin.go:95-111` | ✅ |
| 33 | `/admin/user/list` | GET | `user/src/main/java/com/jb/user/controller/admin/AdminUserController.java:75-79` | `internal/user/admin.go:113-123` | ✅ |
| 34 | `/admin/user/clear` | POST | `user/src/main/java/com/jb/user/controller/admin/AdminUserController.java:81-85` | `internal/user/admin.go:125-135` | ✅ |
| 35 | `/admin/user/course` | POST | `user/src/main/java/com/jb/user/controller/admin/AdminUserController.java:87-92` | `internal/user/admin.go:137-147` | ✅ |
| 36 | `/admin/user/add_black_list` | POST | `user/src/main/java/com/jb/user/controller/admin/AdminUserController.java:94-98` | `internal/user/admin.go:149-160` | ✅ |
| 37 | `/admin/user/del_black_list` | DELETE | `user/src/main/java/com/jb/user/controller/admin/AdminUserController.java:100-104` | `internal/user/admin.go:162-173` | ✅ |
| 38 | `/admin/user/black_list` | GET | `user/src/main/java/com/jb/user/controller/admin/AdminUserController.java:106-110` | `internal/user/admin.go:175-182` | ✅ |
| 39 | `/admin/user/certification/list` | GET | `user/src/main/java/com/jb/user/controller/admin/AdminUserController.java:112-119` | `internal/user/admin.go:184-193` | ✅ |
| 40 | `/admin/user/certification/review` | POST | `user/src/main/java/com/jb/user/controller/admin/AdminUserController.java:121-125` | `internal/user/admin.go:195-205` | ✅ |

状态说明：
- ✅ 端点存在且路径、方法一致
- ❌ Go 中不存在该端点
- ⚠️ 端点存在但路径或方法不一致

### 差异清单

#### DIFF-USR-01: 返回 `User` 的接口在 Go 暴露了 Java 隐藏字段，并省略了 `stuPwd`

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `user-entity/src/main/java/com/jb/userentity/entity/User.java:43-110`
```java
@JSONField(serialize = false)
private String openId;
@JSONField(serialize = false)
private Long createdBy;
@JSONField(serialize = false)
private Date createdAt;
private String stuPwd;
```
- **Go 证据**: `internal/user/model.go:10-33`
```go
OpenID    string    `json:"openId"`
StuPwd    string    `json:"-"`
CreatedAt time.Time `json:"createdAt"`
CreatedBy int64     `json:"createdBy"`
```
- **模拟场景**:
  - 输入: `GET /api/user`，JWT 对应用户 `id=101`
  - Java 行为: `{"success":true,"code":0,"msg":"","data":{"id":101,"nickname":"Alice","avatar":"a.png","gender":"保密","power":0,"accountType":"base","rootUserId":101,"lastSwitchId":101,"signature":"","stuNum":"2023001","stuName":"Alice","stuCla":"计科1班","stuPwd":"","school":"SZTU","stuIsCheck":true,"tag":"student"}}`
  - Go 行为: `{"success":true,"code":0,"msg":"","data":{"id":101,"openId":"wx_abc","nickname":"Alice","avatar":"a.png","power":0,"accountType":"base","stuNum":"2023001","stuName":"Alice","stuCla":"计科1班","stuIsCheck":true,"school":"SZTU","tag":"student","gender":"保密","rootUserId":101,"lastSwitchId":101,"signature":"","createdAt":"2026-03-20T10:00:00Z","createdBy":0,"updatedAt":"2026-03-25T10:00:00Z","updatedBy":101}}`
- **预期行为**: Go 对外返回 `User` 时应保持 Java 的字段可见性和字段集合。
- **影响面**: `/api/user/login`、`/api/user/official/login`、`/api/user`、`/admin/user/login`、`/admin/user/{id}`、`/admin/user/list`、`/admin/user/black_list`

#### DIFF-USR-02: `/api/user/login` 未恢复根账号的当前身份，始终回到基座账号

- **等级**: P2
- **分类**: 业务逻辑
- **Java 证据**: `user/src/main/java/com/jb/user/service/impl/UserServiceImpl.java:264-277,785-833`
```java
User activeIdentity = resolveActiveIdentity(rootUser);
JwtClaimsConf j = buildJwtClaims(activeIdentity, rootUser);
wxLoginVO.setCurrentIdentity(buildIdentityVO(activeIdentity));
```
- **Go 证据**: `internal/user/service.go:155-210`，`internal/user/handler.go:31-38`，`internal/user/service_identity.go:114-116`
```go
u, err := s.GetByOpenID(ctx, resp.OpenID)
token, refreshToken, err := s.jwtHelper.GenerateTokenPair(&jwtutil.TokenUser{
    ID: u.ID, AccountType: u.AccountType, RootUserID: u.RootUserID,
})
db.Model(&User{}).Where("id = ?", current.ID).Update("lastSwitchId", target.ID)
```
- **模拟场景**:
  - 输入: 根账号 `id=101, openId=wx_root, lastSwitchId=205`，匿名身份 `id=205, rootUserId=101, accountType=anonymous`；`POST /api/user/login {"code":"wx-code-101"}`
  - Java 行为: `{"success":true,"code":200,"msg":"成功","data":{"token":"token-for-205","refresh_token":"rtk-for-205","user":{"id":101,"lastSwitchId":205,...},"is_new":false,"currentIdentity":{"userId":205,"accountType":"anonymous","nickname":"蓬莱问道のe","avatar":"39862f6788f0d8852a2c095e0d4f7057","tag":"student"},"rootUserId":101}}`
  - Go 行为: `{"success":true,"code":200,"msg":"成功","data":{"token":"token-for-101","refresh_token":"rtk-for-101","user":{"id":101,"lastSwitchId":205,...},"is_new":false,"currentIdentity":{"userId":101,"accountType":"base","nickname":"Alice","avatar":"a.png","tag":"student"},"rootUserId":101}}`
- **预期行为**: 登录后返回的 token 和 `currentIdentity` 应反映根账号上次切换的身份，而不是固定回到基座账号。
- **影响面**: `/api/user/login`、`/api/user/identity/switch`

#### DIFF-USR-03: `/api/user/pre_authentication` 从 query 参数更新变成了 JSON 绑定空操作

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `user/src/main/java/com/jb/user/controller/UserController.java:211-216`，`user/src/main/java/com/jb/user/service/impl/UserServiceImpl.java:487-499`
```java
@PutMapping("/pre_authentication")
public Result<?> preAuthentication(
    @RequestParam("user_id") String userId,
    @RequestParam("nick_name") String nickName,
    @RequestParam("pwd") String pwd)
```
```java
.eq("id", Long.parseLong(userId))
.eq("nickname", nickName)
.set("stu_is_check", 1);
```
- **Go 证据**: `internal/user/handler.go:63-72`，`internal/user/service_extra.go:21-25`
```go
var req AuthenticationReq
if !result.BindJSON(c, &req) { return }
if err := h.svc.PreAuthentication(ctx, req.StuNum, req.StuPwd); err != nil { ... }
```
```go
func (s *Service) PreAuthentication(ctx context.Context, stuNum, stuPwd string) error {
    if stuNum == "" || stuPwd == "" { return result.ErrParam }
    return nil
}
```
- **模拟场景**:
  - 输入: `PUT /api/user/pre_authentication?user_id=101&nick_name=Alice&pwd=zjb%26bjz`
  - Java 行为: `{"success":true,"code":200,"msg":"预认证成功","data":null}`，并把 `campus_user.stu_is_check` 更新为 `1`
  - Go 行为: HTTP 400，`{"success":false,"code":7,"msg":"请求体不能为空","data":null}`
- **预期行为**: Go 应继续接受 Java 现有的 query 参数契约，并产生相同的认证状态更新。
- **影响面**: `/api/user/pre_authentication`

#### DIFF-USR-04: `/api/user/official/certification` 的请求体、响应体和 Mongo 文档结构都被缩减了

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `user/src/main/java/com/jb/user/dto/OfficialCertificationDTO.java:9-41`，`user/src/main/java/com/jb/user/service/impl/UserServiceImpl.java:868-909`
```java
private String avatarUrl;
private String fullName;
private String shortName;
private String loginAccount;
private String loginPassword;
```
```java
OfficialCertification certification = OfficialCertification.builder()
    .avatarUrl(dto.getAvatarUrl())
    .fullName(dto.getFullName())
    .loginAccount(dto.getLoginAccount())
    .loginPassword(encryptedPassword)
    .status(OfficialCertification.STATUS_PENDING)
```
- **Go 证据**: `internal/user/model_req.go:25-28`，`internal/user/service_extra.go:133-143`，`internal/user/model.go:66-73`
```go
type OfficialCertReq struct {
    Name string `json:"name" binding:"required"`
    Reason string `json:"reason" binding:"required"`
}
```
```go
doc := OfficialCertification{
    UserID: strconv.FormatInt(userID, 10),
    Name: name, Reason: reason, Status: 0,
}
```
- **模拟场景**:
  - 输入:
```json
{"avatarUrl":"logo.png","fullName":"计算机协会","shortName":"计协","nature":"student","introduction":"intro","responsiblePerson":"张三","wechatAccount":"wx123","loginAccount":"clubA","loginPassword":"pass123"}
```
  - Java 行为: `{"success":true,"code":200,"msg":"认证申请提交成功，请等待审核","data":{"id":"66f0...","avatarUrl":"logo.png","fullName":"计算机协会","shortName":"计协","nature":"student","introduction":"intro","responsiblePerson":"张三","wechatAccount":"wx123","loginAccount":"clubA","loginPassword":"<AES密文>","status":"PENDING","rejectReason":"","reviewedBy":0,"reviewedAt":null,"createdAt":"2026-03-25T10:00:00Z","updatedAt":"2026-03-25T10:00:00Z"}}`
  - Go 行为: HTTP 400，`{"success":false,"code":1,"msg":"参数错误","data":null}`
- **预期行为**: Go 应继续接受 Java 已公开的官方认证申请字段，并生成相同的对外响应和后续可审核文档。
- **影响面**: `/api/user/official/certification`、`/admin/user/certification/list`、`/admin/user/certification/review`

#### DIFF-USR-05: `/api/user/official/login` 无法读取 Java 审核流创建的官方账号

- **等级**: P1
- **分类**: 数据兼容
- **Java 证据**: `user/src/main/java/com/jb/user/service/impl/UserServiceImpl.java:912-941`，`user/src/main/java/com/jb/user/service/impl/AdminServiceImpl.java:529-557`
```java
.eq(User::getStuNum, dto.getLoginAccount())
.eq(User::getStuPwd, AESUtil.encrypt(dto.getLoginPassword(), encryptionKey))
if (officialUser.getOpenId() == null || !officialUser.getOpenId().startsWith("official:"))
```
```java
officialUser.setOpenId("official:" + certification.getLoginAccount());
officialUser.setStuNum(certification.getLoginAccount());
officialUser.setStuPwd(certification.getLoginPassword());
```
- **Go 证据**: `internal/user/service_extra.go:96-130`
```go
err := s.db.WithContext(ctx).Where("stuNum = ? AND accountType = ?", username, "official").First(&u).Error
if errors.Is(err, gorm.ErrRecordNotFound) {
    return "", "", nil, ErrUserNotFound
}
```
- **模拟场景**:
  - 输入: DB 中已有 Java 审核通过后创建的用户行：`openId="official:clubA", stuNum="clubA", stuPwd=AES(pass123), accountType="base"`；请求 `POST /api/user/official/login {"loginAccount":"clubA","loginPassword":"pass123"}`
  - Java 行为: `{"success":true,"code":200,"msg":"登录成功","data":{"token":"atk","refresh_token":"rtk","user":{"id":301,"nickname":"计协","stuNum":"clubA","stuPwd":"","...":""},"is_new":false}}`
  - Go 行为: `{"success":false,"code":-1,"msg":"系统错误","data":null}`
- **预期行为**: Go 应能读取 Java 审核流已创建的官方账号数据，并给出相同的登录结果。
- **影响面**: `/api/user/official/login`

#### DIFF-USR-06: `/api/user/authentication` 和 `/api/user/re_authentication` 的请求体、成功返回和入库字段都与 Java 不一致

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `user/src/main/java/com/jb/user/dto/AuthDto.java:20-31`，`user/src/main/java/com/jb/user/service/impl/UserServiceImpl.java:364-403`，`user/src/main/java/com/jb/user/controller/UserController.java:87-107`
```java
private String schoolId;
private String password;
private String school;
```
```java
.set(User::getStuCla, loginResp.getMajor())
.set(User::getStuName, loginResp.getName())
.set(User::getStuPwd, AESUtil.encrypt(password, encryptionKey))
.set(User::getSchool, authDto.getSchool()));
return R.success(loginResp).msg("认证成功");
```
- **Go 证据**: `internal/user/model_req.go:12-15`，`internal/user/handler.go:138-159`，`internal/user/service_extra.go:28-49`
```go
type AuthenticationReq struct {
    StuNum string `json:"stuNum" binding:"required"`
    StuPwd string `json:"stuPwd" binding:"required"`
}
```
```go
result.Success(c, nil)
```
```go
Updates(map[string]interface{}{
    "stuNum": stuNum,
    "stuPwd": encPwd,
    "stuIsCheck": true,
})
```
- **模拟场景**:
  - 输入:
```json
{"schoolId":"2023001","password":"jw-pass","school":"SZTU"}
```
  - Java 行为: `{"success":true,"code":200,"msg":"认证成功","data":{"isLogin":true,"major":"计算机科学与技术","name":"张三"}}`，并写入 `stu_num=2023001, stu_pwd=<AES>, stu_name=张三, stu_cla=计算机科学与技术, school=SZTU, stu_is_check=true`
  - Go 行为: HTTP 400，`{"success":false,"code":1,"msg":"参数错误","data":null}`
- **预期行为**: Go 应保持 Java 认证接口的请求字段、成功响应和认证后用户数据写入结果。
- **影响面**: `/api/user/authentication`、`/api/user/re_authentication`

#### DIFF-USR-07: `/api/user/check_login` 从“校验给定教务凭据”变成了“读取本地 `stuIsCheck` 布尔值”

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `user/src/main/java/com/jb/user/dto/CheckLoginDto.java:14-21`，`user/src/main/java/com/jb/user/service/impl/UserServiceImpl.java:433-449`
```java
private String schoolId;
private String password;
JWCommonResp<?> login = jwService.login(schoolId, password);
return R.success(loginResp).msg("认证成功");
```
- **Go 证据**: `internal/user/handler.go:170-177`，`internal/user/service_extra.go:65-73`
```go
ok, err := h.svc.CheckLogin(ctx, currentUserID(c))
result.Success(c, ok)
```
```go
return u.StuIsCheck, nil
```
- **模拟场景**:
  - 输入: 已认证用户 `id=101, stuIsCheck=true`；`POST /api/user/check_login {"schoolId":"2023001","password":"bad-pass"}`
  - Java 行为: `{"success":false,"code":400,"msg":"<教务返回的失败消息>","data":null}`
  - Go 行为: `{"success":true,"code":200,"msg":"成功","data":true}`
- **预期行为**: Go 应继续用客户端提交的教务凭据进行实时校验，并返回与 Java 相同的结果类型。
- **影响面**: `/api/user/check_login`

#### DIFF-USR-08: `/api/user/get_course_by_weeks`、`/get_exam`、`/get_exam_score` 在 Go 中退化为本地空实现

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `user/src/main/java/com/jb/user/controller/UserController.java:132-195`
```java
JWCommonResp<?> course = jwService.getCourseByWeeks(...)
return R.data(course.getData()).msg(course.getMessage()).code(course.getCode());
```
- **Go 证据**: `internal/user/handler.go:179-208`，`internal/user/service_extra.go:76-93`
```go
func (s *Service) GetCourseByWeeks(...) ([]map[string]interface{}, error) {
    return []map[string]interface{}{}, nil
}
func (s *Service) GetExam(...) ([]map[string]interface{}, error) { return []map[string]interface{}{}, nil }
func (s *Service) GetExamScore(...) ([]map[string]interface{}, error) { return []map[string]interface{}{}, nil }
```
- **模拟场景**:
  - 输入: `POST /api/user/get_exam {"schoolId":"2023001","password":"jw-pass","xnxqid":"2025-2026-1"}`
  - Java 行为: 在教务返回成功时，透传例如 `{"success":true,"code":200,"msg":"success","data":[{"course":"高数","time":"2026-01-08 09:00"}]}`
  - Go 行为: `{"success":true,"code":200,"msg":"成功","data":[]}`
- **预期行为**: Go 应继续接受 Java 现有请求字段，并返回教务服务的实际结果，而不是固定空数组。
- **影响面**: `/api/user/get_course_by_weeks`、`/api/user/get_exam`、`/api/user/get_exam_score`

#### DIFF-USR-09: `/api/user` 编辑接口的 HTTP 返回和 MQ 更新内容都缩水了

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `user/src/main/java/com/jb/user/service/impl/UserServiceImpl.java:321-358`，`theme-entity/src/main/java/com/jb/themeentity/entity/inner/CmtUser.java:11-19`
```java
cmtUser.setNickName(userDTO.getNickname());
cmtUser.setGender(userDTO.getGender());
cmtUser.setSignature(userDTO.getSignature());
return R.success(user);
```
- **Go 证据**: `internal/user/handler.go:126-135`，`internal/user/service.go:133-151`，`internal/mq/model.go:48-59`
```go
result.Success(c, nil)
```
```go
msg := mq.TopicUserUpdateMsg{
    UserID: strconv.FormatInt(userID, 10),
    NickName: req.Nickname,
    Avatar: req.Avatar,
    AccountType: accountType,
}
```
- **模拟场景**:
  - 输入: `PUT /api/user {"nickname":"Alice2","gender":"女","signature":"新签名"}`
  - Java 行为: `{"success":true,"code":200,"msg":"成功","data":{"id":101,"nickname":"Alice2","gender":"女","signature":"新签名",...}}`；MQ `data` 至少包含 `{"userId":"101","nickName":"Alice2","gender":"女","signature":"新签名"}`
  - Go 行为: `{"success":true,"code":200,"msg":"成功","data":null}`；MQ `data` 只有 `{"userId":"101","nickName":"Alice2","avatar":"","accountType":0}`
- **预期行为**: Go 编辑用户后应继续返回更新后的用户对象，并向下游发送与 Java 等价的用户更新消息内容。
- **影响面**: `/api/user`；依赖用户更新 MQ 的 Topic/Comment 下游链路

#### DIFF-USR-10: `/api/user/identity/anonymous` 从“无请求体创建匿名身份”变成了“必须提交昵称且返回完整 User”

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `user/src/main/java/com/jb/user/controller/UserController.java:231-235`，`user/src/main/java/com/jb/user/service/impl/UserServiceImpl.java:508-529,663-687`
```java
@PostMapping("/identity/anonymous")
public Result<?> createAnonymousIdentity() { return userService.createAnonymousIdentity(); }
```
```java
anonymous.setNickname(RandomName.generateAnonymousID());
anonymous.setOpenId(buildAnonymousOpenId(rootUser));
return R.success(buildIdentityVO(anonymous));
```
- **Go 证据**: `internal/user/handler_identity.go:7-18`，`internal/user/model_req.go:30-32`，`internal/user/service_identity.go:12-45`
```go
type IdentityAnonymousReq struct {
    Nickname string `json:"nickname" binding:"required"`
}
```
```go
u := &User{
    OpenID: baseUser.OpenID,
    Nickname: nickname,
    AccountType: "anonymous",
}
result.Success(c, u)
```
- **模拟场景**:
  - 输入: `POST /api/user/identity/anonymous`，空请求体
  - Java 行为: `{"success":true,"code":200,"msg":"成功","data":{"userId":205,"accountType":"anonymous","nickname":"蓬莱问道のe","avatar":"39862f6788f0d8852a2c095e0d4f7057","tag":"student"}}`
  - Go 行为: HTTP 400，`{"success":false,"code":7,"msg":"请求体不能为空","data":null}`
- **预期行为**: Go 应继续支持无请求体创建匿名身份，并返回 Java 的 `IdentityVO` 结构。
- **影响面**: `/api/user/identity/anonymous`

#### DIFF-USR-11: `/api/user/identity/anonymous/nickname` 在基座账号上下文下不再生效，也丢失了 72 小时限制

- **等级**: P2
- **分类**: 业务逻辑
- **Java 证据**: `user/src/main/java/com/jb/user/service/impl/UserServiceImpl.java:533-564`
```java
User anonymous = getIdentityByType(rootUserId, AccountType.ANONYMOUS.getCode());
if (hoursSinceUpdate < ANONYMOUNS_NICKNAME_UPDATE_HOUR_LIMIT) {
    return R.fail().msg(String.format("昵称修改还需等待 %d 小时", remainingHours));
}
```
- **Go 证据**: `internal/user/handler_identity.go:20-29`，`internal/user/service_identity.go:48-58`
```go
Where("id = ? AND accountType = ?", userID, "anonymous").
Update("nickname", nickname)
```
- **模拟场景**:
  - 输入: 当前 JWT 是基座账号 `userId=101, rootUserId=101`，匿名身份 `id=205` 的 `updatedAt` 距今 1 小时；`PUT /api/user/identity/anonymous/nickname {"nickname":"新匿名名"}`
  - Java 行为: `{"success":false,"code":400,"msg":"昵称修改还需等待 71 小时","data":null}`
  - Go 行为: `{"success":true,"code":200,"msg":"成功","data":null}`
- **预期行为**: Go 应在根账号上下文下找到匿名身份，并保持 Java 的修改频率限制。
- **影响面**: `/api/user/identity/anonymous/nickname`

#### DIFF-USR-12: `/api/user/identity/list` 返回结构从 `IdentityListVO` 变成了原始 `User[]`

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `user/src/main/java/com/jb/user/vo/IdentityListVO.java:14-28`，`user/src/main/java/com/jb/user/service/impl/UserServiceImpl.java:574-599`
```java
IdentityListVO.builder()
    .rootUserId(rootUserId)
    .identities(identityVOList)
    .hasAnonymous(hasAnonymous)
```
- **Go 证据**: `internal/user/handler_identity.go:32-39`，`internal/user/service_identity.go:61-82`
```go
data, err := h.svc.ListIdentities(...)
result.Success(c, data)
```
- **模拟场景**:
  - 输入: `GET /api/user/identity/list`
  - Java 行为: `{"success":true,"code":200,"msg":"成功","data":{"rootUserId":101,"identities":[{"userId":101,"accountType":"base","nickname":"Alice","avatar":"a.png","tag":"student"},{"userId":205,"accountType":"anonymous","nickname":"蓬莱问道のe","avatar":"anon.png","tag":"student"}],"hasAnonymous":true}}`
  - Go 行为: `{"success":true,"code":200,"msg":"成功","data":[{"id":101,"openId":"wx_root","nickname":"Alice",...},{"id":205,"openId":"wx_root","nickname":"蓬莱问道のe",...}]}`
- **预期行为**: Go 应保持 Java 的 `rootUserId + identities + hasAnonymous` 返回结构。
- **影响面**: `/api/user/identity/list`

#### DIFF-USR-13: `/api/user/follow` 和 `/api/user/unfollow` 改了入参绑定，并放宽了重复关注/未关注取关的约束

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `user/src/main/java/com/jb/user/controller/UserController.java:269-290`，`user/src/main/java/com/jb/user/service/impl/UserServiceImpl.java:956-1008`
```java
@PostMapping("/follow")
public Result<?> follow(@RequestParam("following_id") Long followingId)
```
```java
if (mongoTemplate.exists(query, Follow.class)) {
    return R.fail(RC.FL_REPEAT);
}
if (!mongoTemplate.exists(query, Follow.class)) {
    return R.fail(RC.FL_UNFOLLOW_NOT_FOLLOW);
}
```
- **Go 证据**: `internal/user/handler_follow.go:11-33`，`internal/user/service_follow.go:17-48`，`internal/user/model_req.go:39-41`
```go
type FollowReq struct {
    TargetUserID int64 `json:"targetUserId" binding:"required"`
}
```
```go
coll.UpdateOne(..., bson.M{"$setOnInsert": doc}, options.Update().SetUpsert(true))
coll.DeleteOne(...)
```
- **模拟场景**:
  - 输入: `POST /api/user/follow?following_id=202`
  - Java 行为: 若 `202` 已关注，返回 `{"success":false,"code":1002,"msg":"不可重复关注","data":null}`；否则正常创建
  - Go 行为: HTTP 400，`{"success":false,"code":7,"msg":"请求体不能为空","data":null}`
- **预期行为**: Go 应继续接受 Java 的 query 参数契约，并保留重复关注、未关注取关、自关注的判定结果。
- **影响面**: `/api/user/follow`、`/api/user/unfollow`

#### DIFF-USR-14: `/api/user/followers` 和 `/api/user/followings` 忽略了 `targetId`，且返回项从 `FollowVO` 变成了 `User`

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `user/src/main/java/com/jb/user/controller/UserController.java:298-333`，`user/src/main/java/com/jb/user/vo/FollowVO.java:15-25`，`user/src/main/java/com/jb/user/service/impl/UserServiceImpl.java:1018-1160`
```java
@RequestParam(value = "targetId") Long userId
```
```java
FollowVO.builder()
    .follower_id(id)
    .following_id(userId)
    .follow_at(follow.getFollowAt())
    .both_follow(each_follow)
```
- **Go 证据**: `internal/user/handler_follow.go:36-54`，`internal/user/service_follow.go:113-166`
```go
data, err := h.svc.ListFollowers(ctx, currentUserID(c), page, size)
return result.NewCusPage(users, total, page, size)
```
- **模拟场景**:
  - 输入: 当前用户 `101` 请求 `GET /api/user/followers?targetId=202&page=1&size=2`
  - Java 行为: 返回 `202` 的粉丝页，例如 `{"success":true,"code":200,"msg":"成功","data":{"data":[{"avatar":"x.png","nickname":"Bob","follower_id":301,"following_id":202,"follow_at":"2026-03-20T10:00:00Z","co_follow":false,"both_follow":true}],"current":1,"total":1,"size":2}}`
  - Go 行为: 返回 `101` 自己的粉丝页，且项为原始用户对象，例如 `{"success":true,"code":200,"msg":"成功","data":{"data":[{"id":301,"openId":"wx_301","nickname":"Bob",...}],"current":1,"total":1,"size":2}}`
- **预期行为**: Go 应按 `targetId` 查询目标用户的关注关系，并返回 Java 的 `FollowVO` 项结构。
- **影响面**: `/api/user/followers`、`/api/user/followings`

#### DIFF-USR-15: `/api/user/user_profile`、`/api/user/stats`、`/api/user/is_following` 的 query 参数名和返回语义都变了

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `user/src/main/java/com/jb/user/controller/UserController.java:205-208,340-357`，`user/src/main/java/com/jb/user/service/impl/UserServiceImpl.java:470-483,1170-1215`，`user/src/main/java/com/jb/user/vo/UserVO.java:12-18`，`user/src/main/java/com/jb/user/vo/UserStatsVO.java:12-20`
```java
@RequestParam("target_user_id") String targetUserId
@RequestParam("user_id") Long userId
@RequestParam("target_user_id") Long targetUserId
```
- **Go 证据**: `internal/user/handler.go:210-221`，`internal/user/handler_follow.go:56-81`，`internal/user/model.go:81-87`，`internal/user/service_follow.go:63-110`
```go
targetUserID, err := strconv.ParseInt(c.Query("targetUserId"), 10, 64)
```
```go
type UserProfile struct {
    User
    FollowerCount  int64
    FollowingCount int64
    LikeCount      int64
    TopicCount     int64
    IsFollowing    bool
}
```
- **模拟场景**:
  - 输入: `GET /api/user/stats?user_id=202`
  - Java 行为: `{"success":true,"code":200,"msg":"成功","data":{"followerCount":3,"followingCount":5,"likeCount":40}}`
  - Go 行为: `{"success":false,"code":1,"msg":"参数错误","data":null}`
- **预期行为**: Go 应继续接受 Java 的 snake_case query 参数，并保持 Java 既有的 `UserVO` / `UserStatsVO` / `boolean` 返回语义。
- **影响面**: `/api/user/user_profile`、`/api/user/stats`、`/api/user/is_following`

#### DIFF-USR-16: `/admin/user/login` 当前不会接受 Java 基线的二级密码

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `user/src/main/java/com/jb/user/service/impl/AdminServiceImpl.java:100-104`
```java
if (!Objects.equals(adminDTO.getSecondaryPassword(), "pyhtip-nyxqen-6rigvE")) {
    int failCount = handleLoginFail(...);
    return R.fail().msg("二级密码错误，今日还有 " + (10 - failCount) + " 次机会");
}
```
- **Go 证据**: `internal/user/service_admin.go:38-44`，`configs/ecampus-crm/application.yml:26-28`
```go
if s.cfg != nil && s.cfg.Admin.SecondaryPassword != "" &&
   req.SecondaryPassword != s.cfg.Admin.SecondaryPassword { ... }
```
```yaml
admin:
  power_sign: 999
  secondary_password: "replace-me"
```
- **模拟场景**:
  - 输入: `POST /admin/user/login {"username":"root","password":"pass123","secondaryPassword":"pyhtip-nyxqen-6rigvE"}`
  - Java 行为: 在账号密码正确时返回 `{"success":true,"code":200,"msg":"成功","data":{"token":"atk","refresh_token":"rtk","user":{...}}}`
  - Go 行为: `{"success":false,"code":-1,"msg":"系统错误","data":null}`，因为当前配置下二级密码恒不匹配，且该错误未映射为固定业务码
- **预期行为**: Go 应对同一组管理员登录凭据给出与 Java 相同的通过/拒绝结果。
- **影响面**: `/admin/user/login`

#### DIFF-USR-17: `/admin/user/add` 和 `/admin/user/{id}` PUT 可写字段集比 Java 少，管理员无法设置 `power`/学籍字段

- **等级**: P2
- **分类**: 业务逻辑
- **Java 证据**: `user/src/main/java/com/jb/user/dto/admin/AdminDTO.java:12-22`，`user/src/main/java/com/jb/user/service/impl/AdminServiceImpl.java:257-279`，`user/src/main/java/com/jb/user/service/impl/UserServiceImpl.java:164-189`
```java
private Integer power;
Admin newAdmin = new Admin(..., resolveAdminPower(adminDTO.getPower()));
```
```java
.set(editUserDTO.getPower() != null, User::getPower, editUserDTO.getPower())
.set(StringUtils.isNotBlank(editUserDTO.getStuNum()), User::getStuNum, editUserDTO.getStuNum())
.set(editUserDTO.getStuIsCheck() != null, User::getStuIsCheck, editUserDTO.getStuIsCheck())
```
- **Go 证据**: `internal/user/model_req.go:48-52`，`internal/user/service_admin_ops.go:34-44`，`internal/user/admin.go:74-92`，`internal/user/service.go:109-124`
```go
type AddAdminReq struct {
    UserID   int64  `json:"userId" binding:"required"`
    Username string `json:"username" binding:"required"`
    Password string `json:"password" binding:"required"`
}
```
```go
admin := Admin{UserID: userID, Username: username, Password: md5Hex(password), Power: 2}
```
```go
if req.Tag != "" { updates["tag"] = req.Tag }
if req.Gender != "" { updates["gender"] = req.Gender }
if req.Signature != "" { updates["signature"] = req.Signature }
```
- **模拟场景**:
  - 输入: `PUT /admin/user/101 {"power":8,"stuNum":"2023001","stuName":"张三","stuCla":"计科1班","stuIsCheck":true}`
  - Java 行为: `{"success":true,"code":200,"msg":"更新成功","data":null}`，并更新 `power/stuNum/stuName/stuCla/stuIsCheck`
  - Go 行为: `{"success":true,"code":200,"msg":"成功","data":null}`，但这些字段不会被写入
- **预期行为**: Go 管理员新增/编辑接口应支持 Java 已公开的管理员字段集，并产生相同的数据更新效果。
- **影响面**: `/admin/user/add`、`/admin/user/{id}` PUT

#### DIFF-USR-18: `/admin/user/clear` 会额外清空 `stuPwd`，破坏了 Java 保留的认证密码数据

- **等级**: P1
- **分类**: 数据兼容
- **Java 证据**: `user/src/main/java/com/jb/user/service/impl/AdminServiceImpl.java:311-313`，`user/src/main/java/com/jb/user/service/impl/UserServiceImpl.java:413-424`
```java
.set(User::getStuIsCheck, false)
.set(User::getStuName, "")
.set(User::getStuCla, "")
.set(User::getStuNum, "")
```
- **Go 证据**: `internal/user/service_admin_ops.go:79-80`，`internal/user/service_extra.go:52-59`
```go
return s.DelAuthentication(ctx, userID)
```
```go
Updates(map[string]interface{}{
    "stuNum": "", "stuPwd": "", "stuName": "", "stuCla": "", "stuIsCheck": false,
})
```
- **模拟场景**:
  - 输入: `POST /admin/user/clear {"userId":101}`，用户原有 `stuPwd="<AES密文>"`
  - Java 行为: MySQL 更新后 `stuPwd` 保持 `<AES密文>`
  - Go 行为: MySQL 更新后 `stuPwd=""`
- **预期行为**: Go 管理端清除校园认证时应保持与 Java 相同的字段保留策略。
- **影响面**: `/admin/user/clear`，以及后续依赖已保存教务密码的课程导出/认证链路

#### DIFF-USR-19: `/admin/user/course` 从“同步文件下载接口”变成了“JSON 确认接口”

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `user/src/main/java/com/jb/user/controller/admin/AdminUserController.java:87-92`，`user/src/main/java/com/jb/user/service/impl/UserServiceImpl.java:454-465`
```java
public CompletableFuture<ResponseEntity<?>> getCourse(@RequestParam("key") String key)
```
```java
return ResponseEntity.ok()
    .header(HttpHeaders.CONTENT_DISPOSITION, "attachment;filename=" + key)
    .contentType(mediaType)
    .body(course.getVal().getData());
```
- **Go 证据**: `internal/user/admin.go:137-147`，`internal/user/service_admin_ops.go:245-289`
```go
var req CourseFetchReq
if !result.BindJSON(c, &req) { return }
if err := h.svc.RequestCourseByKey(ctx, req.Key); err != nil { ... }
result.Success(c, nil)
```
- **模拟场景**:
  - 输入: `POST /admin/user/course?key=campus:user_course:101:2025-2026-1:1`
  - Java 行为: HTTP 200，响应体为 Excel 文件字节流，`Content-Disposition: attachment;filename=campus:user_course:101:2025-2026-1:1`
  - Go 行为: HTTP 400，`{"success":false,"code":7,"msg":"请求体不能为空","data":null}`
- **预期行为**: Go 应保留 Java 的请求方式和下载响应语义，而不是改成标准 JSON 确认。
- **影响面**: `/admin/user/course`

#### DIFF-USR-20: `/admin/user/certification/list` 和 `/admin/user/certification/review` 不再使用 Java 的官方认证数据模型和审核流程

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `user/src/main/java/com/jb/user/controller/admin/AdminUserController.java:112-124`，`user/src/main/java/com/jb/user/dto/admin/ReviewCertificationDTO.java:9-21`，`user/src/main/java/com/jb/user/service/impl/AdminServiceImpl.java:470-577`
```java
private String certificationId;
private String action;
private String rejectReason;
private String tag;
```
```java
query.addCriteria(Criteria.where("status").is(status));
query.fields().exclude("loginPassword");
officialUser.setOpenId("official:" + certification.getLoginAccount());
officialUser.setStuNum(certification.getLoginAccount());
```
- **Go 证据**: `internal/user/admin.go:184-204`，`internal/user/model.go:66-73`，`internal/user/model_req.go:62-65`，`internal/user/service_admin_ops.go:185-242`
```go
type OfficialCertification struct {
    UserID string `bson:"userId" json:"userId"`
    Name   string `bson:"name" json:"name"`
    Reason string `bson:"reason" json:"reason"`
    Status int    `bson:"status" json:"status"`
}
```
```go
type CertReviewReq struct {
    CertID   string `json:"certId" binding:"required"`
    Approved bool   `json:"approved"`
}
```
```go
FindOneAndUpdate(..., bson.M{"$set": bson.M{"status": status}})
if approved {
    db.Model(&User{}).Where("id = ?", userID).Updates(map[string]interface{}{"accountType": "official"})
}
```
- **模拟场景**:
  - 输入:
```json
{"certificationId":"66f0abc12345678901234567","action":"APPROVED","tag":"organization"}
```
  - Java 行为: `{"success":true,"code":200,"msg":"审核通过，用户创建成功","data":null}`，并创建 `openId="official:loginAccount"` 的新官方用户，更新申请 `status="APPROVED"`
  - Go 行为: HTTP 400，`{"success":false,"code":1,"msg":"参数错误","data":null}`
- **预期行为**: Go 应继续支持 Java 的审核请求字段、状态过滤和“审核通过后创建官方账号”的完整流程。
- **影响面**: `/admin/user/certification/list`、`/admin/user/certification/review`

### 模块总结

- 活跃端点: 40 个
- Go 已覆盖: 40 个
- P0 差异: 15 个
- P1 差异: 2 个
- P2 差异: 3 个
