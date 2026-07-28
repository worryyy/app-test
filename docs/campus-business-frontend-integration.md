# 校园扩展业务前端对接说明

本文对应 `notification`、`moderation`、`academic`、`reservation`、`marketplace` 五个模块。用户端接口见 `docs/user-openapi.yaml`，管理端接口见 `docs/admin-openapi.yaml`。

## 公共约定

- 用户端基址为 `http://localhost:8080`，管理端基址为 `http://localhost:8081`。
- JSON 请求使用 Bearer Token；WebSocket 在连接后的第一帧发送 Token。
- 新业务 ID 都是十进制字符串。前端不得转成 JavaScript `number`，应始终按 `string` 保存和传递。
- 金额字段以 `Cents` 结尾，单位为分；费率字段以 `Bps` 结尾，`500` 表示 5%。
- 时间是带 `+08:00` 的 RFC 3339，例如 `2026-07-28T14:00:00+08:00`。
- JSON 接口的真实 HTTP 状态通常是 200 或 400。业务成功读取 `body.success`，业务语义读取 `body.httpstatus`，不要只判断 HTTP status。
- 分页结构固定为 `{data, current, total, size}`。空列表返回 `[]`，不会返回 `null`。

## Notification WebSocket

连接地址为 `/notification/ws`，连接建立后必须先发送：

```json
{"type":"auth","token":"<access-token>"}
```

鉴权成功后服务端返回 `auth_success`。通知事件格式为：

```json
{
  "type": "notification",
  "data": {
    "id": "1888888888888888888",
    "category": "marketplace",
    "eventType": "marketplace.payment.succeeded",
    "title": "支付成功",
    "content": "九成新机械键盘",
    "resourceType": "marketplaceOrder",
    "resourceId": "1888888888888888999",
    "createdTime": "2026-07-28T14:00:00+08:00",
    "isRead": false
  }
}
```

同一主账号可以建立多条连接。断线重连后应重新发送鉴权首帧，并通过 `GET /api/notification` 拉取断线期间的通知。分类固定为 `social`、`academic`、`marketplace`、`reservation`、`moderation`、`system`。

旧 `/api/notify/**` 和 chat WebSocket 通知仍保留原格式，旧页面无需同时订阅两个通道。新页面只使用 `/notification/ws`。

## 文件上传

课程资料直接请求 `POST /api/academic/courses/{id}/materials`，使用 `multipart/form-data`：

- `file`：原文件，最大 50MB。
- `semester`：学期。
- `title`：资料标题。
- `description`：可选说明。

允许 PDF、Word、PowerPoint、Excel、纯文本和图片。OOXML 文件虽然本质为 ZIP，但会按内部签名识别；普通压缩包和可执行文件会被拒绝。下载接口返回 302，前端应允许跟随重定向。

商品图片沿用 `/file/upload`。每张图上传成功后读取响应中的 `data.path`（32 位 MD5），将 1 至 9 个 MD5 放入商品请求的 `images` 数组。商品模块不接收 base64 或外部 URL。

## 状态机

课程状态：`normal -> hidden`，或 `normal -> merged`。评价和资料被删除或隐藏后不出现在公开列表。课程合并后，前端应刷新目标课程详情和评分聚合，不要继续缓存来源课程。

预约状态：

```text
reserved -> checkedIn
reserved -> cancelled
reserved -> closureCancelled
reserved -> noShow
```

取消截止时间为场馆配置的“开始前分钟数”；恰好到截止时刻已经不能取消。核销码只能使用一次。

商品状态：

```text
draft -> published -> reserved -> sold
draft/published/hidden -> withdrawn
published/reserved -> hidden (管理处置)
```

订单状态：

```text
pendingPayment -> paid -> delivered -> completed
pendingPayment -> cancelled
paid/delivered/disputed -> refunded
paid/delivered -> disputed -> paid/delivered/refunded
```

`pendingPayment` 保留 15 分钟；`delivered` 后 48 小时没有纠纷会自动完成。`disputed` 或存在待审批退款时会冻结自动完成和结算。

治理能力限制：

- `content`：阻止新增、修改和发送聊天消息；删除仍允许。
- `trade`：阻止发布/修改商品和创建新订单；已有订单的取消、交付、确认、退款和纠纷仍允许。
- `reservation`：阻止创建新预约；已有预约仍可取消。
- `account`：只保留登录、登出、自身处罚和申诉访问。

## 测试支付网关

测试网关仅在 `APP_PROFILE=dev` 或 `APP_PROFILE=test` 时启用。生产及其他 profile 会在发起支付前返回业务失败，不会写支付记录。

1. `POST /api/marketplace/orders` 创建订单。
2. `POST /api/marketplace/orders/{id}/pay`，传入客户端生成且重试时保持不变的 `requestId`。
3. 响应包含 `requestId`、`gatewayTransactionId` 和 `amountCents`。
4. 调用 `POST /api/marketplace/payments/test/callback`，原样传回上述三个字段。
5. 重复提交相同回调是安全的，只会完成一次支付状态迁移。

示例：

```json
{
  "requestId": "checkout-20260728-0001",
  "gatewayTransactionId": "test-pay-checkout-20260728-0001",
  "amountCents": 19900
}
```

退款由用户提交后在管理端审批。测试网关会生成稳定的退款和结算流水号；同一请求号重试不会产生第二笔业务记录。

## 前端刷新策略

- 创建订单、支付回调、取消、交付、收货、退款或纠纷后，重新拉取订单列表和商品详情。
- 创建或取消预约后，重新拉取当日时段与我的预约，不能只在本地加减余量。
- 标记通知已读后，可先乐观更新本地未读数；请求失败时以 `/unread-counts` 结果回滚。
- 管理端完成举报、退款、纠纷、闭馆或课程合并后，应刷新当前列表和目标详情，因为操作可能同时改变关联记录。
