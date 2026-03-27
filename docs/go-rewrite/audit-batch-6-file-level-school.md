# Batch 6 审计报告：File + Level + School 模块

审计日期：2026-03-26

审计前提：
- 本次重构切换后，不存在 Java 与 Go 双版本同时部署。
- 旧 Redis 缓存允许在切换前清空。
- 主报告仅计入 Go 独立部署场景下必然触发的问题；本轮未发现仅在混部、共享旧缓存或互认旧 token 条件下才成立的问题。

## 模块：File

### 活跃 API 端点清单

| # | 端点 | 方法 | Java 入口 | Go 入口 | 状态 |
|---|------|------|----------|---------|------|
| 1 | `/file/upload` | POST | `file/src/main/java/com/jb/file/controller/FileController.java:71-121` | `internal/file/handler.go:40-53` | ✅ |
| 2 | `/file/{md5}` | GET | `file/src/main/java/com/jb/file/controller/FileController.java:123-156` | `internal/file/handler.go:21-28` | ✅ |
| 3 | `/file/del/{md5}` | DELETE | `file/src/main/java/com/jb/file/controller/FileController.java:159-178` | `internal/file/handler.go:55-62` | ✅ |
| 4 | `/file` | GET | `file/src/main/java/com/jb/file/controller/FileController.java:180-199` | `internal/file/handler.go:30-38` | ✅ |
| 5 | `/admin/file` | POST | `file/src/main/java/com/jb/file/controller/admin/AdmFileController.java:39-61` | `internal/file/admin.go:15-25` | ✅ |
| 6 | `/admin/file` | GET | `file/src/main/java/com/jb/file/controller/admin/AdmFileController.java:63-87` | `internal/file/admin.go:27-35` | ✅ |

状态说明：
- ✅ 端点存在且路径、方法一致
- ❌ Go 中不存在该端点
- ⚠️ 端点存在但路径或方法不一致

### 差异清单

#### DIFF-FIL-01: `/file/upload` 返回体从 `{path}` 变成了 `{md5,url}`

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `file/src/main/java/com/jb/file/controller/FileController.java:87-116`，`file/src/main/java/com/jb/file/vo/FileVO.java:11-16`
```java
FileVO vo = new FileVO();
...
vo.setPath(save.getMd5());
return CompletableFuture.completedFuture(
        R.data(vo)
);
```
- **Go 证据**: `internal/file/handler.go:46-53`
```go
md5Value, url, err := h.svc.Upload(c.Request.Context(), file, header, userID)
...
result.Success(c, gin.H{"md5": md5Value, "url": url})
```
- **模拟场景**:
  - 输入: 已登录用户 `id=101`，`POST /file/upload`，上传 1KB PNG 文件，服务端计算 MD5 为 `900150983cd24fb0d6963f7d28e17f72`
  - Java 行为: `{"success":true,"code":0,"msg":"","data":{"path":"900150983cd24fb0d6963f7d28e17f72"}}`
  - Go 行为: `{"success":true,"code":200,"msg":"成功","data":{"md5":"900150983cd24fb0d6963f7d28e17f72","url":"https://cdn.example.com/900150983cd24fb0d6963f7d28e17f72"}}`
- **预期行为**: 上传接口应继续返回 Java 已公开的 `path` 字段与对应的响应包装语义。
- **影响面**: `/file/upload`

#### DIFF-FIL-02: `/file/upload` 的文件大小上限从 10MB 放宽到了 15MB，且丢失了 Java 的类型白名单

- **等级**: P2
- **分类**: 业务逻辑
- **Java 证据**: `file/src/main/java/com/jb/file/controller/FileController.java:75-86`，`starter/src/main/resources/application-dev.yml:38-46`
```java
if (file.getSize() > customConf.getFileSize()*MB_SIZE ) {
    return CompletableFuture.completedFuture(R.fail(RC.ERROR_FILE_LIMITED));
}
Assert.isTrue(
        IMG_TYPE.contains(contentType.toLowerCase()),
        "图片格式只能是"+ IMG_TYPE);
```
- **Go 证据**: `internal/file/service.go:57-68`，`configs/ecampus/application.yml:42-46`
```go
maxBytes := s.maxUploadBytes()
if maxBytes > 0 && header != nil && header.Size > maxBytes {
    return "", "", result.ErrFileLimited
}
data, err := io.ReadAll(file)
if maxBytes > 0 && int64(len(data)) > maxBytes {
    return "", "", result.ErrFileLimited
}
```
- **模拟场景**:
  - 输入: 已登录用户 `id=101`，`POST /file/upload`，上传 12MB PNG 文件
  - Java 行为: `{"success":false,"code":6,"msg":"File size exceeds the limit.","data":null}`
  - Go 行为: 上传继续执行并返回 `200` 成功响应，因为 Go 配置读取的是 `max_file_size_mb: 15`
