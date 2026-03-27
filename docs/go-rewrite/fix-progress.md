# Ecampus-Go 修复记录

> 说明：
> - 后续修复统一追加到本文件。
> - 为满足仓库文件大小约束，本文件使用紧凑表格记录每批次结果。

## Fix-1: 全局响应契约 + 鉴权契约
| ID | 状态 | 文件/结果 | 验证 |
| --- | --- | --- | --- |
| DIFF-INF-02 | ⚠️ 部分修复 | `internal/other/ad.go`、`internal/other/vote.go`、`internal/other/sensitive_admin.go`、`internal/file/handler.go`、`internal/file/admin.go`、`internal/event/admin.go` 改为走 `result.Data()`；剩余少量与模块载荷耦合端点留待对应批次 | `go build ./cmd/ecampus ./cmd/ecampus-crm`、`go vet ./...`、`go test ./...` |
| DIFF-INF-03 | ✅ 已修复 | `internal/user/model_req.go` 收紧为只接受 `refresh_token`；`internal/user/handler.go` 的 refresh 只读 Java 字段名 | 同上 |
| DIFF-INF-04 | ✅ 已修复 | `internal/pkg/result/result.go` 的 `BindJSON` 改为 HTTP 400；空 body 返回 `code=7`；校验失败返回 `code=1` | 同上 |

**汇总**: 差异 3，已修复 2，部分修复 1，编译通过；遗留为 `DIFF-INF-02` 的少量模块耦合端点。

## Fix-2: 访问控制 + 统一拦截
| ID | 状态 | 文件/结果 | 验证 |
| --- | --- | --- | --- |
| DIFF-INF-01 | ⏭️ 已跳过 | 当前 `cmd/ecampus/routes.go`、`cmd/ecampus-crm/routes.go` 已把审计列出的受保护端点放回 JWT/BlackList 路由组 | 路由代码对比 |
| DIFF-INF-07 | ⏭️ 已跳过 | 当前 `internal/middleware/admin.go` 已使用 `((power>>1)&1)==1`，`power=2/3` 放行，`0/1/4/8` 拒绝 | 中间件代码推演 |
| DIFF-INF-08 | ✅ 已修复 | `internal/middleware/blacklist.go` 命中文案改回 `权限不足` | `go build ./cmd/ecampus && go build ./cmd/ecampus-crm` |
| DIFF-INF-10 | ⏭️ 已跳过 | 当前 `internal/user/admin.go` 读取 query `blockedUserIds`，`internal/user/service_admin_ops.go` 同时支持数值 ID 与 `openId` | Handler/Service 对照 |
| DIFF-INF-13 | ⏭️ 已跳过 | 当前 `internal/user/service_admin_ops.go` 已实现 Mongo 持久化 + Redis 同步 + miss 回填；在线查询 `db.campus_user_blacklist.findOne()` 命中 `_id: "global_blacklist"`，`SMEMBERS campus:global_blacklist` 当前为空集合 | 代码确认 + `mongosh` + `redis-cli` |
| DIFF-CRN-02 | ✅ 已修复 | 新增 `internal/middleware/auth_permission.go`；`cmd/ecampus/main.go`、`cmd/ecampus/routes.go`、`cmd/ecampus-crm/routes.go` 接入统一认证拦截 | `go build ./cmd/ecampus && go build ./cmd/ecampus-crm`、`go vet ./...`、`go test ./...` |
| DIFF-CRN-03 | ✅ 已修复 | 新增 `internal/topic/service_permission.go`，并在 `internal/topic/service.go` 接入商家主题发帖权限检查 | 同上 |

**汇总**: 差异 7，已修复 3，已跳过 4，编译通过。

