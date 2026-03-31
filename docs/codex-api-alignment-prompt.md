# Codex Task: Java → Go API Behavioral Alignment

## Mission

You are aligning the Go rewrite (`Ecampus-go`) with the Java original (`Ecampus`) so that every API endpoint returns **identical JSON structure** — same field names, same null handling, same response wrapper codes. The frontend (WeChat mini-program) and CRM admin panel must work without any changes.

**You MUST read the Java source files as the baseline of truth before making any Go changes.** Do not rely solely on the instructions below — always verify against the actual Java code.

## Project Locations

- **Java baseline (READ-ONLY)**: `../Ecampus/` (relative to this repo root)
  - Entities: `../Ecampus/user-entity/src/main/java/com/jb/userentity/entity/`
  - Controllers: `../Ecampus/*/src/main/java/com/jb/*/controller/`
  - Admin controllers: `../Ecampus/*/src/main/java/com/jb/*/controller/admin/`
  - Services: `../Ecampus/*/src/main/java/com/jb/*/service/impl/`
  - Result classes: `../Ecampus/service-base/src/main/java/com/jb/common/result/`
  - Config (FastJSON): `../Ecampus/service-base/src/main/java/com/jb/common/config/InterceptorConf.java`
  - Interceptors: `../Ecampus/service-base/src/main/java/com/jb/common/interceptor/`
- **Go project (MODIFY)**: current repo root
  - Models: `internal/*/model.go`, `internal/*/model_resp.go`, `internal/*/model_req.go`
  - Handlers: `internal/*/handler*.go`, `internal/*/admin.go`
  - Result package: `internal/pkg/result/`
  - Middleware: `internal/middleware/`
  - Routes: `cmd/ecampus/routes.go`, `cmd/ecampus-crm/routes.go`

## Critical Background

### Java serialization rules (InterceptorConf.java lines 106-118)

Java uses FastJSON with these SerializerFeatures:
- `WriteMapNullValue` — null fields ARE serialized (not skipped)
- `WriteNullStringAsEmpty` — null String → `""`
- `WriteNullNumberAsZero` — null Integer/Long → `0`
- `WriteNullListAsEmpty` — null List → `[]`
- `WriteNullBooleanAsFalse` — null Boolean → `false`

FastJSON serializes Java field names as-is (camelCase). There is no snake_case conversion.

### Java response wrapper (R.java, Result.java, RC.java)

```
R.success(data) → {"success":true, "code":200, "msg":"成功", "data":...}
R.data(data)    → {"success":true, "code":0,   "msg":"",   "data":...}
R.fail(RC.XXX)  → {"success":false,"code":XXX, "msg":"...", "data":null}
R.fail().msg(m) → {"success":false,"code":400, "msg":m,    "data":null}
```

### Go response wrapper (internal/pkg/result/result.go)

```
result.Success(c, data) → code=200, msg="成功"
result.Data(c, data)    → code=0,   msg=""
result.Fail(c, code, msg)
result.SuccessMsg(c, msg, data) → code=200, msg=custom
```

---

## Task 1: JSON Field Name Alignment

### What to do

For every Go struct that appears in API responses (model.go, model_resp.go), compare its `json:"..."` tags with the corresponding Java entity's field names. Java field names = JSON output names (FastJSON default camelCase).

### How to verify each field

1. Find the Java entity class (e.g., `../Ecampus/user-entity/.../User.java`)
2. Look at each `private` field name — that IS the JSON key (unless `@JSONField(name=...)` overrides it)
3. Check if `@JSONField(serialize=false)` hides the field — if so, Go should use `json:"-"`
4. Compare with Go struct's `json:"..."` tag
5. If mismatch, fix the Go json tag. **Never change `gorm:"column:..."` or `bson:"..."` tags.**

### Known mismatches to fix (verify each against Java before changing)

**`internal/user/model.go` — compare with `../Ecampus/user-entity/.../User.java`:**
- `AccountType` json tag: Go `account_type` → Java field `accountType`
- `StuNum` json tag: Go `stu_num` → Java field `stuNum`
- `StuPwd` json tag: Go `stu_pwd` → Java field `stuPwd`
- `StuIsCheck` json tag: Go `stu_is_check` → Java field `stuIsCheck`
- `RootUserID` json tag: Go `root_user_id` → Java field `rootUserId`
- `LastSwitchID` json tag: Go `last_switch_id` → Java field `lastSwitchId`
- `Admin.UserID` json tag: Go `user_id` → Java field `userId`

**`internal/user/model_resp.go` — compare with `../Ecampus/user/.../vo/IdentityVO.java`, `WXLoginVO.java`:**
- `IdentityVO.UserID` json tag: Go `user_id` → Java field `userId`
- `IdentityVO.AccountType` json tag: Go `account_type` → Java field `accountType`
- `IdentityListResp.RootUserID` json tag: Go `root_user_id` → verify Java
- `FollowVO.FollowerID` json tag: Go `follower_id` → check Java `Follow.java` field `followerId`
- `FollowVO.FollowingID`: Go `following_id` → Java `followingId`
- `FollowVO.FollowAt`: Go `follow_at` → Java `followAt`
- NOTE: `LoginResp.RefreshToken` uses `refresh_token` — verify against Java `WXLoginVO.java` field `refresh_token` (Java also uses snake here!)
- NOTE: `LoginResp.IsNew` uses `is_new` — verify against Java `WXLoginVO.java` field `is_new`

