# Ecampus Go 重写审计最终汇总

审计日期：2026-03-26

口径说明：
- 本汇总基于 `audit-batch-1` 到 `audit-batch-8` 8 份审计报告去重整理。
- 已剔除用户明确排除的“本地缓存已统一切到 Redis”类问题。
- Batch 8 中对 Batch 6/7 已报告问题的重复交叉验证项未重复计数。
- 端点覆盖统计把 `/api/event/` 与 `/admin/sensitive/getByWord/` 视为“路径不一致”，不计入“缺失/多余”数量。

## 端点覆盖统计

| 服务 | Java 端点数 | Go 端点数 | 缺失 | 多余 |
|------|-----------|----------|------|------|
| ecampus | 88 | 88 | 2 | 2 |
| ecampus-crm (admin) | 67 | 68 | 0 | 1 |

补充：
- 路径不一致但并非缺失的端点有 2 个：`/api/event/`、`/admin/sensitive/getByWord/`
- Go 多余端点为：`ecampus` 的 `/health`、`/metrics`，以及 `ecampus-crm` 的 `/health`

缺失端点清单：

| # | 端点 | 所属模块 | 严重程度评估 |
|---|------|---------|------------|
| 1 | `/api/testAop` | Level | P0 |
| 2 | `/api/exp+3/{id}` | Level | P0 |

## 全部 P0 差异

已按模块排序。