- **预期行为**: 上传接口对文件大小与可接受类型的校验边界应保持与现网一致。
- **影响面**: `/file/upload`

#### DIFF-FIL-03: Go 按 `md5` 全局去重，改变了跨用户上传与删除时的 Mongo 写入结果

- **等级**: P1
- **分类**: 数据兼容
- **Java 证据**: `file/src/main/java/com/jb/file/controller/FileController.java:93-112,161-177`，`file/src/main/java/com/jb/file/service/impl/FileServiceImpl.java:78-81`
```java
File f = fileService.findOneWithUserIdMd5(userId, md5);
if(f != null) {
    fileService.fileRefChange(userId, md5, 1);
    ...
}
...
Criteria.where("userId").is(userId)
        .and("md5").is(md5));
```
- **Go 证据**: `internal/file/service.go:72-81,116-149`
```go
err = coll.FindOneAndUpdate(
    ctx,
    bson.M{"md5": md5Str},
    bson.M{"$inc": bson.M{"refCount": 1}},
...
filter := bson.M{"md5": md5Value}
if !force && userID != "" {
    filter["userId"] = userID
}
```
- **模拟场景**:
  - 输入: 用户 `101` 先上传、用户 `202` 再上传同一张图片，随后用户 `202` 调用 `DELETE /file/del/900150983cd24fb0d6963f7d28e17f72`
  - Java 行为:
    - 第 1 次上传写入 `campus_file {md5:"900150...", userId:"101", refCount:1, isPublic:false}`
    - 第 2 次上传写入另一条 `campus_file {md5:"900150...", userId:"202", refCount:1, isPublic:false}`
    - 删除时命中 `userId="202" AND md5="900150..."` 的记录并删除该用户自己的引用
  - Go 行为:
    - 第 1 次上传插入 1 条记录
    - 第 2 次上传执行 `findOneAndUpdate({"md5":"900150..."}, {"$inc":{"refCount":1}})`，不会生成 `userId:"202"` 的独立记录
    - 用户 `202` 删除时按 `{"md5":"900150...","userId":"202"}` 查不到文档，但接口仍返回成功，引用计数也不会回退
- **预期行为**: 上传和删除应保持 Java 现有的按“用户 + md5”维度管理文件引用的结果。
- **影响面**: `/file/upload`、`/file/del/{md5}`，多用户上传相同文件的场景

#### DIFF-FIL-04: `/file/{md5}` 忽略了 `show_origin`，且上传后的下载目标与 Java 不一致

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `file/src/main/java/com/jb/file/controller/FileController.java:128-154`，`file/src/main/java/com/jb/file/service/impl/CosServiceImpl.java:98-100`
```java
if(showOrigin < 1) {
    httpServletResponse.sendRedirect(fileService.getCompressDownloadUrl(file));
    return CompletableFuture.completedFuture(
            ResponseEntity.status(HttpStatus.MOVED_PERMANENTLY).build()
    );
}
```
- **Go 证据**: `internal/file/handler.go:21-28`，`internal/file/service.go:219-235`，`internal/pkg/cosutil/client.go:47-55`
```go
url, err := h.svc.GetDownloadURL(c.Request.Context(), c.Param("md5"))
...
c.Redirect(http.StatusFound, url)
```
```go
if strings.TrimSpace(process) != "" && c.compressClient != nil {
    if _, err := putObject(ctx, c.compressClient, objectKey, data, contentType); err == nil {
        return buildURL(c.cfg.CompressBaseCDN, objectKey), nil
    }
}
```
- **模拟场景**:
  - 输入: `GET /file/900150983cd24fb0d6963f7d28e17f72?show_origin=0`
  - Java 行为: 重定向到压缩图地址，例如 `Location: https://cdn-compress.fangfangfang.top/900150983cd24fb0d6963f7d28e17f72.webp`
  - Go 行为: 始终重定向到原图地址，例如 `Location: https://cdn.example.com/900150983cd24fb0d6963f7d28e17f72`，不会区分 `show_origin`
