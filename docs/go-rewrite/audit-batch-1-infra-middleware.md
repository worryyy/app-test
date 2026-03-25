# Batch 1 审计报告：基础设施 + 中间件

审计日期：2026-03-25

审计前提：
- 本次重构切换后，不存在 Java 与 Go 双版本同时部署。
- 旧 Redis 缓存允许在切换前清空。
- 因此前报告中仅在“跨版本互认 token / 共享旧缓存”条件下成立的问题已剔除，不再单列条件性问题。

## 模块：基础设施 + 中间件

### 活跃 API 端点清单

本表只列出本轮与基础设施/中间件差异直接相关、且已从 Java Controller 追到 Go 路由的活跃端点。

| # | 端点 | 方法 | Java 入口 | Go 入口 | 状态 |
|---|------|------|----------|---------|------|
| 1 | /api/user/login | POST | UserController.java:45 | handler.go:21 | ✅ |
| 2 | /api/user/refresh | POST | UserController.java:72 | handler.go:35 | ✅ |
| 3 | /admin/user/login | POST | AdminUserController.java:37 | admin.go:19 | ✅ |
| 4 | /api/user/identity/switch | POST | UserController.java:250 | handler_identity.go:43 | ✅ |
| 5 | /api/user/nickname/random | GET | UserController.java:256 | handler.go:93 | ✅ |
| 6 | /api/support/{key} | GET | FrontendSupportController.java:34 | support.go:9 | ✅ |
| 7 | /api/support/list | GET | FrontendSupportController.java:47 | support.go:18 | ✅ |
| 8 | /api/theme/campus/init | POST | ThemeController.java:34 | handler.go:15 | ✅ |
| 9 | /api/theme/campus | GET | ThemeController.java:40 | handler.go:28 | ✅ |
| 10 | /api/term/list | GET | TermController.java:40 | handler.go:18 | ✅ |
| 11 | /api/term | GET | TermController.java:48 | handler.go:27 | ✅ |
| 12 | /api/notice/list | GET | NoticeController.java:35 | notice.go:9 | ✅ |
| 13 | /api/wx/unlimited/wxa_code | POST | WXaCodeController.java:26 | handler.go:210 | ✅ |
| 14 | /api/user/authentication | POST | UserController.java:87 | handler.go:121 | ✅ |
| 15 | /api/user/re_authentication | POST | UserController.java:98 | handler.go:134 | ✅ |
| 16 | /admin/user/list | GET | AdminUserController.java:75 | admin.go:104 | ✅ |
| 17 | /admin/local_cache/all_key | GET | LocalCacheController.java:31 | monitor/admin.go:15 | ✅ |
| 18 | /admin/local_cache/stats | GET | LocalCacheController.java:38 | monitor/admin.go:24 | ✅ |
| 19 | /admin/user/add_black_list | POST | AdminUserController.java:94 | admin.go:142 | ✅ |
| 20 | /admin/user/del_black_list | DELETE | AdminUserController.java:100 | admin.go:155 | ✅ |
| 21 | /admin/user/black_list | GET | AdminUserController.java:106 | admin.go:168 | ✅ |
| 22 | /file/upload | POST | FileController.java:72 | file/handler.go:40 | ✅ |
| 23 | /admin/ad/{id} | DELETE | AdmAdController.java:41 | other/ad_admin.go:24 | ✅ |

状态说明：
- ✅ 端点存在且路径、方法一致
- ❌ Go 中不存在该端点
- ⚠️ 端点存在但路径或方法不一致

Java Controller 本轮确认直接调用 `R.data()` 的活跃端点包括：
- `/api/user/nickname/random`
- `/api/support/{key}`
- `/api/support/list`
- `/api/theme/campus/init`
- `/api/theme/campus`
- `/api/term/list`
- `/api/term`
- `/api/notice/list`

### 差异清单

#### DIFF-INF-01: Go 将 Java 受 JWT/BlackList 保护的多个 `/api/**` 端点暴露为公开接口