| # | ID | 模块 | 标题 | 影响面 |
|---|-----|------|------|-------|
| 1 | DIFF-INF-01 | 基础设施 + 中间件 | Go 将 Java 受 JWT/BlackList 保护的多个 `/api/**` 端点暴露为公开接口 | 公开访问控制 |
| 2 | DIFF-INF-02 | 基础设施 + 中间件 | Java `R.data()` 成功响应是 `code=0,msg=""`，Go 统一返回 `code=200,msg="成功"`，空列表还会变成 `null` | 全站成功响应契约 |
| 3 | DIFF-INF-03 | 基础设施 + 中间件 | 认证相关 token JSON 契约与 Java 不一致 | 登录/续签/鉴权客户端 |
| 4 | DIFF-INF-04 | 基础设施 + 中间件 | Java 对空请求体和校验失败返回专门错误码/HTTP 400，Go 统一回成 `code=1` 且 HTTP 200 | 全站参数校验失败响应 |
| 5 | DIFF-INF-10 | 基础设施 + 中间件 | 黑名单管理接口的输入契约从 query 参数变成了 JSON body，并且标识语义从 Java 的 `open_id` 变成了数值用户 ID | 管理员黑名单接口 |
| 6 | DIFF-CHT-02 | Chat | `conversation_enter` 从 query 参数改成 JSON body，并丢失 `last_message_id` 回执写入 | 会话进入 |
| 7 | DIFF-CHT-03 | Chat | `unread_count` 返回值从 `ConversationMember[]` 变成了裸整数 | 未读数接口 |
| 8 | DIFF-CHT-04 | Chat | `conversation_query` 的入参与出参都不再兼容 Java 客户端 | 会话查询 |
| 9 | DIFF-CHT-05 | Chat | `profile_by_conversation_id` 的参数名改变，返回对象还扩大成了完整用户实体 | 会话资料查询 |
| 10 | DIFF-CHT-08 | Chat | `history_messages` 的参数、排序、游标和鉴权语义都偏离了 Java | 历史消息 |
| 11 | DIFF-CHT-09 | Chat | `unread_messages` 在 Java 只是布尔标志，Go 改成了消息列表 | 未读消息接口 |
| 12 | DIFF-CHT-11 | Chat | `notify` 列表在空结果形态和已读副作用上都偏离了 Java | 通知列表 |
| 13 | DIFF-CHT-13 | Chat | WebSocket 消息与通知帧协议整体不兼容 Java 客户端 | 聊天与实时通知 |
| 14 | DIFF-CMT-06 | Comment | 新评论的公开字段与 Java 契约不一致 | 评论创建返回体 |
| 15 | DIFF-CMT-07 | Comment | `/api/comment` 返回 `Comment` 分页，Java 返回 `MyCommentVO{comment,topic}` | 我的评论 |
| 16 | DIFF-CMT-08 | Comment | `/api/comment/target_user_comments` 查询参数从 `target_user_id` 变成了 `targetUserId` | 他人评论列表 |
| 17 | DIFF-CMT-11 | Comment | `/api/report_comment` 的请求体和成功返回体都变了 | 举报评论 |
| 18 | DIFF-CMT-12 | Comment | `/admin/report_comment/{id}` 处理举报接口改成了另一套协议 | 管理员处理举报 |
| 19 | DIFF-CMT-13 | Comment | `/admin/report_comment/list` 返回的数据模型与排序都不兼容 | 管理员举报列表 |
| 20 | DIFF-CMT-15 | Comment | 评论通知写入的类型和值结构与 Java 不同 | 评论通知 |
| 21 | DIFF-EVT-01 | Event | `/api/event/` 在 Go 中注册成了 `/api/event`，POST 需要额外重定向 | 前端埋点入口 |
| 22 | DIFF-EVT-04 | Event | `/admin/event/list` 的查询参数和返回结构整体换了协议 | 管理端埋点列表 |
| 23 | DIFF-FIL-01 | File | `/file/upload` 返回体从 `{path}` 变成了 `{md5,url}` | 上传接口 |
| 24 | DIFF-FIL-04 | File | `/file/{md5}` 忽略了 `show_origin`，且上传后的下载目标与 Java 不一致 | 文件下载 |
| 25 | DIFF-FIL-05 | File | `/file` 从“直接返回文件数组”变成了分页对象 | 公开文件列表 |
| 26 | DIFF-FIL-06 | File | `/admin/file` 设置公有图的入参从 query `img_list` 改成了 JSON `md5List` | 管理端文件公开设置 |
| 27 | DIFF-FIL-07 | File | `/admin/file` 从“直接返回文件数组”变成了分页对象，并丢失了 `createdTime` | 管理端文件列表 |
| 28 | DIFF-LVL-01 | Level | Go 缺失 Java 中仍然可调用的 `/api/testAop` 与 `/api/exp+3/{id}` | Level 模块缺失端点 |
| 29 | DIFF-LVL-02 | Level | `/api/getUserSignDetail` 的返回结构从 `{userId,userExp,signed}` 变成了 `{exp,signDays,todaySigned}` | 签到详情 |
| 30 | DIFF-LVL-03 | Level | `/api/sign_in` 在“今日已签到”分支不再返回 Java 的业务错误，而是落成通用系统错误 | 签到接口 |
| 31 | DIFF-LVL-05 | Level | `/api/UserExp` 的 query 参数名和返回字段名都变了 | 经验值批量查询 |
| 32 | DIFF-OTH-01 | Other | `/api/ad/list_level` 把 `size` 排行接口改成了 `level` 过滤接口 | 广告列表 |
| 33 | DIFF-OTH-03 | Other | `/admin/task` 仍由 Java 客户端提交 `{name}` 时，Go 会写出完全不同的任务记录 | 任务管理 |
| 34 | DIFF-OTH-04 | Other | 投票列表与选项返回体把 `0/1` 字段改成了布尔值，并把 `createdBy` 改成了 `userId` | 投票列表/草稿 |
| 35 | DIFF-OTH-05 | Other | `/api/vote` 创建接口不再接受 Java 现网的 `{info, options}` 请求体 | 投票创建 |
| 36 | DIFF-SCH-01 | School | `/api/course_color` 的请求体从颜色数组变成了颜色映射，且 Go 不再写 `campus_course_color` | 课程颜色 |
| 37 | DIFF-SCH-02 | School | JW 登录相关接口把 `is_login` 改成了 `isLogin`，而且会把成功结果反序列化成 `false` | 教务登录 |
| 38 | DIFF-SCH-05 | School | `/admin/term/cur` 的成功返回从当前学期文档变成了 `null` | 当前学期设置 |
| 39 | DIFF-THM-02 | Theme | 编辑主题时，Java 的 `category_name` 请求体会在 Go 中被写成空字符串，且返回体从 Theme 对象变成 `null` | 主题编辑 |
| 40 | DIFF-THM-03 | Theme | `PUT /admin/theme/search` 从批量启用搜索变成单主题切换 | 主题搜索开关 |
| 41 | DIFF-THM-04 | Theme | `POST /admin/theme/suggest` 从批量按主题名配置推荐，变成单主题按 ID 配置 | 主题推荐配置 |
| 42 | DIFF-TOP-02 | Topic | 创建帖子返回体从完整 Topic 变成字符串 ID，且作者信息不再自动补齐 | 发帖接口 |
| 43 | DIFF-TOP-03 | Topic | 帖子详情缺少 `createdTime`，且当前用户的点赞/收藏状态始终不准 | 帖子详情 |
| 44 | DIFF-TOP-04 | Topic | Topic 过滤查询参数名变更，Java 客户端请求会直接走错分支 | 帖子列表/搜索 |
| 45 | DIFF-TOP-05 | Topic | 更新帖子在 Go 中变成必须提交完整创建体，Java 的部分更新请求会失败 | 编辑帖子 |
| 46 | DIFF-TOP-06 | Topic | 多个空列表场景在 Java 返回 `[]`，Go 返回分页对象 | 帖子列表空态 |
| 47 | DIFF-USR-01 | User | 返回 `User` 的接口在 Go 暴露了 Java 隐藏字段，并省略了 `stuPwd` | 用户资料返回体 |
| 48 | DIFF-USR-03 | User | `/api/user/pre_authentication` 从 query 参数更新变成了 JSON 绑定空操作 | 预认证入口 |
| 49 | DIFF-USR-04 | User | `/api/user/official/certification` 的请求体、响应体和 Mongo 文档结构都被缩减了 | 官方认证申请 |
| 50 | DIFF-USR-06 | User | `/api/user/authentication` 和 `/api/user/re_authentication` 的请求体、成功返回和入库字段都与 Java 不一致 | 校园认证/重新认证 |
| 51 | DIFF-USR-07 | User | `/api/user/check_login` 从“校验给定教务凭据”变成了“读取本地 `stuIsCheck` 布尔值” | 教务登录校验 |
| 52 | DIFF-USR-08 | User | `/api/user/get_course_by_weeks`、`/get_exam`、`/get_exam_score` 在 Go 中退化为本地空实现 | 课表/考试/成绩 |
| 53 | DIFF-USR-09 | User | `/api/user` 编辑接口的 HTTP 返回和 MQ 更新内容都缩水了 | 用户资料编辑与同步 |
| 54 | DIFF-USR-10 | User | `/api/user/identity/anonymous` 从“无请求体创建匿名身份”变成了“必须提交昵称且返回完整 User” | 匿名身份创建 |
| 55 | DIFF-USR-12 | User | `/api/user/identity/list` 返回结构从 `IdentityListVO` 变成了原始 `User[]` | 身份列表 |
| 56 | DIFF-USR-13 | User | `/api/user/follow` 和 `/api/user/unfollow` 改了入参绑定，并放宽了重复关注/未关注取关的约束 | 关注/取关 |
| 57 | DIFF-USR-14 | User | `/api/user/followers` 和 `/api/user/followings` 忽略了 `targetId`，且返回项从 `FollowVO` 变成了 `User` | 粉丝/关注列表 |
| 58 | DIFF-USR-15 | User | `/api/user/user_profile`、`/api/user/stats`、`/api/user/is_following` 的 query 参数名和返回语义都变了 | 用户主页统计 |
| 59 | DIFF-USR-16 | User | `/admin/user/login` 当前不会接受 Java 基线的二级密码 | 管理员登录 |
| 60 | DIFF-USR-19 | User | `/admin/user/course` 从“同步文件下载接口”变成了“JSON 确认接口” | 管理员拉取课表 |
| 61 | DIFF-USR-20 | User | `/admin/user/certification/list` 和 `/admin/user/certification/review` 不再使用 Java 的官方认证数据模型和审核流程 | 管理员认证审核 |
| 62 | DIFF-WXS-01 | WX / Sensitive | 小程序码接口从 Base64 字符串变成 PNG 二进制，且把 `page` 从可选改成必填 | 小程序码生成 |
| 63 | DIFF-WXS-02 | WX / Sensitive | 敏感词新增/更新接口从 Query 参数改成 JSON Body | 敏感词新增/更新 |
| 64 | DIFF-WXS-03 | WX / Sensitive | 敏感词批量接口从原始数组改成 `{"words":[...]}` 对象 | 敏感词批量接口 |