- **预期行为**: 下载接口应继续支持 Java 现有的 `show_origin` 语义，并把缩略图请求导向压缩对象地址。
- **影响面**: `/file/{md5}`，图片查看与原图/压缩图切换场景

#### DIFF-FIL-05: `/file` 从“直接返回文件数组”变成了分页对象

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `file/src/main/java/com/jb/file/controller/FileController.java:194-198`
```java
Query query = Query.query(Criteria.where("isPublic").is(true));
query.fields().exclude("data");
List<File> files = mongoTemplate.find(query.with(pageRequest), File.class);
return R.data(files);
```
- **Go 证据**: `internal/file/handler.go:30-38`，`internal/file/service.go:153-201`
```go
data, err := h.svc.ListPublic(c.Request.Context(), page, size)
...
result.Success(c, data)
```
- **模拟场景**:
  - 输入: `GET /file?page=1&size=2`
  - Java 行为: `{"success":true,"code":0,"msg":"","data":[{"id":"660000000000000000000001","md5":"m1","isPublic":true,"userId":"101","refCount":1},{"id":"660000000000000000000002","md5":"m2","isPublic":true,"userId":"102","refCount":1}]}`
  - Go 行为: `{"success":true,"code":200,"msg":"成功","data":{"data":[{"id":"660000000000000000000001","md5":"m1","isPublic":true,"userId":"101","refCount":1},{"id":"660000000000000000000002","md5":"m2","isPublic":true,"userId":"102","refCount":1}],"current":1,"total":37,"size":2}}`
- **预期行为**: 公有文件列表接口应继续返回 Java 已公开的数组结构，而不是新增分页包裹层。
- **影响面**: `/file`

#### DIFF-FIL-06: `/admin/file` 设置公有图的入参从 query `img_list` 改成了 JSON `md5List`

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `file/src/main/java/com/jb/file/controller/admin/AdmFileController.java:39-58`
```java
@PostMapping
public Result edit(@RequestParam(value = "img_list")List<String> ids){
    ...
    UpdateResult updateResult = mongoTemplate.updateMulti(
        Query.query(Criteria.where("_id").in(list)),
        new Update().set("isPublic", true),
        File.class);
```
- **Go 证据**: `internal/file/model_req.go:3-5`，`internal/file/admin.go:15-24`，`internal/file/service.go:161-169`
```go
type FilePublicReq struct {
    MD5List []string `json:"md5List" binding:"required"`
}
```
```go
if !result.BindJSON(c, &req) {
    return
}
```
- **模拟场景**:
  - 输入: `POST /admin/file?img_list=660000000000000000000001&img_list=660000000000000000000002`
  - Java 行为: `{"success":true,"code":200,"msg":"更改 2 条记录","data":null}`，并按 Mongo `_id` 批量更新 `isPublic=true`
  - Go 行为: HTTP 400，`{"success":false,"code":7,"msg":"请求体不能为空","data":null}`
- **预期行为**: 管理端设置公有图接口应继续接受 Java 现有的 `img_list` query 参数，并按对应的文件记录标识执行更新。
- **影响面**: `/admin/file` `POST`

#### DIFF-FIL-07: `/admin/file` 从“直接返回文件数组”变成了分页对象，并丢失了 `createdTime`

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `file/src/main/java/com/jb/file/controller/admin/AdmFileController.java:77-86`
```java
List<File> files = mongoTemplate.find(query.with(pageRequest), File.class);
List<File> res = files.stream()
        .filter(Objects::nonNull)
        .peek(o -> o.setCreatedTime(new ObjectId(o.getId()).getDate()))
        .collect(Collectors.toList());
return R.data(res);
```
- **Go 证据**: `internal/file/admin.go:27-35`，`internal/file/service.go:157-201`
```go
data, err := h.svc.List(c.Request.Context(), page, size)
...
result.Success(c, data)
```
- **模拟场景**:
  - 输入: `GET /admin/file?page=1&size=1`
  - Java 行为: `{"success":true,"code":0,"msg":"","data":[{"id":"660000000000000000000001","md5":"m1","isPublic":true,"userId":"101","refCount":1,"createdTime":1711526400000}]}`
  - Go 行为: `{"success":true,"code":200,"msg":"成功","data":{"data":[{"id":"660000000000000000000001","md5":"m1","isPublic":true,"userId":"101","refCount":1}],"current":1,"total":37,"size":1}}`