## Fix-3: 加密兼容 + User 登录/认证链路
| ID | 状态 | 文件/结果 | 验证 |
| --- | --- | --- | --- |
| DIFF-INF-05 | ✅ 已修复 | 新增 `internal/pkg/encrypt/aes_test.go`，锁定 Java 兼容密文 `B9gMpgfaS/4RIi72YJa+tA==`；运行时代码已是 ECB + PKCS5/PKCS7 等价实现 | `jshell`、`go test ./internal/pkg/encrypt -run TestAES -v`、`go build ./cmd/ecampus && go build ./cmd/ecampus-crm` |
| DIFF-USR-01 | ⏭️ 已跳过 | 当前 `internal/user/model.go` + `service_user_helpers.go` 已隐藏敏感字段并补齐 `stuPwd` 脱敏 | 代码确认 |
| DIFF-USR-02 | ⏭️ 已跳过 | 当前 `internal/user/service.go` 登录已恢复 `lastSwitchId` 对应当前身份并返回 `currentIdentity` | 代码确认 |
| DIFF-USR-03 | ⏭️ 已跳过 | 当前 `internal/user/handler.go` 的 `PreAuth` 已回到 query 契约并执行真实更新 | 代码确认 |
| DIFF-USR-04 | ⏭️ 已跳过 | 当前 `internal/user/model_req.go`、`internal/user/model.go`、`internal/user/service_extra.go` 已覆盖官方认证完整字段；在线 `db.campus_official_certification.findOne()` 当前无样本 | 代码确认 + `mongosh` |
| DIFF-USR-05 | ⏭️ 已跳过 | 当前 `OfficialLogin` 已按 Java 审核流产出的官方账号读取 `stuNum + stuPwd(AES)` | 代码确认 |
| DIFF-USR-06 | ⏭️ 已跳过 | 当前认证/重新认证请求体、返回体与落库字段已与 Java 对齐 | 代码确认 |
| DIFF-USR-07 | ⏭️ 已跳过 | 当前 `CheckLogin` 仍走教务校验，不是本地布尔短路 | 代码确认 |
| DIFF-USR-08 | ⏭️ 已跳过 | 当前 `get_course_by_weeks/get_exam/get_exam_score` 已走 `jwClient` 实现 | 代码确认 |
| DIFF-USR-09 | ⏭️ 已跳过 | 当前用户编辑返回完整脱敏用户，且 MQ 更新消息已补齐 `nickName/avatar/gender/signature/accountType` | 代码确认 |
| DIFF-USR-10 | ⏭️ 已跳过 | 当前匿名身份创建无需请求体，自动生成昵称并返回 `IdentityVO` | 代码确认 |
| DIFF-USR-11 | ⏭️ 已跳过 | 当前匿名昵称修改已恢复根账号范围与 72 小时限制 | 代码确认 |
| DIFF-USR-12 | ⏭️ 已跳过 | 当前身份列表返回 `IdentityListResp`，不是 `User[]` | 代码确认 |
| DIFF-USR-13 | ⏭️ 已跳过 | 当前 follow/unfollow 已恢复 `following_id` query 契约与重复/非法状态校验 | 代码确认 |
| DIFF-USR-14 | ⏭️ 已跳过 | 当前 followers/followings 已读取 `targetId` 并返回 `CusPage[FollowVO]` | 代码确认 |
| DIFF-USR-15 | ⏭️ 已跳过 | 当前 `user_profile/stats/is_following` 已恢复 Java 参数名与返回语义 | 代码确认 |
| DIFF-USR-16 | ⏭️ 已跳过 | 当前 `internal/user/service_admin.go` 在配置缺省时回落 Java 二级密码；在线 `DESCRIBE admin` 为 `id/user_id/username/password/power` | 代码确认 + MySQL |
| DIFF-USR-17 | ⏭️ 已跳过 | 当前管理员新增/编辑用户已恢复 `power` 与学籍字段写入 | 代码确认 |
| DIFF-USR-18 | ⏭️ 已跳过 | 当前管理端 clear 不再额外清空 `stuPwd` | 代码确认 |
| DIFF-USR-19 | ⏭️ 已跳过 | 当前管理端课表接口已恢复文件下载语义 | 代码确认 |
| DIFF-USR-20 | ⏭️ 已跳过 | 当前官方认证管理列表/审核流已恢复 Java 模型与通过后建号逻辑；在线集合暂无样本 | 代码确认 + `mongosh` |

**汇总**: 差异 21，已修复 1，已跳过 20，编译通过；本轮唯一代码改动为 AES 固定向量测试。