- **等级**: P0
- **分类**: 中间件行为
- **Java 证据**: `service-base/src/main/java/com/jb/common/config/InterceptorConf.java:50-75`
```java
registry.addInterceptor(getMyJwt())
    .addPathPatterns("/api/**")
    .excludePathPatterns("/api/user/login")
    .excludePathPatterns("/api/user/refresh")
```
- **Go 证据**: `cmd/ecampus/routes.go:47-68`
```go
pub.GET("/api/support/list", handlers.Other.SupportList)
pub.GET("/api/term/list", handlers.School.TermList)
pub.GET("/api/notice/list", handlers.Other.NoticeList)
```
- **模拟场景**:
  - 输入: `GET /api/support/list`，无 `Authorization` 头，`campus_frontend_support` 集合为空
  - Java 行为: `{"success":false,"code":10005,"msg":"authorization 找不到","data":null}`
  - Go 行为: `{"success":true,"code":200,"msg":"成功","data":null}`
- **预期行为**: 这些 Java 默认受保护的 `/api/**` 端点在 Go 中也应保持同等 JWT 和黑名单保护范围。
- **影响面**: `/api/user/nickname/random`、`/api/theme/campus/init`、`/api/theme/campus`、`/api/support/{key}`、`/api/support/list`、`/api/term/list`、`/api/term`、`/api/notice/list`、`/api/wx/unlimited/wxa_code`

#### DIFF-INF-02: Java `R.data()` 成功响应是 `code=0,msg=""`，Go 统一返回 `code=200,msg="成功"`，空列表还会变成 `null`

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `service-base/src/main/java/com/jb/common/result/Result.java:34-46`，`theme/src/main/java/com/jb/theme/controller/FrontendSupportController.java:47-50`
```java
public Result<T> data(T data) {
    this.setData(data);
    return this;
}
public Result<T> ok() { this.setSuccess(true); return this; }
```
- **Go 证据**: `internal/pkg/result/result.go:44-50`，`internal/other/support.go:18-24`，`internal/other/service_merchant_support_task.go:102-117`
```go
func Success(c *gin.Context, data interface{}) {
    c.JSON(http.StatusOK, Result{Success: true, Code: 200, Msg: "成功", Data: data})
}
```
- **模拟场景**:
  - 输入: `GET /api/support/list`，携带有效 JWT，`campus_frontend_support` 集合为空
  - Java 行为: `{"success":true,"code":0,"msg":"","data":[]}`
  - Go 行为: `{"success":true,"code":200,"msg":"成功","data":null}`
- **预期行为**: Java 走 `R.data()` 的成功返回，在 Go 中也应保持相同的 `code/msg` 语义，并且空列表仍返回 `[]`。
- **影响面**: `/api/user/nickname/random`、`/api/support/{key}`、`/api/support/list`、`/api/theme/campus/init`、`/api/theme/campus`、`/api/term/list`、`/api/term`、`/api/notice/list`

#### DIFF-INF-03: 认证相关 token JSON 契约与 Java 不一致

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `user/src/main/java/com/jb/user/service/impl/UserServiceImpl.java:264-277`，`user/src/main/java/com/jb/user/vo/WXLoginVO.java:14-20`，`user/src/main/java/com/jb/user/service/impl/UserServiceImpl.java:286-317`，`user/src/main/java/com/jb/user/vo/RenewTokenVO.java:12-15`
```java
wxLoginVO.setRefresh_token(tokenHelper.getReToken(j));
wxLoginVO.setCurrentIdentity(buildIdentityVO(activeIdentity));
wxLoginVO.setRootUserId(rootUser.getId());
```
- **Go 证据**: `internal/user/handler.go:21-46`，`internal/user/model_req.go:7-9`，`internal/user/admin.go:19-30`，`internal/user/handler_identity.go:43-54`
```go
type RefreshTokenReq struct {
    RefreshToken string `json:"refreshToken" binding:"required"`
}
result.Success(c, gin.H{"token": token, "refreshToken": refreshToken})
```
- **模拟场景**:
  - 输入 A: `POST /api/user/refresh`，请求体 `{"refresh_token":"r1"}`，其中 `r1` 是有效 refresh token
  - Java 行为: `{"success":true,"code":200,"msg":"成功","data":{"token":"atk2","refresh_token":"rtk2","currentIdentity":{"userId":101,"accountType":"base","nickname":"Alice","avatar":"a.png"}}}`
  - Go 行为: `{"success":false,"code":1,"msg":"参数错误","data":null}`
  - 输入 B: `POST /api/user/login`，请求体 `{"code":"wx-ok"}`
  - Java 行为: `{"success":true,"code":200,"msg":"成功","data":{"token":"atk","refresh_token":"rtk","user":{"id":101},"is_new":false,"currentIdentity":{"userId":101,"accountType":"base","nickname":"Alice","avatar":"a.png"},"rootUserId":101}}`
  - Go 行为: `{"success":true,"code":200,"msg":"成功","data":{"token":"atk","refreshToken":"rtk","user":{"id":101}}}`
