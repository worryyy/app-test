# AGENTS.md

面向所有 AI / agent 协作者（Claude Code / Codex / Cursor / Gemini 等）的通用契约。本文件是规范主入口，`CLAUDE.md` 通过 `@AGENTS.md` 导入。

项目：`Ecampus-go`，Go 1.25，双服务（`cmd/ecampus` 用户端 8080 / `cmd/ecampus-crm` 管理端 8081），Gin + GORM + MongoDB + Redis + RabbitMQ。

---

## 0. 行为准则（YOU MUST 遵守）

**Tradeoff**：以下准则偏向谨慎优先。修 typo / 加日志 / 改单个变量名这类琐事可直接做，不需要问。**所有超出"改一行"的改动都走以下 4 条准则。**

### 0.1 Think Before Coding

不要做假设后继续执行；不懂先停下来。

- 请求有 ≥ 2 种合理解释时，列出来让用户选，不要静默挑一种。
- 不清楚就问：命名、边界、字段含义、迁移范围、是否复用 `internal/platform/`。
- 发现更简单的或者更好更优秀的方案要说出来，必要时全网搜索参考最佳实践。
- **Success signal**：澄清问题出现在实现之前，而不是之后。

### 0.2 Simplicity First

只写解决当前问题的最少代码。

- 不加未被要求的配置项 / 开关 / 抽象层。
- 本项目 Repository **不是抽象层**，不要给它套 interface。
- 200 行能写成 50 行就重写。自检：*"高级工程师会不会觉得这过度设计？"*
- **Success signal**：diff 体积与需求规模匹配；没有"顺便加上的可能以后有用"。

### 0.3 Surgical Changes

**YOU MUST** 让每行改动可追溯到用户请求。

- 改 A 模块时 import / 命名 / 错误包装路径**沿用该模块既有做法**，不以"你认为更好"的方式重写相邻代码。
- 修 bug 时不要顺手把 helper 上提到公共层——**迁移类改动独立 PR**。
- 自己改动造成的孤儿 import / 变量才清理；看到预存 dead code 可提一嘴，不删。
- **Success signal**：diff 里没有无关格式化 / 重命名 / 注释修改。

### 0.4 Goal-Driven Execution

多步任务先列 "step → verify" 计划。

- "加字段" → "补请求结构体 + 补响应字段 + 对应 handler 测试通过"
- "修 bug" → "先写能复现的测试，再让它过"
- 跑不起来 / 没自证的改动 = 没完成。
- **Success signal**：完成时能指出"跑了什么命令验证了什么"。

---

## 1. 何时直接做 / 何时先问

**直接做**：单文件单处改动；按 CLAUDE.md 第 8 节起手式新增模块；修 bug 且有现成复现路径；修 typo / 日志。

**IMPORTANT：先问**：
- 跨多个模块的迁移 / 重构
- 涉及接口契约变更（JSON 字段名、HTTP Status 语义、错误码）
- 请求有 2+ 种合理解释
- 修改 `internal/platform/` 的公共能力
- 修改 `internal/app/bootstrap/` 或装配层结构

---

## 2. 完成自检（改完必过）

- [ ] diff 里每行都能追溯到当前请求
- [ ] 分层清晰：Handler 无 DB 访问、Service 参数没有 `*gin.Context`
- [ ] 新依赖在 `internal/app/<service>/app.go` 注入，模块内无全局单例
- [ ] 兼容逻辑 / 字段映射 / 日期格式有窄测试覆盖
- [ ] 本地 `go build ./...` 和相关包 `go test` 通过

---

## 3. Commit / PR 约定

- **Commit 前缀**：`feat:` / `bugfix:` / `refactor:` / `docs:` / `chore:`，半角冒号后直接跟中文描述，**不加空格，不用英文 Conventional Commits 格式**。
- **PR 描述**：说清"改了什么 + 为什么"；测试命令列在"测试"章节。

---

## 4. 延伸阅读

- 项目详细规范：[CLAUDE.md](./CLAUDE.md)
- 模块布局模板：[docs/module-layout.md](./docs/module-layout.md)
- 响应与错误速查：CLAUDE.md 第 3 节
- 本地易踩雷约定：CLAUDE.md 第 4 节