## Fix-4: Topic + Comment 核心契约
| ID | 状态 | 文件/结果 |
| --- | --- | --- |
| DIFF-TOP-01 | ⏭️ 已跳过 | 当前 `internal/topic/model.go` 的 `accountType` 已为字符串；线上 `campus_topic` 样本文档可正常读取 |
| DIFF-TOP-02 | ⏭️ 已跳过 | 当前 `internal/topic/service.go` 已校验 `themeId` 并自动补齐作者信息，返回完整 Topic |
| DIFF-TOP-03 | ⏭️ 已跳过 | 当前 `internal/topic/service.go` + `service_helpers.go` 已补 `createdTime`，并回填 `hasLike/hasCollection` |
| DIFF-TOP-04 | ⏭️ 已跳过 | 当前 `internal/topic/handler.go` 已兼容 `themeIds/content/ord_created/theme_id/target_user_id` |
| DIFF-TOP-05 | ⏭️ 已跳过 | 当前 `UpdateTopicReq` 为部分更新模型，支持仅提交变更字段 |
| DIFF-TOP-06 | ⏭️ 已跳过 | 当前审计列出的帖子空列表场景均返回 `[]` |
| DIFF-TOP-07 | ⏭️ 已跳过 | 当前 `internal/mq/consumer_topic_comment.go` 审核通过前已过滤二维码图片 |
| DIFF-TOP-08 | ⏭️ 已跳过 | 当前 `internal/topic/service_search.go` 已真正重建推荐榜并清缓存 |
| DIFF-TOP-09 | ⏭️ 已跳过 | 当前 `internal/topic/service_social.go` 已在点赞/收藏他人帖子后发送通知 MQ |
| DIFF-CMT-01 | ⏭️ 已跳过 | 当前 `internal/comment/model.go` 的 `user.accountType` 已为字符串；线上 `campus_comment` 样本文档可正常读取 |
| DIFF-CMT-02 | ⏭️ 已跳过 | 当前 `internal/comment/service_list.go` 已按 `hasCheck != false` 过滤，兼容 Java 老评论 |
| DIFF-CMT-03 | ⏭️ 已跳过 | 当前 `internal/comment/service_helpers.go` 已恢复匿名评论与自评匿名帖限制 |
| DIFF-CMT-04 | ⏭️ 已跳过 | 当前 `internal/comment/service_create.go` 会根据父评论自动推导 `parent/rootCmtId` |
| DIFF-CMT-05 | ⏭️ 已跳过 | 当前 `internal/comment/handler.go` + `service_list.go` 已按 `root_id` 分层查询、排序并回填 `hasLike` |
| DIFF-CMT-06 | ⏭️ 已跳过 | 当前新增评论的用户快照、`isAuthor`、`rootCmtId` 生成逻辑已与 Java 对齐 |
| DIFF-CMT-07 | ⏭️ 已跳过 | 当前 `internal/comment/service_list.go` 已返回 `MyCommentVO{comment,topic}` |
| DIFF-CMT-08 | ⏭️ 已跳过 | 当前 `internal/comment/handler.go` 已兼容 `target_user_id`，空结果返回 `[]` |
| DIFF-CMT-09 | ⏭️ 已跳过 | 当前 `internal/comment/service_like.go` 已在不存在/未点赞等非法状态下返回失败 |
| DIFF-CMT-10 | ✅ 已修复 | `internal/mq/consumer_topic_comment.go` 在内容安全拒绝时删除 pending 评论，避免残留 `hasCheck=false` 文档 |
| DIFF-CMT-11 | ⏭️ 已跳过 | 当前 `internal/other/report.go` 已接受 `reportContent` 并返回创建后的举报对象 |
| DIFF-CMT-12 | ⏭️ 已跳过 | 当前 `internal/other/report_admin.go` + `service_report.go` 已恢复 `handlerContent` 审核协议 |
| DIFF-CMT-13 | ⏭️ 已跳过 | 当前举报列表模型、字段与 `hasHandle` 升序排序已与 Java 对齐 |
| DIFF-CMT-14 | ⏭️ 已跳过 | 当前评论删除成功路径不再顺带清理 `campus_report_comment` |
| DIFF-CMT-15 | ⏭️ 已跳过 | 当前 `NotifyMsg`/`campus_notifications` 落库字段已使用 Java 兼容结构 |

**汇总**: 差异 24，已修复 1，已跳过 23，编译通过；Topic/Comment 线上样本通过 `mongosh` 复核。

