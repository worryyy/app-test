# Java → Go API 行为对齐方案

## 背景

Go 重写已部署到测试环境，但部分 API 响应与 Java 原版不一致。本文档系统性梳理所有差异点，按优先级排列。

---

## P0: JSON 字段名不一致（前端直接报错）

### 问题

Java 使用 FastJSON，默认按 Java 字段名（camelCase）序列化。Go 的 `json` tag 中部分使用了 snake_case，导致前端解析字段找不到。

### 1.1 User 实体字段名

| Java 输出 | Go 当前输出 | 需改为 |
|-----------|-----------|--------|
| `accountType` | `account_type` | `accountType` |
| `stuNum` | `stu_num` | `stuNum` |
| `stuPwd` | `stu_pwd` | `stuPwd` |
| `stuIsCheck` | `stu_is_check` | `stuIsCheck` |
| `rootUserId` | `root_user_id` | `rootUserId` |
| `lastSwitchId` | `last_switch_id` | `lastSwitchId` |

**不需要改的**（已一致）: `id`, `nickname`, `avatar`, `power`, `stuName`, `stuCla`, `school`, `tag`, `gender`, `signature`

**修改文件**: `internal/user/model.go`

```go
// Before:
AccountType  string  `gorm:"column:account_type;default:base" json:"account_type"`
// After:
AccountType  string  `gorm:"column:account_type;default:base" json:"accountType"`
```

同理修改: `StuNum`, `StuPwd`, `StuIsCheck`, `RootUserID`, `LastSwitchID`

### 1.2 IdentityVO 字段名

| Java 输出 | Go 当前输出 | 需改为 |
|-----------|-----------|--------|
| `userId` | `user_id` | `userId` |
| `accountType` | `account_type` | `accountType` |

**修改文件**: `internal/user/model_resp.go`

```go
// Before:
UserID      int64  `json:"user_id"`
AccountType string `json:"account_type"`
// After:
UserID      int64  `json:"userId"`
AccountType string `json:"accountType"`
```

### 1.3 LoginResp 的 omitempty 问题

Java FastJSON 配置了 `WriteMapNullValue`，null 字段也会输出（零值）。Go 的 `omitempty` 会完全省略这些字段。

**修改文件**: `internal/user/model_resp.go`

```go
// Before:
User            *User       `json:"user,omitempty"`
CurrentIdentity *IdentityVO `json:"currentIdentity,omitempty"`
RootUserID      int64       `json:"rootUserId,omitempty"`
// After:
User            *User       `json:"user"`
CurrentIdentity *IdentityVO `json:"currentIdentity"`
RootUserID      int64       `json:"rootUserId"`
```

对所有 response VO 检查并移除不当的 `omitempty`。

### 1.4 FollowVO 字段名

| Java 输出 | Go 当前输出 | 需改为 |
|-----------|-----------|--------|
| `followerId` (Long) | `follower_id` | `followerId` |
| `followingId` (Long) | `following_id` | `followingId` |
| `followAt` (Date) | `follow_at` | `followAt` |
| `co_follow` (Boolean) | `co_follow` | `co_follow` (一致) |
| N/A | `both_follow` | 确认 Java 是否有此字段 |

**修改文件**: `internal/user/model_resp.go`

### 1.5 UserStatsVO 字段名

Go 当前用 camelCase (`followerCount` 等) — 需确认 Java 端字段名是否一致。

### 1.6 全局排查

对所有 `model_resp.go` / `model.go` 文件执行：
```bash
grep -rn 'json:".*_.*"' internal/*/model*.go
```
逐个与 Java 对应实体的字段名比对。

---

## P0: FastJSON 零值序列化行为

### 问题

Java FastJSON 配置：
- `WriteMapNullValue` → null 字段也序列化
- `WriteNullStringAsEmpty` → null String → `""`
- `WriteNullNumberAsZero` → null Long/Integer → `0`
- `WriteNullListAsEmpty` → null List → `[]`
- `WriteNullBooleanAsFalse` → null Boolean → `false`

Go `encoding/json` 行为：
- 零值类型 (string/int/bool) 自动输出零值 ✅
- 指针类型 (*int64, *string) 为 nil 时输出 `null` ❌
- 切片为 nil 时输出 `null` ❌ (需要 `[]`)
- `omitempty` 会完全跳过零值字段 ❌

### 修复方案

**2.1 nil slice → 空数组**

Go 已有 `result.EnsureSlice()` 和 `result.normalizeData()` 处理顶层 data。但嵌套结构体中的 slice 不会被处理。

需要在所有返回 slice 的地方确保初始化：
```go
// 返回前
if records == nil {
    records = []TopicVO{}
}
```

或在 model 的 MarshalJSON 中统一处理。

**2.2 指针字段零值**

检查所有 response VO 中的指针字段，确认 Java 对应字段是否为 null 或零值。如果 Java 输出零值，Go 不应使用指针类型。

**涉及文件**: 所有 `internal/*/model_resp.go`, `internal/*/model.go`

---

## P1: 时间格式不一致

### 问题

