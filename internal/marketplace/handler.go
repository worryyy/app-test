package marketplace

import (
	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/middleware"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/pagination"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/responses"
)

type Handler struct{ service *Service }
type AdminHandler struct{ service *Service }

func NewHandler(service *Service) *Handler           { return &Handler{service: service} }
func NewAdminHandler(service *Service) *AdminHandler { return &AdminHandler{service: service} }

func (h *Handler) Categories(c *gin.Context) {
	value, err := h.service.ListCategories(c.Request.Context(), false)
	writeMarketplace(c, value, err)
}
func (h *Handler) SearchItems(c *gin.Context) {
	page, size := pagination.PageSize(c)
	value, err := h.service.SearchItems(c.Request.Context(), marketplaceRoot(c), false, c.Query("keyword"), c.Query("categoryId"), page, size)
	writeMarketplace(c, value, err)
}
func (h *Handler) MyItems(c *gin.Context) {
	page, size := pagination.PageSize(c)
	value, err := h.service.SearchItems(c.Request.Context(), marketplaceRoot(c), true, c.Query("keyword"), c.Query("categoryId"), page, size)
	writeMarketplace(c, value, err)
}
func (h *Handler) ItemDetail(c *gin.Context) {
	value, err := h.service.ItemDetail(c.Request.Context(), marketplaceRoot(c), c.Param("id"))
	writeMarketplace(c, value, err)
}
func (h *Handler) CreateItem(c *gin.Context) {
	var req ItemReq
	if !bindMarketplace(c, &req) {
		return
	}
	value, err := h.service.CreateItem(c.Request.Context(), middleware.GetUserID(c), marketplaceRoot(c), req)
	writeMarketplace(c, value, err)
}
func (h *Handler) UpdateItem(c *gin.Context) {
	var req ItemReq
	if !bindMarketplace(c, &req) {
		return
	}
	value, err := h.service.UpdateItem(c.Request.Context(), middleware.GetUserID(c), marketplaceRoot(c), c.Param("id"), req)
	writeMarketplace(c, value, err)
}
func (h *Handler) WithdrawItem(c *gin.Context) {
	writeMarketplaceEmpty(c, h.service.WithdrawItem(c.Request.Context(), marketplaceRoot(c), c.Param("id")))
}
func (h *Handler) CreateOrder(c *gin.Context) {
	var req CreateOrderReq
	if !bindMarketplace(c, &req) {
		return
	}
	value, err := h.service.CreateOrder(c.Request.Context(), middleware.GetUserID(c), marketplaceRoot(c), req)
	writeMarketplace(c, value, err)
}
func (h *Handler) Orders(c *gin.Context) {
	page, size := pagination.PageSize(c)
	value, err := h.service.ListOrders(c.Request.Context(), marketplaceRoot(c), false, page, size)
	writeMarketplace(c, value, err)
}
func (h *Handler) Pay(c *gin.Context) {
	var req PaymentReq
	if !bindMarketplace(c, &req) {
		return
	}
	value, err := h.service.StartPayment(c.Request.Context(), marketplaceRoot(c), c.Param("id"), req)
	writeMarketplace(c, value, err)
}
func (h *Handler) TestPaymentCallback(c *gin.Context) {
	var req TestPaymentCallbackReq
	if !bindMarketplace(c, &req) {
		return
	}
	writeMarketplaceEmpty(c, h.service.TestPaymentCallback(c.Request.Context(), req))
}
func (h *Handler) CancelOrder(c *gin.Context) {
	writeMarketplaceEmpty(c, h.service.CancelOrder(c.Request.Context(), marketplaceRoot(c), c.Param("id")))
}
func (h *Handler) MarkDelivered(c *gin.Context) {
	writeMarketplaceEmpty(c, h.service.MarkDelivered(c.Request.Context(), marketplaceRoot(c), c.Param("id")))
}
func (h *Handler) ConfirmReceived(c *gin.Context) {
	writeMarketplaceEmpty(c, h.service.ConfirmReceived(c.Request.Context(), marketplaceRoot(c), c.Param("id")))
}
func (h *Handler) RequestRefund(c *gin.Context) {
	var req RefundReq
	if !bindMarketplace(c, &req) {
		return
	}
	value, err := h.service.RequestRefund(c.Request.Context(), marketplaceRoot(c), c.Param("id"), req)
	writeMarketplace(c, value, err)
}
func (h *Handler) CreateDispute(c *gin.Context) {
	var req DisputeReq
	if !bindMarketplace(c, &req) {
		return
	}
	value, err := h.service.CreateDispute(c.Request.Context(), marketplaceRoot(c), c.Param("id"), req)
	writeMarketplace(c, value, err)
}