- **预期行为**: 管理端文件列表应继续返回 Java 现有的数组结构，并保留每条记录上的 `createdTime`。
- **影响面**: `/admin/file` `GET`

### 模块总结

- 活跃端点: 6 个
- Go 已覆盖: 6 个
- P0 差异: 5 个
- P1 差异: 1 个
- P2 差异: 1 个

## 模块：Level

### 活跃 API 端点清单

| # | 端点 | 方法 | Java 入口 | Go 入口 | 状态 |
|---|------|------|----------|---------|------|
| 1 | `/api/getUserSignDetail` | GET | `level/src/main/java/com/jb/level/controller/LevelController.java:35-45` | `internal/level/handler.go:21-28` | ✅ |
| 2 | `/api/testAop` | GET | `level/src/main/java/com/jb/level/controller/LevelController.java:47-52` | — | ❌ 缺失 |
| 3 | `/api/sign_in` | POST | `level/src/main/java/com/jb/level/controller/LevelController.java:54-66` | `internal/level/handler.go:30-36` | ✅ |
| 4 | `/api/exp+3/{id}` | POST | `level/src/main/java/com/jb/level/controller/LevelController.java:68-72` | — | ❌ 缺失 |
| 5 | `/api/UserExp` | GET | `level/src/main/java/com/jb/level/controller/LevelController.java:74-83` | `internal/level/handler.go:38-66` | ✅ |

状态说明：
- ✅ 端点存在且路径、方法一致
- ❌ Go 中不存在该端点
- ⚠️ 端点存在但路径或方法不一致

### 差异清单

#### DIFF-LVL-01: Go 缺失 Java 中仍然可调用的 `/api/testAop` 与 `/api/exp+3/{id}`

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `level/src/main/java/com/jb/level/controller/LevelController.java:47-52,68-72`
```java
@GetMapping("/testAop")
public Result<?> testAop() {
    String a = "这是/testAop接口";
    return R.data(a);
}
...
@PostMapping("/exp+3/{id}")
public Result<?> expPlus3(@PathVariable int id) {
```
- **Go 证据**: `cmd/ecampus/routes.go:128-130`
```go
api.GET("/getUserSignDetail", handlers.Level.GetUserSignDetail)
api.POST("/sign_in", handlers.Level.SignIn)
api.GET("/UserExp", handlers.Level.UserExp)
```
- **模拟场景**:
  - 输入: `GET /api/testAop`
  - Java 行为: `{"success":true,"code":0,"msg":"","data":"这是/testAop接口"}`
  - Go 行为: `404 Not Found`
- **预期行为**: Java 中仍然注册可调用的端点，应在 Go 中继续提供相同路径与响应语义。
- **影响面**: `/api/testAop`、`/api/exp+3/{id}`

#### DIFF-LVL-02: `/api/getUserSignDetail` 的返回结构从 `{userId,userExp,signed}` 变成了 `{exp,signDays,todaySigned}`

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `level/src/main/java/com/jb/level/controller/LevelController.java:37-44`，`level/src/main/java/com/jb/level/vo/UserSignDetailVo.java:13-19`
```java
UserExpVo userExpVo = userExpDao.getThisUserExp(userId);
UserSignDetailVo userSignDetailVo = new UserSignDetailVo();
BeanUtils.copyProperties(userExpVo, userSignDetailVo);
userSignDetailVo.setSigned(userSignRedisService.checkSign(userId));
return R.success().data(userSignDetailVo);
```
- **Go 证据**: `internal/level/model.go:16-19`，`internal/level/service.go:79-108`
```go
type SignDetail struct {
    Exp         int  `json:"exp"`
    SignDays    int  `json:"signDays"`
    TodaySigned bool `json:"todaySigned"`
}
```
- **模拟场景**:
  - 输入: 已登录用户 `id=101`，其当前经验值为 `23`，今天已签到
  - Java 行为: `{"success":true,"code":200,"msg":"成功","data":{"userId":101,"userExp":23,"signed":true}}`
  - Go 行为: `{"success":true,"code":200,"msg":"成功","data":{"exp":23,"signDays":1,"todaySigned":true}}`