| | Java (FastJSON) | Go (encoding/json) |
|---|---|---|
| 默认格式 | 时间戳 (毫秒 long) 如 `1711785600000` | RFC3339 如 `"2026-03-30T12:00:00+08:00"` |

前端如果用时间戳解析，Go 的 RFC3339 字符串会解析失败。

### 修复方案

**方案 A**: 自定义 JSON 时间类型，输出为毫秒时间戳（推荐，与 Java 完全一致）

```go
// internal/pkg/result/timestamp.go
type Timestamp time.Time

func (t Timestamp) MarshalJSON() ([]byte, error) {
    ms := time.Time(t).UnixMilli()
    return []byte(strconv.FormatInt(ms, 10)), nil
}
```

**方案 B**: 在 Java 侧确认是否有 `@JSONField(format="yyyy-MM-dd HH:mm:ss")` 注解，如果有则 Go 用对应格式。

**需排查**: 每个返回时间字段的 VO，确认 Java 端实际输出格式。

**涉及位置**:
- `FollowVO.FollowAt`
- `OfficialCertificationListItem.ReviewedAt`, `CreatedAt`, `UpdatedAt`
- 所有 MongoDB 文档中的时间字段 (`createdTime`, `handlerTime` 等)
- `NoticeVO.UpdatedAt`

---

## P1: 分页响应格式差异

### 问题

Java MyBatis-Plus `IPage<T>` 序列化后包含额外字段：

```json
{
  "records": [...],
  "total": 100,
  "size": 10,
  "current": 1,
  "pages": 10,
  "orders": [],
  "optimizeCountSql": true,
  "searchCount": true,
  "countId": null,
  "maxLimit": null
}
```

Go `PageResult[T]` 只有核心字段：

```json
{
  "records": [...],
  "total": 100,
  "current": 1,
  "size": 10,
  "pages": 10
}
```

### 修复方案

如果前端不依赖额外字段（大概率不依赖），则 **不需要修改**。

如果前端确实依赖，在 `PageResult` 中补充对齐：

```go
type PageResult[T any] struct {
    Records          []T    `json:"records"`
    Total            int64  `json:"total"`
    Current          int    `json:"current"`
    Size             int    `json:"size"`
    Pages            int    `json:"pages"`
    Orders           []any  `json:"orders"`           // 空数组
    OptimizeCountSql bool   `json:"optimizeCountSql"` // true
    SearchCount      bool   `json:"searchCount"`      // true
    CountId          *int   `json:"countId"`           // null
    MaxLimit         *int   `json:"maxLimit"`          // null
}
```

---

## P1: R.success() vs R.data() 使用场景混用

### 问题

Java 有两种成功响应模式：
- `R.success(data)` → `{"success":true, "code":200, "msg":"成功", "data":...}`
- `R.data(data)` → `{"success":true, "code":0, "msg":"", "data":...}`

Go 也有两种：
- `result.Success(c, data)` → code=200, msg="成功"
- `result.Data(c, data)` → code=0, msg=""

### 修复方案

需逐个 API 对比 Java controller 用的是 `R.success()` 还是 `R.data()`，确保 Go 端使用对应的 `result.Success()` 或 `result.Data()`。

**关键对比列表**:

| API | Java | Go 应使用 |
|-----|------|----------|
| GET /api/user | `R.data(one)` | `result.Data()` |
| POST /api/user/login | `R.success(wxLoginVO)` (在 service 中) | `result.Success()` |
| PUT /api/user | 需检查 | 需检查 |
| GET /api/topic/:id | 需检查 | 需检查 |
| ... | 逐个检查 | ... |

**排查方法**: 对每个 Java controller 方法，确认其直接或间接返回的是 `R.success()` / `R.data()` / `R.fail()`。

---

## P1: 错误码和错误消息不一致

### 问题

Java 部分错误消息是中文硬编码在 service 中，如：
- `R.fail().msg("账号或密码错误，今日还有 X 次机会")`
- `R.fail().msg("权限不足，无法审核")`

Go 使用通用错误类型映射，错误消息可能不同。

### 修复方案

**6.1** 对比 Java RC 枚举与 Go result 常量，确认 code 完全一致（已基本一致）。

**6.2** 对比每个 API 的业务错误分支：
- Java service 中的每个 `R.fail().msg(...)` 调用
- Go service 中对应的 error 返回

确保：
1. 相同的业务场景返回相同的 code
2. 错误消息文本一致（前端可能展示给用户）

**重点排查模块**: `user` (admin login 有大量自定义错误消息)

---

## P2: User 实体序列化字段范围

### 问题

Java User entity 使用 `@JSONField(serialize = false)` 隐藏：
- `openId` ✅ Go 用 `json:"-"` 一致
- `createdBy`, `updatedBy`, `deletedBy` ✅ Go 用 `json:"-"` 一致
- `createdAt`, `updatedAt`, `deletedAt` ✅ Go 用 `json:"-"` 一致

但 Java 的 `stuPwd` 在 controller 中手动 `one.setStuPwd(null)` (见 UserController:55)，FastJSON 会输出 `"stuPwd": ""`。

Go 的 `stuPwd` 字段默认会输出原始值。

### 修复方案