func (h *AdminHandler) Categories(c *gin.Context) {
	value, err := h.service.ListCategories(c.Request.Context(), true)
	writeMarketplace(c, value, err)
}
func (h *AdminHandler) CreateCategory(c *gin.Context) {
	var req CategoryReq
	if !bindMarketplace(c, &req) {
		return
	}
	value, err := h.service.SaveCategory(c.Request.Context(), "", req)
	writeMarketplace(c, value, err)
}
func (h *AdminHandler) UpdateCategory(c *gin.Context) {
	var req CategoryReq
	if !bindMarketplace(c, &req) {
		return
	}
	value, err := h.service.SaveCategory(c.Request.Context(), c.Param("id"), req)
	writeMarketplace(c, value, err)
}
func (h *AdminHandler) Orders(c *gin.Context) {
	page, size := pagination.PageSize(c)
	value, err := h.service.ListOrders(c.Request.Context(), 0, true, page, size)
	writeMarketplace(c, value, err)
}
func (h *AdminHandler) Refunds(c *gin.Context) {
	page, size := pagination.PageSize(c)
	value, err := h.service.ListRefunds(c.Request.Context(), page, size)
	writeMarketplace(c, value, err)
}
func (h *AdminHandler) DecideRefund(c *gin.Context) {
	var req RefundDecisionReq
	if !bindMarketplace(c, &req) {
		return
	}
	writeMarketplaceEmpty(c, h.service.DecideRefund(c.Request.Context(), c.Param("id"), marketplaceAdminID(c), req))
}
func (h *AdminHandler) Disputes(c *gin.Context) {
	page, size := pagination.PageSize(c)
	value, err := h.service.ListDisputes(c.Request.Context(), page, size)
	writeMarketplace(c, value, err)
}
func (h *AdminHandler) ResolveDispute(c *gin.Context) {
	var req DisputeDecisionReq
	if !bindMarketplace(c, &req) {
		return
	}
	writeMarketplaceEmpty(c, h.service.ResolveDispute(c.Request.Context(), c.Param("id"), marketplaceAdminID(c), req))
}
func (h *AdminHandler) Settlements(c *gin.Context) {
	page, size := pagination.PageSize(c)
	value, err := h.service.ListSettlements(c.Request.Context(), page, size)
	writeMarketplace(c, value, err)
}
func (h *AdminHandler) RetrySettlement(c *gin.Context) {
	writeMarketplaceEmpty(c, h.service.RetrySettlement(c.Request.Context(), c.Param("id")))
}

func marketplaceRoot(c *gin.Context) int64 {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return 0
	}
	if claims.RootUserID > 0 {
		return claims.RootUserID
	}
	return claims.UserID
}
func marketplaceAdminID(c *gin.Context) int64 {
	claims := middleware.GetAdminClaims(c)
	if claims == nil {
		return 0
	}
	return claims.AdminID
}
func bindMarketplace(c *gin.Context, value any) bool {
	if err := c.ShouldBindJSON(value); err != nil {
		responses.Fail(c, bizerr.Param("请求参数错误"))
		return false
	}
	return true
}
func writeMarketplace(c *gin.Context, value any, err error) {
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, value)
}
func writeMarketplaceEmpty(c *gin.Context, err error) {
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.Resp(c)
}