## Fix-5: Chat 模块
| ID | 状态 | 文件/结果 | 验证 |
| --- | --- | --- | --- |
| DIFF-CHT-01 | ⏭️ 已跳过 | 当前 `internal/chat/model.go` 已改为 `snake_case` 列映射；在线 `DESCRIBE conversations`、`DESCRIBE conversation_members` 与 Go `gorm` tag 对齐 | MySQL + 代码确认 |
| DIFF-CHT-02 | ⏭️ 已跳过 | 当前 `internal/chat/handler.go` 的 `ConversationEnter` 已恢复 `conversation_id/last_message_id` query 契约，`internal/chat/service_conversation.go` 会同时清零未读并写 `last_read_message_id` | Controller/Service 对照 |
| DIFF-CHT-03 | ⏭️ 已跳过 | 当前 `GetUnreadCount` 返回 `[]ConversationUnreadCount`，不再是裸整数 | 代码确认 |
| DIFF-CHT-04 | ⏭️ 已跳过 | 当前 `conversation_query` 已读取 `target_user_id` 并返回会话 ID 数组，空结果走 `result.Data([]string{})` | 代码确认 |
| DIFF-CHT-05 | ⏭️ 已跳过 | 当前 `profile_by_conversation_id` 已读取 `conversation_id` 并返回轻量 `ConversationProfile`，不暴露完整用户实体 | 代码确认 |
| DIFF-CHT-06 | ⏭️ 已跳过 | 当前删除会话已恢复存在性/成员权限校验，删除成员记录、Mongo 消息，并在空会话时删除会话表记录 | 代码确认 |
| DIFF-CHT-07 | ⏭️ 已跳过 | 当前 `internal/chat/model.go` 的消息 ID/会话 ID/收发件人 ID 已为字符串；在线 `db.campus_messages.findOne()` 样本文档可正常映射 | Mongo + 代码确认 |
| DIFF-CHT-08 | ⏭️ 已跳过 | 当前 `history_messages` 已恢复 `conversation_id`、`oldest_message_id`、成员鉴权、按 `message_id ASC` 与 `limit 50` 语义 | 代码确认 |
| DIFF-CHT-09 | ⏭️ 已跳过 | 当前 `UnreadMessages` 已返回布尔值 `HasUnreadMessages`，不再返回消息列表 | 代码确认 |
| DIFF-CHT-10 | ⏭️ 已跳过 | 当前离线消息拉取已按“用户参与会话 + message_id 游标”同步，不再只按 `receiver_id` | 代码确认 |
| DIFF-CHT-11 | ⏭️ 已跳过 | 当前 `/api/notify` 空结果返回 `result.Data([]Notification{})`，非空时读取列表并原子更新最新通知 `is_read=true`；在线 `db.campus_notifications.findOne()` 字段结构与 Java 一致 | Mongo + 代码确认 |
| DIFF-CHT-12 | ⏭️ 已跳过 | 当前 `HaveUnreadNotification` 已只看该类型最新一条通知的 `isRead` | 代码确认 |
| DIFF-CHT-13 | ✅ 已修复 | 当前 WebSocket 聊天消息链路已兼容 Java；本轮补修 `internal/chat/realtime.go` 的实时通知帧，将在线通知推送恢复为扁平 camelCase 对象，并把 `createdTime` 改为 Java 兼容的毫秒时间戳；新增 `internal/chat/realtime_test.go` 固定序列化用例 | `go build ./cmd/ecampus`、`go build ./cmd/ecampus-crm`、`go vet ./...`、`go test ./internal/chat -run TestRealtimeNotificationMarshal -v`、`go test ./...` |

**汇总**: 差异 13，已修复 1，已跳过 12，编译通过；本轮在线复核了 `conversations`、`conversation_members`、`campus_messages`、`campus_notifications` 实际结构。