- **预期行为**: Go 的认证接口应保持 Java 既有 JSON 字段名和字段集合，尤其是 `refresh_token`、`currentIdentity`、`rootUserId`、`is_new`。
- **影响面**: `/api/user/login`、`/api/user/refresh`、`/admin/user/login`、`/api/user/identity/switch`

#### DIFF-INF-04: Java 对空请求体和校验失败返回专门错误码/HTTP 400，Go 统一回成 `code=1` 且 HTTP 200

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `service-base/src/main/java/com/jb/common/result/exceptionHandler/GlobalException.java:36-68`，`user/src/main/java/com/jb/user/dto/RenewTokenDTO.java:12-14`
```java
@ResponseStatus(HttpStatus.BAD_REQUEST)
@ExceptionHandler(HttpMessageNotReadableException.class)
public Result<?> bodyIsNull(...) {
    return R.fail(RC.ERROR_BODY_IS_NULL);
}
```
- **Go 证据**: `internal/user/handler.go:35-39`
```go
if err := c.ShouldBindJSON(&req); err != nil {
    result.Fail(c, result.CodeParamError, "参数错误")
    return
}
```
- **模拟场景**:
  - 输入: `POST /api/user/refresh`，`Content-Type: application/json`，请求体为空
  - Java 行为: HTTP 400，`{"success":false,"code":7,"msg":"请求体不能为空","data":null}`
  - Go 行为: HTTP 200，`{"success":false,"code":1,"msg":"参数错误","data":null}`
- **预期行为**: 空请求体和校验失败在 Go 中也应保留 Java 已暴露给客户端的状态码和错误码语义。
- **影响面**: 所有依赖 JSON body 的活跃接口，确认示例包括 `/api/user/login`、`/api/user/refresh`、`/api/user/authentication`、`/api/user/re_authentication`、`/api/wx/unlimited/wxa_code`、`/admin/user/login`

#### DIFF-INF-05: AES 加密算法不兼容，Go 无法匹配 Java 已写入的密文

- **等级**: P1
- **分类**: 数据兼容
- **Java 证据**: `service-base/src/main/java/com/jb/common/utils/AESUtil.java:10-32`，`user/src/main/java/com/jb/user/service/impl/AdminServiceImpl.java:145-150`
```java
private static final String TRANSFORMATION = "AES";
Cipher cipher = Cipher.getInstance(TRANSFORMATION);
.eq(User::getStuPwd, AESUtil.encrypt(password, encryptionKey))
```
- **Go 证据**: `internal/pkg/encrypt/aes.go:10-26`，`internal/user/service_admin.go:125-137`
```go
iv := keyBytes[:block.BlockSize()]
mode := cipher.NewCBCEncrypter(block, iv)
Where("stuNum = ? AND stuPwd = ? AND power >= 8", stuNum, encPwd)
```
- **模拟场景**:
  - 输入: AES key `W0F7PePvolUJHmZtkv1MusWpwhpVJIJI`，明文 `test123`
  - Java 行为: `AESUtil.encrypt` 输出 `B9gMpgfaS/4RIi72YJa+tA==`
  - Go 行为: `AESEncrypt` 输出 `fLM/DD9DMyJSTB7lF6sPZA==`
  - 数据库行为:
  - Java 旧管理员兜底登录查询条件是 `stuPwd = 'B9gMpgfaS/4RIi72YJa+tA=='`
  - Go 旧管理员兜底登录查询条件是 `stuPwd = 'fLM/DD9DMyJSTB7lF6sPZA=='`
- **预期行为**: 同一明文和同一 key 在 Go 中必须生成与 Java 一致的密文，保证历史 `stuPwd/loginPassword` 数据可读。
- **影响面**: `/admin/user/login` 的 legacy admin 迁移路径，以及 Go 自身会写入 `stuPwd`/`loginPassword` 的相关认证链路