- **预期行为**: 签到详情接口应继续返回 Java 已公开的字段集合与字段名。
- **影响面**: `/api/getUserSignDetail`

#### DIFF-LVL-03: `/api/sign_in` 在“今日已签到”分支不再返回 Java 的业务错误，而是落成通用系统错误

- **等级**: P0
- **分类**: 业务逻辑
- **Java 证据**: `level/src/main/java/com/jb/level/dao/impl/UserExpDaoImpl.java:68-75`
```java
if (userSignRedisService.checkSign(userId)) {
    return R.fail().msg("今日已签到");
}
userSignRedisService.sign(userId);
return R.success().msg("签到成功");
```
- **Go 证据**: `internal/level/service.go:52-57`，`internal/pkg/result/result.go:165-170`
```go
already, err := s.redis.GetBit(ctx, key, offset).Result()
if already == 1 {
    return ErrAlreadySigned
}
```
```go
case errors.Is(err, ErrRTKError):
    Fail(c, CodeRTKError, err.Error())
default:
    Fail(c, CodeUnknownError, "系统错误")
```
- **模拟场景**:
  - 输入: 用户 `101` 当天已签过到，再次调用 `POST /api/sign_in`
  - Java 行为: `{"success":false,"code":400,"msg":"今日已签到","data":null}`
  - Go 行为: `{"success":false,"code":-1,"msg":"系统错误","data":null}`
- **预期行为**: 重复签到分支应继续向客户端返回 Java 现有的明确业务提示。
- **影响面**: `/api/sign_in`

#### DIFF-LVL-04: `/api/sign_in` 在 Go 中额外写入了经验流水，5 分钟后会把签到变成 `+10 exp`

- **等级**: P1
- **分类**: 数据兼容
- **Java 证据**: `level/src/main/java/com/jb/level/dao/impl/UserExpDaoImpl.java:69-75`，`level/src/main/java/com/jb/level/redis/UserSignRedisService.java:21-26`
```java
if (userSignRedisService.checkSign(userId)) {
    return R.fail().msg("今日已签到");
}
userSignRedisService.sign(userId);
return R.success().msg("签到成功");
```
- **Go 证据**: `internal/level/service.go:60-75`，`internal/cron/exp_flush.go:33-65`
```go
if err := s.redis.SetBit(ctx, key, offset, 1).Err(); err != nil {
    return fmt.Errorf("set sign bit: %w", err)
}
detail := ExpDetail{UserID: userID, GetExpDate: time.Now(), GetExp: 10}
...
if err := s.redis.LPush(ctx, rediskey.ExpDetailKey, string(data)).Err(); err != nil {
```
```go
if err := j.db.WithContext(ctx).CreateInBatches(list[i:end], 1000).Error; err != nil {
    return fmt.Errorf("batch insert exp details: %w", err)
}
```
- **模拟场景**:
  - 输入: 2026-03-26，用户 `101` 尚未签到，调用 `POST /api/sign_in`
  - Java 行为:
    - Redis 写入: `SETBIT campus:userSign:101:202603 25 1`
    - 不会向 `campus:expDetail:DETAIL_KEY` 写入经验流水
    - 5 分钟、10 分钟后都不会因为本次签到新增 `exp_detail` 记录
  - Go 行为:
    - Redis 写入: `SETBIT campus:userSign:202603 3157 1`
    - 追加 `LPUSH campus:expDetail:DETAIL_KEY {"userId":101,"getExp":10,...}`
    - 定时任务 `0 */5 * * * *` 会把该流水刷入 MySQL `exp_detail`，之后 `/api/getUserSignDetail` 与 `/api/UserExp` 都会看到额外的 `+10`
- **预期行为**: 签到接口的 Redis/MySQL 副作用应继续与 Java 当前现网保持一致，不应额外制造经验值增长。
- **影响面**: `/api/sign_in`、`/api/getUserSignDetail`、`/api/UserExp`