**`internal/topic/model.go` — compare with Java Topic MongoDB entity:**
- All `user_id` → should match Java field name (likely `userId`)
- All `account_type` → should match Java (likely `accountType`)
- IMPORTANT: these are MongoDB documents. Check the Java entity's field names or `@Field` annotations.

**`internal/comment/model.go`** — same pattern, check Java Comment entity.

**`internal/chat/model.go`** — check Java Conversation/Message entities:
- `user_id` → `userId`
- `created_at`/`updated_at` → check Java. If Java uses `@JSONField(serialize=false)`, Go should use `json:"-"`.

**`internal/level/model.go`** — check Java Level entity:
- `user_id` → `userId`

**`internal/school/model.go`** — check Java CourseColor/Term entities.

**`internal/event/model.go`** — check Java EventData entity.

**`internal/file/model.go`** — check Java File MongoDB entity.

**`internal/theme/model.go`** — check Java Theme entity:
- `category_name` → likely `categoryName`

**`internal/theme/model_req.go`** — check Java request DTOs.

**`internal/monitor/model.go`** — check Java ControllerTimeRecord entity.

**`internal/mq/model.go`** — check Java MQ message classes:
- `user_id`, `stu_num`, `stu_pwd`, `account_type` → fix to camelCase

**`internal/other/*/model*.go`** — check Java other module entities.

### Full sweep command after changes

```bash
# Should return ONLY: _id (bson), refresh_token, is_new, co_follow (confirmed matching Java)
grep -rn 'json:"[a-z]*_[a-z]' internal/*/model*.go internal/other/*/model*.go | grep -v 'bson:' | grep -v 'gorm:'
```

---

## Task 2: omitempty Alignment

### What to do

Java FastJSON with `WriteMapNullValue` always outputs every field, even if null. Go's `omitempty` skips zero-value fields entirely.

### Rules

1. **Remove `omitempty`** from all response VO json tags EXCEPT:
   - `bson:"_id,omitempty"` — needed for MongoDB insert (this is the bson tag, not json tag)
   - Fields where Java also conditionally omits (verify by reading Java code)
2. **Keep `omitempty`** on `bson` tags for `_id` fields (MongoDB behavior)
3. For fields that are `bson:"xxx,omitempty" json:"xxx"`, the json tag should NOT have omitempty even if bson does

### Known locations to fix

Read each file and verify against Java before changing:

- `internal/user/model_resp.go`: `LoginResp.User`, `LoginResp.CurrentIdentity`, `LoginResp.RootUserID`, `AdminLoginResp.User`, `RefreshTokenResp.CurrentIdentity`, `SwitchIdentityResp.CurrentIdentity`, `SwitchIdentityResp.RootUserID` — all have `omitempty`, verify Java always outputs these
- `internal/mq/model.go`: `SenderUserID`, `TopicID`, `CommentID`, `CreatedTime`, `Reason` — check Java MQ message classes
- `internal/file/model.go`: `CreatedTime` — check Java

### nil slice handling

Go nil slices serialize as `null`, but Java empty lists serialize as `[]`.

Go already has `result.EnsureSlice()` in `internal/pkg/result/page.go` and `normalizeData()` in `result.go`. But nested slices inside structs are not covered.

Check all response structs that contain slice fields:
- `IdentityListResp.Identities []*IdentityVO` — ensure initialized to `[]*IdentityVO{}` when empty
- Any `PageResult.Records` — already handled by `NewPage()`

---

## Task 3: result.Success() vs result.Data() Per-Endpoint Audit

### What to do

For EVERY endpoint in `cmd/ecampus/routes.go` and `cmd/ecampus-crm/routes.go`:

1. Find the corresponding Java controller method
2. Trace what it returns: `R.success(...)`, `R.data(...)`, `R.fail(...)`, or a service method that returns `Result<?>`
3. Check the Go handler uses the matching `result.*()` function
4. Fix if mismatched

### How to trace Java returns

- If the controller returns `R.data(service.doSomething())`, the response code is 0
- If the controller returns `R.success(data)`, the response code is 200
- If the controller returns `service.someMethod()` and that method returns `R.success()` / `R.data()`, trace into the service

### Example

Java `UserController.java`:
```java
@GetMapping
public Result<?> getInfo() {
    User one = userService.getOne(userId);
    one.setStuPwd(null);
    return R.data(one);  // ← code=0
}
```

Go handler should use `result.Data(c, user)`, NOT `result.Success(c, user)`.

### Endpoint list to audit

