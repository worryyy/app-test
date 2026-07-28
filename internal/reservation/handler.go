package reservation

import (
	"github.com/Milchstrassse/Ecampus-go/internal/middleware"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/pagination"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/responses"
	"github.com/gin-gonic/gin"
)

type Handler struct{ svc *Service }
type AdminHandler struct{ svc *Service }

func NewHandler(s *Service) *Handler           { return &Handler{svc: s} }
func NewAdminHandler(s *Service) *AdminHandler { return &AdminHandler{svc: s} }
func (h *Handler) Venues(c *gin.Context) {
	v, e := h.svc.ListVenues(c.Request.Context())
	write(c, v, e)
}
func (h *Handler) Resources(c *gin.Context) {
	v, e := h.svc.ListResources(c.Request.Context(), c.Param("id"))
	write(c, v, e)
}
func (h *Handler) Slots(c *gin.Context) {
	v, e := h.svc.AvailableSlots(c.Request.Context(), c.Param("id"), c.Query("date"))
	write(c, v, e)
}
func (h *Handler) Create(c *gin.Context) {
	var r CreateBookingReq
	if !bind(c, &r) {
		return
	}
	v, e := h.svc.CreateBooking(c.Request.Context(), middleware.GetUserID(c), root(c), r)
	write(c, v, e)
}
func (h *Handler) Mine(c *gin.Context) {
	p, z := pagination.PageSize(c)
	v, e := h.svc.ListBookings(c.Request.Context(), root(c), p, z)
	write(c, v, e)
}
func (h *Handler) Cancel(c *gin.Context) {
	if e := h.svc.CancelBooking(c.Request.Context(), root(c), c.Param("id")); e != nil {
		responses.Fail(c, e)
		return
	}
	responses.Success.Resp(c)
}
func (h *AdminHandler) CreateVenue(c *gin.Context) {
	var r VenueReq
	if !bind(c, &r) {
		return
	}
	v, e := h.svc.CreateVenue(c.Request.Context(), r)
	write(c, v, e)
}
func (h *AdminHandler) CreateResource(c *gin.Context) {
	var r ResourceReq
	if !bind(c, &r) {
		return
	}
	v, e := h.svc.CreateResource(c.Request.Context(), c.Param("id"), r)
	write(c, v, e)
}
func (h *AdminHandler) CreateRule(c *gin.Context) {
	var r RuleReq
	if !bind(c, &r) {
		return
	}
	if e := h.svc.CreateRule(c.Request.Context(), c.Param("id"), r); e != nil {
		responses.Fail(c, e)
		return
	}
	responses.Success.Resp(c)
}
func (h *AdminHandler) CreateClosure(c *gin.Context) {
	var r ClosureReq
	if !bind(c, &r) {
		return
	}
	claims := middleware.GetAdminClaims(c)
	if claims == nil {
		responses.Fail(c, bizerr.Unauthorized("管理员未登录"))
		return
	}
	if e := h.svc.CreateClosure(c.Request.Context(), claims.AdminID, r); e != nil {
		responses.Fail(c, e)
		return
	}
	responses.Success.Resp(c)
}
func (h *AdminHandler) Checkin(c *gin.Context) {
	var r CheckinReq
	if !bind(c, &r) {
		return
	}
	if e := h.svc.Checkin(c.Request.Context(), r.Code); e != nil {
		responses.Fail(c, e)
		return
	}
	responses.Success.Resp(c)
}
func root(c *gin.Context) int64 {
	v := middleware.GetClaims(c)
	if v == nil {
		return 0
	}
	if v.RootUserID > 0 {
		return v.RootUserID
	}
	return v.UserID
}
func bind(c *gin.Context, v any) bool {
	if e := c.ShouldBindJSON(v); e != nil {
		responses.Fail(c, bizerr.Param("请求参数错误"))
		return false
	}
	return true
}
func write(c *gin.Context, v any, e error) {
	if e != nil {
		responses.Fail(c, e)
		return
	}
	responses.Success.RespData(c, v)
}