#### DIFF-LVL-05: `/api/UserExp` 的 query 参数名和返回字段名都变了

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `level/src/main/java/com/jb/level/controller/LevelController.java:74-82`，`level/src/main/java/com/jb/level/vo/UserExpVo.java:13-17`
```java
public Result<?> getUserExp(@RequestParam(value = "userIdList", required = false) List<Long> userIdList) {
    if (userIdList == null || userIdList.isEmpty()) {
        return R.fail().msg("无用户id信息");
    }
    List<UserExpVo> userExpList = userExpDao.getUserExp(userIdList);
```
- **Go 证据**: `internal/level/handler.go:38-65`
```go
raw := c.Query("userIds")
if strings.TrimSpace(raw) == "" {
    result.Success(c, []map[string]interface{}{})
    return
}
...
out = append(out, map[string]interface{}{"userId": id, "exp": data[id]})
```
- **模拟场景**:
  - 输入: `GET /api/UserExp?userIdList=101&userIdList=202`
  - Java 行为: `{"success":true,"code":200,"msg":"成功","data":[{"userId":101,"userExp":23},{"userId":202,"userExp":0}]}`
  - Go 行为: `{"success":true,"code":200,"msg":"成功","data":[]}`
- **预期行为**: 批量经验接口应继续接受 Java 现有的 `userIdList` 参数名，并返回 `userExp` 字段。
- **影响面**: `/api/UserExp`

### 模块总结

- 活跃端点: 5 个
- Go 已覆盖: 3 个
- P0 差异: 4 个
- P1 差异: 1 个
- P2 差异: 0 个

## 模块：School

说明：
- 本模块除 `school/controller/**`、`school/controller/admin/**` 自有端点外，还包含 `user/controller/UserController.java` 中实际调用 School/JW 能力的活跃入口。

### 活跃 API 端点清单

| # | 端点 | 方法 | Java 入口 | Go 入口 | 状态 |
|---|------|------|----------|---------|------|
| 1 | `/api/user/authentication` | POST | `user/src/main/java/com/jb/user/controller/UserController.java:87-96` | `internal/user/handler.go:148-170` | ✅ |
| 2 | `/api/user/re_authentication` | POST | `user/src/main/java/com/jb/user/controller/UserController.java:98-108` | `internal/user/handler.go:172-194` | ✅ |
| 3 | `/api/user/check_login` | POST | `user/src/main/java/com/jb/user/controller/UserController.java:120-129` | `internal/user/handler.go:204-225` | ✅ |
| 4 | `/api/user/get_course_by_weeks` | POST | `user/src/main/java/com/jb/user/controller/UserController.java:132-152` | `internal/user/handler.go:228-254` | ✅ |
| 5 | `/api/user/get_exam` | POST | `user/src/main/java/com/jb/user/controller/UserController.java:154-174` | `internal/user/handler.go:256-282` | ✅ |
| 6 | `/api/user/get_exam_score` | POST | `user/src/main/java/com/jb/user/controller/UserController.java:176-196` | `internal/user/handler.go:284-310` | ✅ |
| 7 | `/api/course_color` | POST | `school/src/main/java/com/jb/school/controller/CourseColorController.java:28-30` | `internal/school/handler.go:45-54` | ✅ |
| 8 | `/api/term/list` | GET | `school/src/main/java/com/jb/school/controller/TermController.java:39-45` | `internal/school/handler.go:20-27` | ✅ |
| 9 | `/api/term` | GET | `school/src/main/java/com/jb/school/controller/TermController.java:47-70` | `internal/school/handler.go:29-43` | ✅ |
| 10 | `/admin/term` | POST | `school/src/main/java/com/jb/school/controller/admin/AdmTermController.java:43-54` | `internal/school/admin.go:15-26` | ✅ |
| 11 | `/admin/term/{id}` | DELETE | `school/src/main/java/com/jb/school/controller/admin/AdmTermController.java:56-66` | `internal/school/admin.go:28-34` | ✅ |
| 12 | `/admin/term/cur` | POST | `school/src/main/java/com/jb/school/controller/admin/AdmTermController.java:68-96` | `internal/school/admin.go:36-46` | ✅ |

状态说明：
- ✅ 端点存在且路径、方法一致
- ❌ Go 中不存在该端点
- ⚠️ 端点存在但路径或方法不一致

