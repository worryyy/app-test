package user

import (
	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/responses"
)

const preAuthSuccessMessage = "\u9884\u8ba4\u8bc1\u6210\u529f"

type AdminHandler struct {
	svc *Service
}

func NewAdminHandler(svc *Service) *AdminHandler {
	return &AdminHandler{svc: svc}
}

func (h *AdminHandler) Login(c *gin.Context) {
	var req AdminLoginReq
	if !bindJSON(c, &req) {
		return
	}

	token, refreshToken, user, err := h.svc.AdminLogin(c.Request.Context(), &req)
	if err != nil {
		responses.Fail(c, err)
		return
	}

	responses.Success.RespData(c, &AdminLoginResp{
		Token:        token,
		RefreshToken: refreshToken,
		User:         user,
	})
}

func (h *AdminHandler) AddUser(c *gin.Context) {
	var req User
	if !bindJSON(c, &req) {
		return
	}

	if err := h.svc.CreateUser(c.Request.Context(), &req); err != nil {
		responses.Fail(c, err)
		return
	}

	responses.Success.Resp(c)
}

func (h *AdminHandler) EditUser(c *gin.Context) {
	var uri userIDURI
	if !bindURI(c, &uri) {
		return
	}

	var req AdminEditUserReq
	if !bindJSON(c, &req) {
		return
	}

	if err := h.svc.EditAdminUser(c.Request.Context(), uri.ID, currentUserID(c), req); err != nil {
		responses.Fail(c, err)
		return
	}

	responses.Success.RespMessage(c, "\u66f4\u65b0\u6210\u529f")
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	var query adminListUsersQuery
	if !bindQuery(c, &query) {
		return
	}

	data, err := h.svc.ListUsers(c.Request.Context(), query.Page, query.Size, query.NickName)
	if err != nil {
		responses.Fail(c, err)
		return
	}

	responses.Success.RespData(c, data)
}

func (h *Handler) PreAuth(c *gin.Context) {
	var query preAuthQuery
	if !bindQuery(c, &query) {
		return
	}

	if err := h.svc.PreAuthentication(c.Request.Context(), query.UserID, query.NickName, query.Pwd); err != nil {
		responses.Fail(c, err)
		return
	}

	responses.Success.RespMessage(c, preAuthSuccessMessage)
}

func (h *AdminHandler) PreAuth(c *gin.Context) {
	var query preAuthQuery
	if !bindQuery(c, &query) {
		return
	}

	if err := h.svc.PreAuthentication(c.Request.Context(), query.UserID, query.NickName, query.Pwd); err != nil {
		responses.Fail(c, err)
		return
	}

	responses.Success.RespMessage(c, preAuthSuccessMessage)
}

func (h *AdminHandler) ClearAuthentication(c *gin.Context) {
	var req UserIDReq
	if !bindJSON(c, &req) {
		return
	}

	if err := h.svc.ClearAuthentication(c.Request.Context(), req.UserID); err != nil {
		responses.Fail(c, err)
		return
	}

	responses.Success.Resp(c)
}