## 全部 P1 差异

已按模块排序。

| # | ID | 模块 | 标题 | 影响面 |
|---|-----|------|------|-------|
| 1 | DIFF-INF-05 | 基础设施 + 中间件 | AES 加密算法不兼容，Go 无法匹配 Java 已写入的密文 | 密码/凭据兼容 |
| 2 | DIFF-INF-13 | 基础设施 + 中间件 | 黑名单从 Java 的 Mongo 持久化数据源退化成 Go 的纯 Redis 临时数据，缓存清空后列表与拦截效果都会丢失 | 黑名单持久化 |
| 3 | DIFF-CHT-01 | Chat | Go 仍按 camelCase 列名访问会话表，无法兼容 Java 现网 schema | 会话表兼容 |
| 4 | DIFF-CHT-07 | Chat | Go 无法直接读取 Java 已落库消息文档中的字符串 ID 字段 | 历史消息兼容 |
| 5 | DIFF-CMT-01 | Comment | Go 无法读取 Java 已落库评论中的 `user.accountType` | 评论文档兼容 |
| 6 | DIFF-CMT-02 | Comment | Go 只认 `hasCheck=true`，会把 Java 侧原本可见的评论过滤掉 | 评论可见性 |
| 7 | DIFF-CRN-01 | Cron + AOP | Go 完全缺失 `controller_time` 采集与 10 分钟批量落库链路 | 监控/统计链路 |
| 8 | DIFF-EVT-02 | Event | `/api/event/` 不再由服务端注入当前登录用户，`userId` 写入结果从字符串变成了客户端可控的整型 | 埋点数据可信度 |
| 9 | DIFF-FIL-03 | File | Go 按 `md5` 全局去重，改变了跨用户上传与删除时的 Mongo 写入结果 | 文件引用计数/删除 |
| 10 | DIFF-LVL-04 | Level | `/api/sign_in` 在 Go 中额外写入了经验流水，5 分钟后会把签到变成 `+10 exp` | 经验值数据 |
| 11 | DIFF-MQ-01 | MQ | 用户资料变更后，Go 不再同步更新回复场景中的 `parent.*` 快照 | 评论快照一致性 |
| 12 | DIFF-MQ-02 | MQ | 删帖后的 MQ 清理从“保留软删除记录”变成了“直接硬删帖子和评论” | 帖子/评论数据兼容 |
| 13 | DIFF-OTH-02 | Other | FrontendSupport 读写从 `val/keyDesc` 变成了 `value`，现网 Mongo 文档直接失配 | 前端配置文档兼容 |
| 14 | DIFF-OTH-07 | Other | `/api/vote/vote/{info_id}` 从“追加当天投票记录”变成了“覆盖旧投票记录” | 投票结果兼容 |
| 15 | DIFF-THM-01 | Theme | Go 无法读取现网 Java 主题文档中的 `suggestType` | 主题文档兼容 |
| 16 | DIFF-TOP-01 | Topic | Go 无法读取现网 Java 帖子文档中的 `accountType` | 帖子文档兼容 |
| 17 | DIFF-TOP-08 | Topic | 管理端刷新推荐榜在 Go 中只自增版本号，没有真正重建排行榜 | 推荐榜数据 |
| 18 | DIFF-TOP-09 | Topic | 点赞/收藏别人的帖子时，Go 不再发送通知 MQ | 点赞/收藏通知 |
| 19 | DIFF-USR-05 | User | `/api/user/official/login` 无法读取 Java 审核流创建的官方账号 | 官方账号兼容 |
| 20 | DIFF-USR-18 | User | `/admin/user/clear` 会额外清空 `stuPwd`，破坏了 Java 保留的认证密码数据 | 用户认证数据 |

