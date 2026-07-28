package moderation

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/middleware"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/pagination"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/responses"
)

type Handler struct{ svc *Service }
type AdminHandler struct{ svc *Service }

func NewHandler(svc *Service) *Handler           { return &Handler{svc: svc} }
func NewAdminHandler(svc *Service) *AdminHandler { return &AdminHandler{svc: svc} }

func (h *Handler) CreateReport(c *gin.Context) {
	var req CreateReportReq
	if !bindJSON(c, &req) {
		return
	}
	data, err := h.svc.CreateReport(c.Request.Context(), currentRootUserID(c), middleware.GetUserID(c), req)
	write(c, data, err)
}

func (h *Handler) Reports(c *gin.Context) {
	page, size := pagination.PageSize(c)
	data, err := h.svc.ListMyReports(c.Request.Context(), currentRootUserID(c), page, size)
	write(c, data, err)
}

func (h *Handler) WithdrawReport(c *gin.Context) {
	if err := h.svc.WithdrawReport(c.Request.Context(), currentRootUserID(c), c.Param("id")); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.Resp(c)
}

func (h *Handler) Punishments(c *gin.Context) {
	page, size := pagination.PageSize(c)
	data, err := h.svc.ListMyPunishments(c.Request.Context(), currentRootUserID(c), page, size)
	write(c, data, err)
}

func (h *Handler) CreateAppeal(c *gin.Context) {
	var req CreateAppealReq
	if !bindJSON(c, &req) {
		return
	}
	data, err := h.svc.CreateAppeal(c.Request.Context(), currentRootUserID(c), c.Param("id"), req)
	write(c, data, err)
}

func (h *Handler) Appeals(c *gin.Context) {
	page, size := pagination.PageSize(c)
	data, err := h.svc.ListMyAppeals(c.Request.Context(), currentRootUserID(c), page, size)
	write(c, data, err)
}

func (h *AdminHandler) Reports(c *gin.Context) {
	page, size := pagination.PageSize(c)
	data, err := h.svc.ListReportsAdmin(c.Request.Context(), c.Query("status"), page, size)
	write(c, data, err)
}

func (h *AdminHandler) Claim(c *gin.Context) {
	if err := h.svc.ClaimReport(c.Request.Context(), c.Param("id"), adminID(c)); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.Resp(c)
}

func (h *AdminHandler) Decide(c *gin.Context) {
	var req AdminDecisionReq
	if !bindJSON(c, &req) {
		return
	}
	if err := h.svc.DecideReport(c.Request.Context(), c.Param("id"), adminID(c), req); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.Resp(c)
}

func (h *AdminHandler) Revoke(c *gin.Context) {
	var req RevokePunishmentReq
	if !bindJSON(c, &req) {
		return
	}
	if err := h.svc.RevokePunishment(c.Request.Context(), c.Param("id"), adminID(c), req.Reason); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.Resp(c)
}

func (h *AdminHandler) Appeals(c *gin.Context) {
	page, size := pagination.PageSize(c)
	data, err := h.svc.ListAppealsAdmin(c.Request.Context(), page, size)
	write(c, data, err)
}

func (h *AdminHandler) DecideAppeal(c *gin.Context) {
	var req AppealDecisionReq
	if !bindJSON(c, &req) {
		return
	}
	if err := h.svc.DecideAppeal(c.Request.Context(), c.Param("id"), adminID(c), req); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.Resp(c)
}

func AccountGuard(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodOptions || accountGuardAllowed(c.FullPath()) {
			c.Next()
			return
		}
		blocked, err := svc.AccountBlocked(c.Request.Context(), currentRootUserID(c))
		if err != nil {
			responses.Fail(c, bizerr.InternalWrap("查询账号状态失败", err))
			c.Abort()
			return
		}
		if blocked {
			responses.Fail(c, ErrCapabilityDenied)
			c.Abort()
			return
		}
		c.Next()
	}
}

func accountGuardAllowed(path string) bool {
	return path == "/api/user/logout" || strings.HasPrefix(path, "/api/moderation/punishments") || strings.HasPrefix(path, "/api/moderation/appeals")
}

func currentRootUserID(c *gin.Context) int64 {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return 0
	}
	if claims.RootUserID > 0 {
		return claims.RootUserID
	}
	return claims.UserID
}

func adminID(c *gin.Context) int64 {
	claims := middleware.GetAdminClaims(c)
	if claims == nil {
		return 0
	}
	return claims.AdminID
}

func bindJSON(c *gin.Context, req any) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		responses.Fail(c, bizerr.Param("请求参数错误"))
		return false
	}
	return true
}

func write(c *gin.Context, data any, err error) {
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, data)
}