Read `cmd/ecampus/routes.go` for the full list of user-facing endpoints (~60+).
Read `cmd/ecampus-crm/routes.go` for the full list of admin endpoints (~50+).

---

## Task 4: Time Format

### What to do

1. Read `../Ecampus/service-base/.../InterceptorConf.java` — confirm FastJSON does NOT set a global date format (meaning Date → millisecond timestamp by default)
2. Search Java entities for `@JSONField(format="...")` annotations — fields with this annotation use a custom date format
3. For each Go response VO `time.Time` field that appears in API output:
   - If Java outputs millisecond timestamp → create and use a `Timestamp` type with custom MarshalJSON
   - If Java outputs formatted string → match the format
   - If the field is hidden (`json:"-"`) → no action needed

### Implementation

Create `internal/pkg/result/timestamp.go`:
```go
package result

import (
    "strconv"
    "time"
)

// Timestamp serializes as millisecond epoch (matching Java FastJSON default for Date).
type Timestamp time.Time

func (t Timestamp) MarshalJSON() ([]byte, error) {
    ms := time.Time(t).UnixMilli()
    return []byte(strconv.FormatInt(ms, 10)), nil
}

func (t *Timestamp) UnmarshalJSON(data []byte) error {
    ms, err := strconv.ParseInt(string(data), 10, 64)
    if err != nil {
        return err
    }
    *t = Timestamp(time.Unix(0, ms*int64(time.Millisecond)))
    return nil
}

func (t Timestamp) Time() time.Time {
    return time.Time(t)
}

func NewTimestamp(t time.Time) Timestamp {
    return Timestamp(t)
}
```

Then replace `time.Time` with `result.Timestamp` in response VOs where Java outputs a timestamp.

**Do NOT change GORM model fields** — they must stay `time.Time` for auto-fill to work.
If a GORM model is directly returned in a response (not via a separate VO), you'll need to either create a response VO or add a custom MarshalJSON on the model.

---

## Task 5: Error Message and Middleware Alignment

### 5.1 Error messages

Read Java service implementations (especially `../Ecampus/user/.../service/impl/AdminServiceImpl.java`) and compare every `R.fail().msg("...")` with the corresponding Go error returns. Ensure identical Chinese error text.

### 5.2 Middleware responses

Compare these pairs:
- `../Ecampus/service-base/.../interceptor/JwtInterceptor.java` ↔ `internal/middleware/jwt.go`
- `../Ecampus/service-base/.../interceptor/AdminInterceptor.java` ↔ `internal/middleware/admin.go`
- `../Ecampus/service-base/.../interceptor/BlackListInterceptor.java` ↔ `internal/middleware/blacklist.go`

Check: HTTP status code, response JSON structure, error code, error message.

### 5.3 User entity sanitization

Java `UserController.java:55` does `one.setStuPwd(null)` before returning. Verify Go's GetCurrent handler clears `StuPwd`.

---

## Task 6: Pagination Format

Read how Java's `IPage<T>` (MyBatis-Plus) is serialized. It includes extra fields:
```json
{"records":[], "total":0, "size":10, "current":1, "pages":0,
 "orders":[], "optimizeCountSql":true, "searchCount":true, "countId":null, "maxLimit":null}
```

Go's `PageResult[T]` (`internal/pkg/result/page.go`) only has `records`, `total`, `current`, `size`, `pages`.

**Decision**: Add the missing fields to `PageResult` with default values to match Java. Set:
- `Orders` → `[]any{}` (empty array, not null)
- `OptimizeCountSql` → `true`
- `SearchCount` → `true`
- `CountId` → `nil`
- `MaxLimit` → `nil`

---

## Constraints

1. **Only modify Go files.** Java project is read-only baseline.
2. **Never change `gorm:"column:..."` tags** — they map to database columns.
3. **Never change `bson:"..."` tags** — they map to MongoDB fields.
4. **Only change `json:"..."` tags** for serialization alignment.
5. **Go code must remain idiomatic** — use proper Go naming conventions for struct fields (exported PascalCase), only the json tags change.
6. **Run `go build ./...` after each phase** to ensure compilation.
7. **Run `go test ./...`** to ensure existing tests pass.
8. **When unsure, read the Java source** — it is the single source of truth.

## Execution Order

1. Task 1 (json field names) — highest impact, purely mechanical
2. Task 2 (omitempty and null handling) — high impact
3. Task 3 (Success vs Data audit) — requires careful Java reading
4. Task 4 (time format) — only if relevant fields exist in responses
5. Task 5 (error messages and middleware) — verification-heavy
6. Task 6 (pagination) — lowest priority

## Verification

After all changes:
```bash
go build ./...
go test ./...

# Verify no remaining snake_case json tags (except confirmed matches)
grep -rn 'json:"[a-z]*_[a-z]' internal/*/model*.go internal/other/*/model*.go | grep -v 'bson:' | grep -v 'gorm:'

# Verify no remaining omitempty on response VOs (except confirmed keeps)
grep -rn 'omitempty' internal/*/model_resp.go
```