## 全部 P2 差异

已按模块排序。

| # | ID | 模块 | 标题 | 影响面 |
|---|-----|------|------|-------|
| 1 | DIFF-INF-06 | 基础设施 + 中间件 | refresh token 被使用后的 Redis TTL 比 Java 多 24 小时 | refresh token 生命周期 |
| 2 | DIFF-INF-07 | 基础设施 + 中间件 | Go 管理员中间件把 `power>=2` 都当管理员，Java 只认管理员位 | 管理员鉴权边界 |
| 3 | DIFF-INF-08 | 基础设施 + 中间件 | 黑名单命中后的返回消息从“权限不足”变成了“账号已被封禁” | 黑名单响应文案 |
| 4 | DIFF-INF-11 | 基础设施 + 中间件 | `/file/upload` 超限文件不再返回 Java 的 `ERROR_FILE_LIMITED` | 上传边界错误 |
| 5 | DIFF-INF-12 | 基础设施 + 中间件 | Java 的 `ERROR_ID_ZERO` 在活跃管理端接口上消失了，`/admin/ad/0` 在 Go 会直接成功 | 管理端 ID 校验 |
| 6 | DIFF-CHT-06 | Chat | 删除会话后，Go 不再清理消息和空会话，也不再拦截越权/不存在场景 | 删除会话 |
| 7 | DIFF-CHT-10 | Chat | 离线消息拉取改成了“只看 receiver_id”，丢失 Java 的会话级增量同步语义 | 离线消息同步 |
| 8 | DIFF-CHT-12 | Chat | `haveUnread` 从“看最新一条是否未读”变成了“只要历史里有未读就返回 true” | 未读判断 |
| 9 | DIFF-CMT-03 | Comment | Go 缺少 Java 的评论权限限制，匿名与自评匿名帖都会被放行 | 评论权限 |
| 10 | DIFF-CMT-04 | Comment | Go 直接信任客户端的 `rootCmtId`，回复评论的树结构会写错 | 评论树结构 |
| 11 | DIFF-CMT-05 | Comment | `/api/comment/{topic_id}` 忽略 `root_id`，并且排序与 `hasLike` 语义一起偏离 | 楼中楼列表 |
| 12 | DIFF-CMT-09 | Comment | 评论点赞接口不再对非法状态返回失败 | 点赞边界 |
| 13 | DIFF-CMT-10 | Comment | 内容安全不通过时，Java 不落库，Go 会保留一条 `hasCheck=false` 评论 | 审核失败评论 |
| 14 | DIFF-CMT-14 | Comment | 删除评论时 Go 会额外删除对应举报记录 | 删除评论副作用 |
| 15 | DIFF-CRN-02 | Cron + AOP | `AuthPermissionAOP` 缺失后，未认证用户可以直接调用原本应被统一拦截的非 GET 接口 | 统一认证拦截 |
| 16 | DIFF-CRN-03 | Cron + AOP | `MerchantPermissionAOP` 缺失后，非商家用户可以发布商家专属主题帖子 | 商家主题发帖限制 |
| 17 | DIFF-EVT-03 | Event | `/api/event/` 从“成功即入库”变成了“先写 Redis，最多延迟 15 分钟才入库” | 埋点入库时效 |
| 18 | DIFF-FIL-02 | File | `/file/upload` 的文件大小上限从 10MB 放宽到了 15MB，且丢失了 Java 的类型白名单 | 上传边界 |
| 19 | DIFF-OTH-06 | Other | `/api/vote/draft/{info_id}` 丢失了“仅创建者可见”和 `is_ok` 过滤 | 投票草稿权限 |
| 20 | DIFF-OTH-08 | Other | `/admin/merchant_theme` 新增操作不再幂等，重复提交会生成重复文档 | 商家主题管理 |
| 21 | DIFF-ROUTE-04 | 路由总检 | `/admin/sensitive/getByWord/` 在 Go 中丢了尾斜杠兼容 | 管理端路径兼容 |
| 22 | DIFF-SCH-03 | School | `/admin/term` 对外不再保持 Java 的重复 term 处理结果 | 学期新增边界 |
| 23 | DIFF-SCH-04 | School | `/admin/term/{id}` 在 Go 中可以直接删掉当前学期 | 学期删除边界 |
| 24 | DIFF-THM-05 | Theme | 主题名称过滤从精确匹配变成正则包含匹配 | 主题筛选 |
| 25 | DIFF-TOP-07 | Topic | Java 审核通过前会剔除二维码图片，Go 不会 | 帖子审核边界 |
| 26 | DIFF-USR-02 | User | `/api/user/login` 未恢复根账号的当前身份，始终回到基座账号 | 多身份登录 |
| 27 | DIFF-USR-11 | User | `/api/user/identity/anonymous/nickname` 在基座账号上下文下不再生效，也丢失了 72 小时限制 | 匿名昵称修改 |
| 28 | DIFF-USR-17 | User | `/admin/user/add` 和 `/admin/user/{id}` PUT 可写字段集比 Java 少，管理员无法设置 `power`/学籍字段 | 管理员编辑用户 |