### 差异清单

#### DIFF-SCH-01: `/api/course_color` 的请求体从颜色数组变成了颜色映射，且 Go 不再写 `campus_course_color`

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `school/src/main/java/com/jb/school/model/dto/CourseColorDTO.java:18-21`，`school/src/main/java/com/jb/school/service/impl/CourseColorServiceImpl.java:49-105`，`school/src/main/resources/mapper/CourseColorMapper.xml:4-12`
```java
public class CourseColorDTO implements Serializable {
    @NotNull
    private List<String> colors;
}
```
```java
for(String name : courseNames) {
    CourseColor courseColor = CourseColor.builder()
        .userId(userId).courseName(name).color(colors.get(idx)).build();
```
- **Go 证据**: `internal/school/model_req.go:3-5`，`internal/school/service.go:149-158`
```go
type CourseColorReq struct {
    Colors map[string]string `json:"colors" binding:"required"`
}
```
```go
key := fmt.Sprintf("campus:courseColor:%d", userID)
if err := s.redis.Set(ctx, key, string(data), 0).Err(); err != nil {
    return fmt.Errorf("set course color: %w", err)
}
```
- **模拟场景**:
  - 输入: 用户 `101` 已导入当前学期课表，`POST /api/course_color` 请求体为 `{"colors":["#ff0000","#00ff00"]}`
  - Java 行为:
    - HTTP 响应: `{"success":true,"code":200,"msg":"成功","data":null}`
    - MySQL 写入: 根据当前周课程频次生成并 upsert 例如 `campus_course_color(user_id=101, course_name="高等数学", color="#ff0000")`、`campus_course_color(user_id=101, course_name="大学英语", color="#00ff00")`
  - Go 行为: HTTP 400，`{"success":false,"code":1,"msg":"参数错误","data":null}`
- **预期行为**: 课程颜色接口应继续接受 Java 现有的颜色数组请求体，并产生与现网一致的课程颜色持久化结果。
- **影响面**: `/api/course_color`

#### DIFF-SCH-02: JW 登录相关接口把 `is_login` 改成了 `isLogin`，而且会把成功结果反序列化成 `false`

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `school/src/main/java/com/jb/school/model/pojo/JWLoginResp.java:15-18`，`user/src/main/java/com/jb/user/service/impl/UserServiceImpl.java:380-402,446-449`
```java
public class JWLoginResp implements Serializable {
    @JSONField(name = "is_login")
    Boolean isLogin;
```
```java
JWLoginResp loginResp = gson.fromJson(gson.toJson(login.getData()), JWLoginResp.class);
return R.success(loginResp).msg("认证成功");
```
- **Go 证据**: `internal/user/jw_client.go:30-35`，`internal/user/service_extra.go:267-276`
```go
type JWLoginData struct {
    IsLogin bool   `json:"isLogin"`
    Major   string `json:"major"`
    Message string `json:"message"`
    Name    string `json:"name"`
}
```
```go
var out JWLoginData
if err := json.Unmarshal(raw, &out); err != nil {
    return nil, err
}
```
- **模拟场景**:
  - 输入: `POST /api/user/check_login {"schoolId":"2023001","password":"jw-pass"}`，且 JW 服务返回 `{"code":200,"message":"success","data":{"is_login":true,"major":"计算机科学与技术","name":"张三"}}`
  - Java 行为: `{"success":true,"code":200,"msg":"认证成功","data":{"is_login":true,"major":"计算机科学与技术","name":"张三"}}`
  - Go 行为: `{"success":true,"code":200,"msg":"认证成功","data":{"isLogin":false,"major":"计算机科学与技术","message":"","name":"张三"}}`
- **预期行为**: 认证与登录检查接口应继续向客户端返回 Java 现有的 JW 登录结果字段名和字段值。
- **影响面**: `/api/user/authentication`、`/api/user/re_authentication`、`/api/user/check_login`

#### DIFF-SCH-03: `/admin/term` 对外不再保持 Java 的重复 term 处理结果