#### DIFF-INF-06: refresh token 被使用后的 Redis TTL 比 Java 多 24 小时

- **等级**: P2
- **分类**: 业务逻辑
- **Java 证据**: `service-base/src/main/java/com/jb/common/utils/impl/TokenHelperImpl.java:65-68`，`user/src/main/java/com/jb/user/service/impl/UserServiceImpl.java:286-291`
```java
return rds.set(key, TokenStatusCode.USED, jwtConf.getJwt_minutes_re_token() * 60L);
```
- **Go 证据**: `internal/user/service.go:216-226`
```go
if err := s.redis.Set(ctx, refreshKey, rediskey.TokenStatusUsed, 3*24*time.Hour).Err(); err != nil {
```
- **模拟场景**:
  - 输入: refresh token `r1`，其 Redis key 为 `campus:refresh_token:5573e39b6600496d40f493d00ec7658479a19607`。先成功调用一次 `/api/user/refresh`，60 小时后再次复用同一个 `r1`
  - Java 行为: USED 状态只保留 48 小时，60 小时后 key 已过期，再次调用返回 `{"success":false,"code":10006,"msg":"refresh_token 不存在, 或已过期","data":null}`
  - Go 行为: USED 状态保留 72 小时，60 小时后 key 仍存在，再次调用返回 `{"success":false,"code":10007,"msg":"refresh token 已使用","data":null}`
- **预期行为**: refresh token 被消费后的状态和生命周期应与 Java 保持一致，避免同一 token 在相同时间点返回不同错误码。
- **影响面**: `/api/user/refresh`

#### DIFF-INF-07: Go 管理员中间件把 `power>=2` 都当管理员，Java 只认管理员位

- **等级**: P2
- **分类**: 中间件行为
- **Java 证据**: `service-base/src/main/java/com/jb/common/constant/Admin.java:11-14`，`service-base/src/main/java/com/jb/common/interceptor/AdminInterceptor.java:31-41`，`user/src/main/java/com/jb/user/service/impl/AdminTableAuthChecker.java:18-24`
```java
return ((power>>ADMIN_BIT)&1)==1;
boolean isAdminToken = Admin.isAdmin(power);
boolean isAdminUser = adminAuthChecker.isAdmin(userId);
```
- **Go 证据**: `internal/middleware/admin.go:13-40`
```go
isAdminToken := claims.Power >= 2
isAdminUser := count > 0
```
- **模拟场景**:
  - 输入: 请求 `/admin/user/list`，JWT `user_id=10`，`admin` 表存在 `userId=10` 记录
  - Java 行为:
  - `power=0` 拒绝，`power=1` 拒绝，`power=2` 放行，`power=3` 放行，`power=4` 拒绝，`power=8` 拒绝
  - Go 行为:
  - `power=0` 拒绝，`power=1` 拒绝，`power=2` 放行，`power=3` 放行，`power=4` 放行，`power=8` 放行
- **预期行为**: Go 对管理员 token 的判定应与 Java 使用同一权限位语义，而不是把所有 `>=2` 的值都视为管理员。
- **影响面**: 所有 `/admin/**` 端点

#### DIFF-INF-08: 黑名单命中后的返回消息从“权限不足”变成了“账号已被封禁”

- **等级**: P2
- **分类**: 中间件行为
- **Java 证据**: `service-base/src/main/java/com/jb/common/interceptor/BlackListInterceptor.java:41-48`
```java
Boolean blocked = ...isMember(BLACK_LIST_KEY, rootUserId)
if (Boolean.TRUE.equals(blocked)) {
    response.getWriter().print(JSON.toJSON(R.fail(RC.ERROR_FORBIDDEN)));
    return false;
}
```
- **Go 证据**: `internal/middleware/blacklist.go:38-45`
```go
blocked, err := rds.SIsMember(ctx, rediskey.GlobalBlacklist, rootUserID).Result()
if blocked {
    result.Fail(c, result.CodeForbidden, "账号已被封禁")
```
- **模拟场景**:
  - 输入: `GET /api/user`，携带有效 JWT，claims 中 `rootUserId=101`，Redis `campus:global_blacklist` 包含成员 `"101"`
  - Java 行为: `{"success":false,"code":5,"msg":"权限不足","data":null}`
  - Go 行为: `{"success":false,"code":5,"msg":"账号已被封禁","data":null}`
