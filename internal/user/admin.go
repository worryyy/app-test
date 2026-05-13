package user

import (
	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/adminjwt"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/responses"
)

const preAuthSuccessMessage = "预认证成功"

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

func (h *AdminHandler) RefreshToken(c *gin.Context) {
	var req RefreshTokenReq
	if !bindJSON(c, &req) {
		return
	}

	token, refreshToken, err := h.svc.AdminRefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		responses.Fail(c, err)
		return
	}

	responses.Success.RespData(c, &AdminRefreshResp{
		Token:        token,
		RefreshToken: refreshToken,
	})
}

func (h *AdminHandler) Logout(c *gin.Context) {
	if err := h.svc.AdminLogout(c.Request.Context(), currentAdminClaims(c)); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.Resp(c)
}

func (h *AdminHandler) UserToken(c *gin.Context) {
	token, refreshToken, err := h.svc.AdminUserToken(c.Request.Context(), currentAdminClaims(c))
	if err != nil {
		responses.Fail(c, err)
		return
	}

	responses.Success.RespData(c, &AdminUserTokenResp{
		Token:        token,
		RefreshToken: refreshToken,
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

	if err := h.svc.EditAdminUser(c.Request.Context(), uri.ID, currentAdminUserID(c), req); err != nil {
		responses.Fail(c, err)
		return
	}

	responses.Success.RespMessage(c, "更新成功")
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	var query adminListUsersQuery
	if !bindQuery(c, &query) {
		return
	}

	data, err := h.svc.ListUsers(c.Request.Context(), query.Page, query.Size, query.Filter())
	if err != nil {
		responses.Fail(c, err)
		return
	}

	responses.Success.RespData(c, data)
}

func (h *AdminHandler) PreAuth(c *gin.Context) {
	handleSinglePreAuth(c, h.svc)
}

func (h *AdminHandler) PreAuthBatch(c *gin.Context) {
	var req AdminBatchPreAuthReq
	if !bindJSON(c, &req) {
		return
	}

	data, err := h.svc.PreAuthenticationBatch(c.Request.Context(), req)
	if err != nil {
		responses.Fail(c, err)
		return
	}

	responses.Success.RespData(c, data)
}

func handleSinglePreAuth(c *gin.Context, svc *Service) {
	var query preAuthQuery
	if !bindQuery(c, &query) {
		return
	}

	if err := svc.PreAuthentication(c.Request.Context(), query.UserID, query.NickName, query.Pwd); err != nil {
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

func currentAdminClaims(c *gin.Context) *adminjwt.Claims {
	v, ok := c.Get("admin_claims")
	if !ok || v == nil {
		return nil
	}
	claims, ok := v.(*adminjwt.Claims)
	if !ok {
		return nil
	}
	return claims
}

func currentAdminUserID(c *gin.Context) int64 {
	claims := currentAdminClaims(c)
	if claims == nil {
		return 0
	}
	return claims.UserID
}