在 Go 的 `sanitizeUser()` 方法中确保 `StuPwd` 被清空：
```go
func (s *Service) sanitizeUser(u *User) *User {
    if u != nil {
        u.StuPwd = ""  // 与 Java 保持一致
    }
    return u
}
```

**检查**: Go 的 `sanitizeUser` 是否已经处理了这个字段。

---

## P2: HTTP Status Code 差异

### 问题

Java GlobalException handler：
- 所有未处理异常 → HTTP 200 + `{"success":false, ...}`
- 参数校验异常 → HTTP 400
- Body 为空 → HTTP 400

Go HandleError：
- 所有错误 → HTTP 200（通过 `result.Fail` 和 `result.Write`）
- BindJSON 错误 → HTTP 400

需确认两者完全一致。

### 修复方案

逐个确认 Go 的错误处理是否与 Java 的 HTTP status code 匹配。

---

## P2: Middleware 行为差异

### 8.1 JWT Interceptor

Java `JwtInterceptor` 拦截 `/api/**` 和 `/admin/**`，返回格式需确认。
Go `middleware/jwt.go` 的拦截路径和响应格式需与 Java 完全一致。

### 8.2 Admin Interceptor

Java `AdminInterceptor` 检查 power 权限位，返回格式需确认。

### 8.3 BlackList Interceptor

Java 黑名单拦截的响应格式和 code 需与 Go 一致。

---

## P2: WebSocket (Chat) 协议差异

Chat 模块使用 WebSocket，需确认：
1. 连接握手路径是否一致 (`/chat`)
2. 消息帧格式 (JSON 结构) 是否一致
3. 心跳机制是否一致
4. 错误处理是否一致

---

## 执行计划

### Phase 1: JSON 兼容性修复（最高优先级，影响所有 API）

1. **[P0] 修复所有 JSON 字段名** — 将所有 model/resp 的 json tag 从 snake_case 改为 camelCase，与 Java 一致
2. **[P0] 移除不当的 omitempty** — 排查所有 response VO
3. **[P0] 修复 nil slice** — 确保所有 slice 字段序列化为 `[]` 而非 `null`

### Phase 2: 响应格式精确对齐

4. **[P1] 时间格式统一** — 实现毫秒时间戳序列化（如果 Java 端使用的是时间戳）
5. **[P1] R.success() vs R.data() 对齐** — 逐 API 检查
6. **[P1] 错误消息文本对齐** — 重点 user admin 模块

### Phase 3: 细节补齐

7. **[P2] 分页格式补齐** — 如前端需要
8. **[P2] User.stuPwd 清空逻辑** — 确认 sanitize 覆盖
9. **[P2] HTTP Status Code 一致性** — 逐 API 确认
10. **[P2] Middleware 响应格式** — JWT/Admin/Blacklist 拦截器

### Phase 4: 集成验证

11. **对比测试** — 同时请求 Java 和 Go 的相同 API，diff JSON 响应
12. **前端联调** — 切换到 Go 后端，验证所有页面功能

---

## 快速验证脚本思路

```bash
# 对每个 API，同时请求 Java 和 Go，diff 响应
JAVA_HOST="http://java-host:8080"
GO_HOST="http://go-host:8080"
TOKEN="<test-token>"

apis=(
  "GET /api/user"
  "GET /api/ad/list_level?level=1"
  "GET /api/notice/list"
  "GET /api/term/list"
  # ... 补充所有 API
)

for api in "${apis[@]}"; do
  METHOD=$(echo $api | cut -d' ' -f1)
  PATH=$(echo $api | cut -d' ' -f2)

  java_resp=$(curl -s -X $METHOD "$JAVA_HOST$PATH" -H "Authorization: Bearer $TOKEN")
  go_resp=$(curl -s -X $METHOD "$GO_HOST$PATH" -H "Authorization: Bearer $TOKEN")

  diff <(echo "$java_resp" | jq -S .) <(echo "$go_resp" | jq -S .)
done
```

---

## 需排查的完整文件清单

| 文件 | 排查内容 |
|------|---------|
| `internal/user/model.go` | json tag camelCase |
| `internal/user/model_resp.go` | json tag camelCase + omitempty |
| `internal/user/model_req.go` | 请求字段名 |
| `internal/topic/model.go` | json tag |
| `internal/topic/model_resp.go` | json tag + omitempty |
| `internal/comment/model.go` | json tag |
| `internal/comment/model_resp.go` | json tag |
| `internal/chat/model.go` | json tag + 消息格式 |
| `internal/chat/model_resp.go` | json tag |
| `internal/level/model.go` | json tag |
| `internal/school/model.go` | json tag |
| `internal/file/model.go` | json tag |
| `internal/other/*/model.go` | json tag |
| `internal/other/*/model_resp.go` | json tag |
| `internal/pkg/result/result.go` | 响应包装器 |
| `internal/pkg/result/page.go` | 分页格式 |
| `internal/middleware/jwt.go` | 拦截响应格式 |
| `internal/middleware/admin.go` | 拦截响应格式 |
| `internal/middleware/blacklist.go` | 拦截响应格式 |