## 全部条件性问题

| # | ID | 模块 | 标题 | 触发条件 |
|---|-----|------|------|---------|
| 1 | DIFF-TOP-C1 | Topic | Go 把 `topic_like` / `topic_collection` 的 `themeName` 语义改成了 `themeId` | 仅在 Java/Go 共享 Mongo，或仍有 Java 侧清理/消费逻辑读取这些集合时触发 |
| 2 | DIFF-TOP-C2 | Topic | 推荐榜 Redis key 空间已改名，旧脚本或混部服务会读不到 | 仅在共享 Redis、共享旧运维脚本或外部消费者仍读旧键名时触发 |
| 3 | DIFF-CHT-C01 | Chat | 现网若存在非数字会话 ID，Go 的 Chat 路由参数解析会整体失败 | 仅在现网历史已存在非数字 `conversation_id` 时触发 |

## 建议修复顺序

1. 先修 `DIFF-INF-02`、`DIFF-INF-03`、`DIFF-INF-04`。这是全局响应/鉴权契约，影响所有客户端会话和大量接口回归判断，且修复后能显著降低后续模块联调噪音。
2. 再修 `DIFF-INF-01`、`DIFF-CRN-02`、`DIFF-CRN-03`。这是统一访问控制问题，继续测试会让大量本应失败的写操作误通过，污染数据库与 MQ 结果。
3. 然后修 User 模块的主登录/认证链路：`DIFF-USR-04`、`DIFF-USR-06`、`DIFF-USR-07`、`DIFF-USR-08`、`DIFF-USR-16`、`DIFF-USR-20`。这些接口是用户进入系统和管理员审核的前置条件，后续多数业务依赖它们。
4. 接着修 Topic / Comment / Chat 的核心契约：`DIFF-TOP-02`、`DIFF-TOP-03`、`DIFF-CMT-06`、`DIFF-CMT-07`、`DIFF-CMT-11`、`DIFF-CHT-02`、`DIFF-CHT-03`、`DIFF-CHT-13`。这是社区主链路，覆盖发帖、评论、消息和通知。
5. 然后修 File / School / Other 的高频 P0：`DIFF-FIL-01`、`DIFF-FIL-04`、`DIFF-SCH-01`、`DIFF-SCH-02`、`DIFF-OTH-01`、`DIFF-OTH-04`、`DIFF-OTH-05`、`DIFF-EVT-01`。这些问题会让上传、课程、投票、埋点等功能直接偏离客户端协议。
6. 最后补齐缺失和监控链路：`DIFF-LVL-01`、`DIFF-CRN-01`。缺失端点容易定位，`controller_time` 则属于数据产出类问题，适合在核心业务契约回正后补齐验证。

依赖说明：
- 先修 `DIFF-INF-03`，再验证 User / Chat / Admin 路径，否则 token 契约差异会阻断后续接口测试。
- 先修 `DIFF-INF-02`、`DIFF-INF-04`，再逐模块核对响应 JSON，否则很多模块 P0 会被统一返回包装差异掩盖。
- 先修 `DIFF-INF-01`、`DIFF-CRN-02`、`DIFF-CRN-03`，再跑 Topic / Comment / Vote / Report 联调，否则会写入不该存在的测试数据。

## 整体评估

- Go 重写的完成度: 60%
- 是否可以进入测试阶段: 否
  原因: 虽然路由覆盖率已经很高，但仍有 64 个去重后的 P0 差异，且分布在全局响应、鉴权、用户认证、帖子评论、聊天协议、文件、课程、投票、埋点等核心主链路。
- 必须在测试前修复的 P0 数量: 64
