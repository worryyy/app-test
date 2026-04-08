# Module Layout Convention

后续各业务模块统一参考 `internal/topic` 的组织方式。

## Core Rule

每个业务模块内部按“主接口”和“业务子域”拆分：

- `service.go`
  只放该模块的主接口。
  典型是增删改查、详情、我的列表、面向主流程的核心能力。
- `service_<业务>.go`
  放非主接口、扩展能力、子域能力。
  例如搜索、社交、审核、统计、同步等。
- `repository.go`
  只负责持有数据库实例，以及暴露基础访问方法。
  例如：
  - `NewRepository(...)`
  - `gormDB(ctx)`
  - `mongoCollection(name)`
  - 模块级常量、基础内部类型
- `repository_<业务>.go`
  放具体数据库操作，按子域拆分。
  例如：
  - `repository_topic.go`
  - `repository_search.go`
  - `repository_social.go`

## Topic As Template

`topic` 模块是后续的标准参考：

- [service.go](/d:/EcampusGO/internal/topic/service.go)
  放主接口：创建帖子、删除帖子、帖子详情、更新帖子、我的帖子、目标用户帖子等。
- [service_search.go](/d:/EcampusGO/internal/topic/service_search.go)
  放搜索能力。
- [service_social.go](/d:/EcampusGO/internal/topic/service_social.go)
  放点赞、收藏等社交能力。
- [repository.go](/d:/EcampusGO/internal/topic/repository.go)
  只放仓储基础结构与数据库实例获取。
- [repository_topic.go](/d:/EcampusGO/internal/topic/repository_topic.go)
  放帖子主数据操作。
- [repository_search.go](/d:/EcampusGO/internal/topic/repository_search.go)
  放搜索相关数据操作。
- [repository_social.go](/d:/EcampusGO/internal/topic/repository_social.go)
  放点赞、收藏相关数据操作。

## Service Split Standard

判断内容是否应留在 `service.go`：

- 属于模块主流程，放 `service.go`
- 属于扩展能力或子业务，放 `service_<业务>.go`

建议放在 `service.go` 的内容：

- Create
- Delete
- Update
- GetByID / Detail
- Mine / ListMine
- 主业务列表接口

建议拆到 `service_<业务>.go` 的内容：

- Search
- Social
- AdminOps
- Identity
- Follow
- Notify
- Extra

## Repository Split Standard

判断内容是否应留在 `repository.go`：

- 只要是“具体查询、插入、更新、删除逻辑”，都不要放在 `repository.go`
- `repository.go` 只保留仓储基座能力

建议保留在 `repository.go` 的内容：

- `Repository` 结构体
- `NewRepository`
- `gormDB`
- `mongoCollection`
- 模块范围内通用常量
- 少量所有子仓储都会复用的基础内部类型

建议拆到 `repository_<业务>.go` 的内容：

- 主实体 CRUD
- 搜索查询
- 关系操作
- 聚合查询
- 统计更新
- 审核/状态流转

## Naming Rule

命名统一使用下面的形式：

- `service.go`
- `service_<business>.go`
- `repository.go`
- `repository_<business>.go`


新增这类文件的前提: 同一模块下可提炼出复用的方法 
- `repository_helpers.go`
- `service_helpers.go`

如果确实只是纯辅助函数，优先判断：

- 能否内聚进对应 `service_<业务>.go` 
- 能否内聚进对应 `repository_<业务>.go`
- 只有跨多个子域复用且职责明确时，再保留 `*_helpers.go`
如果代码不超过10行左右,且代码不可复用 内聚对应service或者repository
如果转换代码超过 10 行或者逻辑复杂，建议提取为私有函数 仍放在同一文件下

## 具体例子
简单的 int64(strconv.Atoi(...)) 且错误处理不复杂 → 内联。

需要多处使用的转换（如 string → time.Time 带默认时区） → 。

只在当前 service 内使用的复杂转换 → 私有方法 convertXXX 放在 service 文件末尾

## Migration Rule

后续模块整理时按下面顺序做：

1. 先确定模块的“主接口”范围
2. 把主接口留在 `service.go`
3. 把非主接口按子域拆到 `service_<业务>.go`
4. 把 `repository.go` 收缩为仓储基座
5. 把所有具体数据库操作迁到 `repository_<业务>.go`

## Default Expectation

从现在开始，除非有特殊说明，新模块和旧模块重构都默认遵守这套规则。
