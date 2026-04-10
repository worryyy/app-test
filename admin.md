# 后台接口与用户接口解耦重构计划

## Summary

- 以 `BACKEND_API_ENDPOINTS.md` 作为后台真实使用接口清单。
- 不改动任何现有用户端 `/api/**` 的对外行为；本次只让后台停止依赖其中指定接口。
- 本批次迁移范围确定为：
  - 全量迁移 Topic 管理能力到 admin 专用接口。
  - 将当前学期查询从 `/api/term` 迁到 `GET /admin/term`。
  - 将预认证从 `/api/user/pre_authentication` 迁到 `/admin/user/pre_authentication`。
  - `/api/user/refresh` 本批次保留共享，不迁移。
- admin 侧禁止使用任何 MQ；Topic 管理逻辑全部走纯同步实现。
- Topic 新接口不采用复数 REST，改用单数风格并尽量收敛为规范路径。
- admin 发帖不走微信审核；保留本地敏感词同步过滤。

## API Changes

- 新增 `GET /admin/topic`，用于后台帖子分页列表，只支持 `page`、`size`，默认按最新倒序，不提供 `/search`。
- 新增 `POST /admin/topic`，用于后台创建帖子。
- 新增 `PATCH /admin/topic/{id}`，用于后台编辑帖子。
- 新增 `DELETE /admin/topic/{id}`，用于后台删除帖子。
- `GET /admin/topic` 不复用用户端 `/api/topic/search` 的热榜、关键词、主题筛选协议。
- `POST /admin/topic` v1 复用现有 Topic 创建字段协议，创建后直接生效，`hasCheck=true`。
- `PATCH /admin/topic/{id}` v1 复用现有 Topic 更新字段协议，做部分更新，编辑后保留原有 `hasCheck` 状态。
- `DELETE /admin/topic/{id}` 采用管理员语义，不受“只能删自己帖子”的用户约束；删除时同步清理 topic 的点赞/收藏关联，并将关联评论同步置为不可见。
- 新增 `GET /admin/term`，承接当前后台对 `/api/term` 的依赖。
- 保留现有 `GET /admin/term/list`，作为 `/api/term/list` 的 admin 替代。
- 新增 `PUT /admin/user/pre_authentication`，沿用现有 query 协议 `user_id`、`nick_name`、`pwd`，只改到 admin 前缀。
- 明确保留 `/api/user/refresh`，本批次不新增 `/admin/user/refresh`。
- 删除未出现在文档中的 admin 路由：
  - `/admin/user/add`
  - `DELETE /admin/user/{id}`
  - `GET /admin/user/{id}`
  - `/admin/user/course`
  - 整组 `/admin/comment/**`
  - 整组 `/admin/theme/**`
  - 整组 `/admin/file/**`
  - `/admin/sensitive/getAllList`
  - `/admin/sensitive/getByWord`
  - `/admin/sensitive/getByWord/`
  - `/admin/sensitive/batchDelete`
  - `/admin/sensitive/batchAdd`
  - `/admin/sensitive/update`

## Implementation Changes

- 重构 `ecampuscrm` 路由分层：
  - 保持 `POST /admin/user/login` 为公开入口。
  - 新增仅挂 `JWTAuth + RequestLog + AdminCheck` 的 `adminAuthOnly` 分组，专门承接 `PUT /admin/user/pre_authentication`。
  - 其余业务型 `/admin/**` 继续走 `JWTAuth + RequestLog + AdminCheck + CertifiedUserCheck`。
- 在 Topic 模块新增独立 admin handler 和 admin route 注册，不把现有用户 handler 直接挂到 `/admin`。
- 在 Topic service/repository 增加 admin 专用同步方法：
  - 分页查询全部帖子。
  - 按帖子 ID 管理员更新。
  - 按帖子 ID 管理员删除并同步清理关联数据。
- admin Topic 创建和编辑继续走本地敏感词过滤，但不接入 RabbitMQ，也不触发微信审核链路。
- `ecampuscrm` 注入 Topic service，并把现有 sensitive filter 接到 Topic admin 逻辑上；不为 crm 打开 RabbitMQ。
- Topic 删除的同步清理逻辑在 CRM 进程内直接完成，不依赖现有 MQ consumer：
  - 从 `campus_topic_like`、`campus_topic_collection` 中移除该 topic 引用。
  - 将 `campus_comment` 中该 topic 关联评论批量置为不可见。
- 移除文档未使用的 admin 模块 wiring、handler、admin-only service/repo 方法、构造函数引用和对应测试；若底层共享方法仍被用户端 `/api` 使用，则只移除 admin 入口和 dead code，不碰共享核心逻辑。
- 更新 `BACKEND_API_ENDPOINTS.md`，把后台已迁移接口改成新的 `/admin/**` 合同，并显式标注 `/api/user/refresh` 仍为保留共享接口。

## Test Plan

- 增加 admin 路由暴露测试，确保保留接口可访问、删除接口不再注册。
- 增加 Topic admin 用例：
  - `GET /admin/topic` 返回分页全量帖子，包含 `hasCheck=true/false`。
  - `POST /admin/topic` 创建后直接可见，且不依赖 MQ。
  - `PATCH /admin/topic/{id}` 编辑后保留原 `hasCheck`。
  - `DELETE /admin/topic/{id}` 删除后同步清理点赞/收藏引用，并同步隐藏关联评论。
- 增加 `GET /admin/term` 和 `PUT /admin/user/pre_authentication` 的路由与权限测试，确认 `pre_authentication` 不被 `CertifiedUserCheck` 误拦截。
- 增加回归测试，确认用户端 `/api/topic*`、`/api/term`、`/api/user/pre_authentication` 旧接口在本批次不被修改行为，`/api/user/refresh` 继续保持原样。

## Assumptions

- `BACKEND_API_ENDPOINTS.md` 已完整覆盖当前后台真实使用的接口，没有文档外的后台消费者。
- Topic admin v1 不新增复杂筛选条件，只支持 `page`、`size` 两个查询参数。
- Topic admin v1 的创建/编辑请求体复用现有 Topic 字段协议，以降低后台调用方迁移成本。
- “admin 不用微信审核”仅表示不走微信/MQ 审核链路，不等于关闭本地敏感词过滤。
- 管理后台前端代码不在当前仓库内，后续以更新后的文档合同同步改造调用方。
