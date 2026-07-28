package marketplace

import "github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"

var (
	ErrCategoryNotFound = bizerr.NotFound("商品类目不存在")
	ErrItemNotFound     = bizerr.NotFound("商品不存在")
	ErrItemUnavailable  = bizerr.Biz("商品当前不可购买")
	ErrOwnItem          = bizerr.Biz("不能购买自己的商品")
	ErrOrderNotFound    = bizerr.NotFound("订单不存在")
	ErrOrderState       = bizerr.Biz("订单当前状态不允许此操作")
	ErrPaymentDisabled  = bizerr.Biz("当前环境未启用支付网关")
	ErrPaymentInvalid   = bizerr.Biz("支付回调不合法")
	ErrRefundExists     = bizerr.Biz("该订单已提交退款")
	ErrDisputeExists    = bizerr.Biz("该订单已提交纠纷")
	ErrSettlementState  = bizerr.Biz("结算记录当前不可处理")
)