- **预期行为**: 黑名单命中后的返回 JSON 应与 Java 保持一致。
- **影响面**: 所有经过 BlackList 中间件的 `/api/**`、`/admin/**` 端点

#### DIFF-INF-09: `/admin/local_cache/*` 从“本地缓存监控”变成了“Redis 键监控”

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `monitor/src/main/java/com/jb/monitor/controller/admin/LocalCacheController.java:31-57`
```java
return R.data(cacheManager.getCacheNames());
CaffeineCache caffeineCache = (CaffeineCache) cacheManager.getCache(cacheName);
CacheStats stats = caffeineCache.getNativeCache().stats();
map.put("请求次数", stats.requestCount());
```
- **Go 证据**: `internal/monitor/service.go:40-84`
```go
iter := s.redis.Scan(ctx, 0, "*", 500).Iterator()
keys = append(keys, iter.Val())
...
iter := s.redis.Scan(ctx, 0, cacheName+"*", 500).Iterator()
```
- **模拟场景**:
  - 输入: 先请求一次 `/api/term/list` 和 `/api/term`，再请求 `GET /admin/local_cache/all_key`
  - Java 行为: 返回本地缓存名，例如 `{"success":true,"code":0,"msg":"","data":["termList","curTerm"]}`
  - Go 行为: 返回 Redis 实际 key 列表，例如 `{"success":true,"code":200,"msg":"成功","data":["campus:token:...","campus:refresh_token:..."]}`
- **预期行为**: `/admin/local_cache/*` 应继续暴露本地缓存名称和本地缓存统计信息，而不是 Redis key/Redis 全局命中率。
- **影响面**: `/admin/local_cache/all_key`、`/admin/local_cache/stats`

#### DIFF-INF-10: 黑名单管理接口的输入契约从 query 参数变成了 JSON body，并且标识语义从 Java 的 `open_id` 变成了数值用户 ID

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `user/src/main/java/com/jb/user/controller/admin/AdminUserController.java:94-103`，`user/src/main/java/com/jb/user/service/impl/AdminServiceImpl.java:439-467`
```java
public Result<?> addBlackList(@RequestParam List<String> blockedUserIds)
...
User user = userMapper.selectOne(new QueryWrapper<User>().eq("open_id", id));
```
- **Go 证据**: `internal/user/admin.go:142-165`，`internal/user/model_req.go:54-56`
```go
var req UserIDsReq
if err := c.ShouldBindJSON(&req); err != nil {
    result.Fail(c, result.CodeParamError, "参数错误")
}
type UserIDsReq struct {
    UserIDs []int64 `json:"userIds" binding:"required"`
}
```
- **模拟场景**:
  - 输入: `POST /admin/user/add_black_list?blockedUserIds=official:clubA`
  - Java 行为: 进入 Service，按 `open_id=official:clubA` 校验用户存在后更新黑名单，成功时返回 `{"success":true,"code":200,"msg":"黑名单更新成功","data":null}`
  - Go 行为: 因为没有 JSON body，直接返回 `{"success":false,"code":1,"msg":"参数错误","data":null}`
- **预期行为**: 黑名单管理接口的入参绑定方式和标识语义应与 Java 保持一致。
- **影响面**: `/admin/user/add_black_list`、`/admin/user/del_black_list`

#### DIFF-INF-11: `/file/upload` 超限文件不再返回 Java 的 `ERROR_FILE_LIMITED`

- **等级**: P2
- **分类**: API契约
- **Java 证据**: `file/src/main/java/com/jb/file/controller/FileController.java:74-76`
```java
if (file.getSize() > customConf.getFileSize()*MB_SIZE ) {
    return CompletableFuture.completedFuture(R.fail(RC.ERROR_FILE_LIMITED));
}
```
- **Go 证据**: `internal/file/service.go:50-105`，`internal/pkg/config/config.go:75-80`
```go
func (s *Service) Upload(...) (string, string, error) {
    data, err := io.ReadAll(file)
}
type CustomConfig struct {
    MaxFileSizeMB int `mapstructure:"max_file_size_mb"`
}
```
- **模拟场景**:
  - 输入: 已认证用户上传一个 `16MB` 的 JPEG 文件，配置上限为 `15MB`
  - Java 行为: `{"success":false,"code":6,"msg":"File size exceeds the limit.","data":null}`
  - Go 行为: 不会在入口处返回 `code=6`；依赖正常时会继续上传并返回成功结果