## Fix-6: File + School + Level
| ID | 状态 | 文件/结果 | 验证 |
| --- | --- | --- | --- |
| DIFF-FIL-01 | ✅ 已修复 | `internal/file/handler.go` 上传返回改为 `result.Data(UploadResp{path})`；`internal/file/model_req.go` 新增 `UploadResp` | `go build ./cmd/ecampus`、`go build ./cmd/ecampus-crm` |
| DIFF-FIL-02 | ✅ 已修复 | `internal/file/service.go` 恢复 10MB 上限；`internal/file/service_query.go` 恢复 Java 白名单 `image/png,image/jpeg,image/x-icon,application/octet-stream`；`configs/ecampus/application.yml` 的 `max_file_size_mb` 改回 `10`；新增 `internal/file/service_query_test.go` | `go test ./internal/file -run TestNormalizeContentType -v` |
| DIFF-FIL-03 | ✅ 已修复 | `internal/file/service.go` 上传/删除改回按 `userId + md5` 维度管理引用；未命中本人文件时删除返回 `资源不存在`；Mongo 在线样本 `campus_file` 字段为 `isPublic/userId/refCount/md5` | `mongosh db.campus_file.findOne()`、编译通过 |
| DIFF-FIL-04 | ✅ 已修复 | `internal/file/handler.go` 恢复 `show_origin`；`internal/file/service_query.go` 区分压缩图/原图 URL；`internal/pkg/cosutil/client.go` 恢复原图对象 + 压缩对象 key(`md5.webp`) 语义与删除逻辑 | 编译通过 |
| DIFF-FIL-05 | ✅ 已修复 | `internal/file/service.go` 公有文件列表改回直接返回数组；`internal/file/handler.go` 已走 `result.Data([]File)` | 编译通过 |
| DIFF-FIL-06 | ✅ 已修复 | `internal/file/admin.go` 改回读取 query `img_list`，`internal/file/service.go` 按 Mongo `_id` 批量更新并返回“更改 N 条记录” | 编译通过 |
| DIFF-FIL-07 | ✅ 已修复 | `internal/file/service.go` 管理端列表改回数组；`internal/file/model.go` 增加仅输出用 `createdTime`，按 ObjectID 时间生成 | 编译通过 |
| DIFF-SCH-01 | ✅ 已修复 | `internal/school/model_req.go` 的 `course_color` 请求体改回 `colors: []string`；新增 `internal/school/service_course_color.go`，按当前学期课表频次生成 `campus_course_color` 并 upsert；`internal/school/model.go`、`internal/mq/consumer_course_notify.go` 的 `campus_user_course` 列名改回 `user_id/updated_at` | 编译通过；在线 SQL 查询当前环境未发现 `campus_course_color/campus_user_course` 表，无法在共享环境做真实写入验证 |
| DIFF-SCH-02 | ✅ 已修复 | `internal/user/jw_client.go` 的 `JWLoginData` 改回 `is_login` 输出；`internal/user/service_extra.go` 的 `toJWLoginData` 同时兼容 `is_login/isLogin` 输入；新增 `internal/user/jw_client_test.go` 锁定 snake_case | `go test ./internal/user -run 'Test(ToJWLoginDataSupportsSnakeCase|JWLoginDataMarshalUsesSnakeCase)' -v` |
| DIFF-SCH-03 | ✅ 已修复 | 新增 `internal/school/service_term.go`；`AddTerm` 先按 `term` 去重，重复时返回 `code=4,msg=term: xxx已存在`，成功改回返回完整 `Term` 文档；`internal/school/admin.go` 改回 `result.Data(term)` | Mongo 在线样本 `campus_term` 已复核；编译通过 |
| DIFF-SCH-04 | ✅ 已修复 | `internal/school/service_term.go` 删除学期前恢复“当前学期不可删”校验，命中时返回 `请先更新当前学期为其他学期后重新删除` | Mongo 在线样本 `campus_cur_term` 已复核；编译通过 |
| DIFF-SCH-05 | ✅ 已修复 | `internal/school/service_term.go` 设置当前学期后改回返回 `CurTerm` 文档；`internal/school/admin.go` 改回 `result.Data(curTerm)` | 编译通过 |
| DIFF-LVL-01 | ✅ 已修复 | `cmd/ecampus/routes.go` 补回 `/api/testAop` 与 `/api/exp+3/:id`；`internal/level/handler.go`、`internal/level/service.go` 补回对应实现 | `go build ./cmd/ecampus` |
| DIFF-LVL-02 | ✅ 已修复 | `internal/level/model.go`、`internal/level/service.go`、`internal/level/handler.go` 的签到详情改回 `{userId,userExp,signed}` | 编译通过 |
| DIFF-LVL-03 | ✅ 已修复 | `internal/level/service.go` 重复签到改回业务错误 `今日已签到`，不再落成系统错误；签到成功文案恢复为 `签到成功` | 编译通过 |
| DIFF-LVL-04 | ✅ 已修复 | `internal/level/service.go` 去掉签到时额外写 `campus:expDetail:DETAIL_KEY` 与清经验缓存；签到 Redis key 改回 `campus:userSign:{userId}:{yyyyMM}`；在线 Redis `KEYS campus:userSign:*` 现网也采用该格式 | `redis-cli KEYS 'campus:userSign:*'`、编译通过 |
| DIFF-LVL-05 | ✅ 已修复 | `internal/level/handler.go` 批量经验接口改回读取 `userIdList`，缺参返回 `无用户id信息`，返回字段改回 `userExp` | 编译通过 |

**汇总**: 差异 17，已修复 17，编译通过；`go vet ./...`、`go test ./...` 已通过。本轮在线复核了 `campus_file`、`campus_term`、`campus_cur_term`、`exp_detail` 与 `campus:userSign:*`，其中当前共享 MySQL 环境未发现 `campus_course_color/campus_user_course` 表，故 `School` 的课程颜色持久化只能做到代码与基线对齐、无法做真实落库回放。