- **等级**: P2
- **分类**: 业务逻辑
- **Java 证据**: `school/src/main/java/com/jb/school/controller/admin/AdmTermController.java:45-53`，`school/src/main/java/com/jb/school/service/impl/TermServiceImpl.java:24-27`
```java
boolean b = termService.CheckExistedWithTerm(dto.getTerm());
if(b) {
    return R.fail(RC.ERROR_REPEATED).msg("term: " + dto.getTerm()+"已存在");
}
BeanUtils.copyProperties(dto, term);
Term data = mongoTemplate.save(term);
```
- **Go 证据**: `internal/school/admin.go:15-25`，`internal/school/service.go:99-108`
```go
var req Term
if !result.BindJSON(c, &req) {
    return
}
id, err := h.svc.AddTerm(c.Request.Context(), &req)
```
```go
res, err := s.mongoDB.Collection("campus_term").InsertOne(ctx, term)
```
- **模拟场景**:
  - 输入: `campus_term` 中已存在 `term="2025-2026-1"`，再次提交 `POST /admin/term {"term":"2025-2026-1","startDate":"2025-09-01","totalWeeks":20}`
  - Java 行为: `{"success":false,"code":4,"msg":"term: 2025-2026-1已存在","data":null}`
  - Go 行为: 代码路径会继续尝试 `InsertOne`；在未额外配置 term 唯一索引的情况下，返回 `{"success":true,"code":200,"msg":"成功","data":"<new-object-id>"}`，即使数据库侧有唯一索引，最终响应也会变成通用系统错误而不是 Java 的重复提示
- **预期行为**: 学期新增接口应继续对重复 term 给出与 Java 一致的业务结果。
- **影响面**: `/admin/term`

#### DIFF-SCH-04: `/admin/term/{id}` 在 Go 中可以直接删掉当前学期

- **等级**: P2
- **分类**: 业务逻辑
- **Java 证据**: `school/src/main/java/com/jb/school/controller/admin/AdmTermController.java:56-65`，`school/src/main/java/com/jb/school/service/impl/TermServiceImpl.java:49-56`
```java
if(termService.checkIfCur(id)) {
    return R.fail(RC.ERROR_PARAM_IS_ERROR).msg("请先更新当前学期为其他学期后重新删除");
}
if(termService.deleteOne(id)) {
    return R.success();
}
```
- **Go 证据**: `internal/school/admin.go:28-34`，`internal/school/service.go:111-120`
```go
if err := h.svc.DeleteTerm(c.Request.Context(), c.Param("id")); err != nil {
    result.HandleError(c, err)
    return
}
result.Success(c, nil)
```
- **模拟场景**:
  - 输入: `DELETE /admin/term/660000000000000000000001`，且该 term 当前正被 `campus_cur_term.term` 指向
  - Java 行为: `{"success":false,"code":1,"msg":"请先更新当前学期为其他学期后重新删除","data":null}`
  - Go 行为: `{"success":true,"code":200,"msg":"成功","data":null}`，并直接删除 `campus_term` 文档
- **预期行为**: 删除学期接口应继续阻止删除当前学期。
- **影响面**: `/admin/term/{id}`、`/api/term`

#### DIFF-SCH-05: `/admin/term/cur` 的成功返回从当前学期文档变成了 `null`

- **等级**: P0
- **分类**: API契约
- **Java 证据**: `school/src/main/java/com/jb/school/controller/admin/AdmTermController.java:77-95`
```java
if(all.isEmpty()) {
    CurTerm save = mongoTemplate.save(curTerm);
    return R.data(save);
}
...
return R.data(mongoTemplate.findById(id, CurTerm.class));
```
- **Go 证据**: `internal/school/admin.go:36-46`
```go
if err := h.svc.SetCurrentTerm(c.Request.Context(), req.TermID); err != nil {
    result.HandleError(c, err)
    return
}
result.Success(c, nil)
```
- **模拟场景**:
  - 输入: `POST /admin/term/cur {"termId":"660000000000000000000001"}`
  - Java 行为: `{"success":true,"code":0,"msg":"","data":{"id":"670000000000000000000001","term":"2025-2026-1"}}`
  - Go 行为: `{"success":true,"code":200,"msg":"成功","data":null}`
- **预期行为**: 设置当前学期接口成功后，应继续返回 Java 现有的当前学期文档。
- **影响面**: `/admin/term/cur`

### 模块总结

- 活跃端点: 12 个
- Go 已覆盖: 12 个
- P0 差异: 3 个
- P1 差异: 0 个
- P2 差异: 2 个