- **预期行为**: 文件超限时，Go 应继续暴露 Java 已有的错误码和错误消息。
- **影响面**: `/file/upload`

#### DIFF-INF-12: Java 的 `ERROR_ID_ZERO` 在活跃管理端接口上消失了，`/admin/ad/0` 在 Go 会直接成功

- **等级**: P2
- **分类**: API契约
- **Java 证据**: `other/src/main/java/com/jb/other/controller/admin/AdmAdController.java:41-45`
```java
@DeleteMapping("/{id}")
public Result<?> delete(@PathVariable(value = "id") long id) {
    if (id < 1) {
        return R.fail(RC.ERROR_ID_ZERO);
    }
```
- **Go 证据**: `internal/other/ad_admin.go:24-34`，`internal/other/service_ad.go:17-21`
```go
id, err := strconv.ParseInt(c.Param("id"), 10, 64)
if err := h.svc.DeleteAd(c.Request.Context(), id); err != nil { ... }
result.Success(c, nil)
```
- **模拟场景**:
  - 输入: `DELETE /admin/ad/0`
  - Java 行为: `{"success":false,"code":2,"msg":"id 需要大于0","data":null}`
  - Go 行为: `{"success":true,"code":200,"msg":"成功","data":null}`
- **预期行为**: 对 `id<=0` 的活跃管理端接口，Go 应保留 Java 的显式错误返回，而不是继续执行并返回成功。
- **影响面**: 已确认至少影响 `/admin/ad/{id}`；Java 同样在 `/admin/user/{id}`、`/admin/notice/{id}` 上暴露了相同错误码路径

#### DIFF-INF-13: 黑名单从 Java 的 Mongo 持久化数据源退化成 Go 的纯 Redis 临时数据，缓存清空后列表与拦截效果都会丢失

- **等级**: P1
- **分类**: 数据兼容
- **Java 证据**: `user/src/main/java/com/jb/user/service/impl/AdminServiceImpl.java:327-347`，`user/src/main/java/com/jb/user/service/impl/AdminServiceImpl.java:393-417`
```java
mongoTemplate.upsert(query, update, UserBlacklist.class);
stringRedisTemplate.opsForSet().add(BLACK_LIST_KEY, validUserIds.toArray(new String[0]));
...
UserBlacklist blacklist = mongoTemplate.findOne(query, UserBlacklist.class);
stringRedisTemplate.opsForSet().add(BLACK_LIST_KEY, blockedUserIds.toArray(new String[0]));
```
- **Go 证据**: `internal/user/service_admin_ops.go:78-90`，`internal/user/service_admin_ops.go:114-129`，`internal/user/model.go:59-64`
```go
if err := s.redis.SAdd(ctx, rediskey.GlobalBlacklist, members...).Err(); err != nil { ... }
members, err := s.redis.SMembers(ctx, rediskey.GlobalBlacklist).Result()
type UserBlacklist struct {
    BlockedUserIDs []string `bson:"blocked_user_ids"`
}
```
- **模拟场景**:
  - 输入:
  - 1. 历史 Java 环境中已存在 Mongo 文档 `_id="global_blacklist", blocked_user_ids=["official:clubA"]`
  - 2. 按你的切换前提清空旧 Redis
  - 3. 请求 `GET /admin/user/black_list`
  - Java 行为: 先从 Mongo 回填 Redis，再返回黑名单列表，结果非空
  - Go 行为: 只读 Redis，直接返回 `{"success":true,"code":200,"msg":"成功","data":[]}`
- **预期行为**: 黑名单数据在 Go 中也应具有与 Java 一致的持久化来源和恢复行为，避免清缓存后黑名单丢失。
- **影响面**: `/admin/user/add_black_list`、`/admin/user/del_black_list`、`/admin/user/black_list`，以及 BlackList 中间件的实际拦截效果

### 模块总结

- 活跃端点: 23 个
- Go 已覆盖: 23 个
- P0 差异: 6 个
- P1 差异: 2 个
- P2 差异: 5 个
