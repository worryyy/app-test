package user

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/jwtutil"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/responses"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginReq
	if !bindJSON(c, &req) {
		return
	}

	token, refreshToken, user, activeIdentity, isNew, err := h.svc.WechatLogin(c.Request.Context(), req.Code)
	if err != nil {
		responses.Fail(c, err)
		return
	}

	responses.Success.RespData(c, &LoginResp{
		Token:           token,
		RefreshToken:    refreshToken,
		User:            user,
		IsNew:           isNew,
		CurrentIdentity: buildIdentity(activeIdentity),
		RootUserID:      rootUserID(user),
	})
}

func (h *Handler) RefreshToken(c *gin.Context) {
	var req RefreshTokenReq
	if !bindJSON(c, &req) {
		return
	}

	token, refreshToken, user, err := h.svc.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		responses.Fail(c, err)
		return
	}

	responses.Success.RespData(c, &RefreshTokenResp{
		Token:           token,
		RefreshToken:    refreshToken,
		CurrentIdentity: buildIdentity(user),
	})
}

func (h *Handler) GetCurrent(c *gin.Context) {
	user, ok := h.currentUser(c)
	if !ok {
		return
	}

	responses.Success.RespData(c, h.svc.sanitizeUser(user))
}

func (h *Handler) RandomNickname(c *gin.Context) {
	var query randomNicknameQuery
	if !bindQuery(c, &query) {
		return
	}

	name, err := h.svc.RandomNickname(query.Type)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, name)
}

func (h *Handler) Edit(c *gin.Context) {
	var req UserEditReq
	if !bindJSON(c, &req) {
		return
	}

	updatedUser, err := h.svc.Edit(c.Request.Context(), currentUserID(c), req)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, updatedUser)
}

func (h *Handler) GetUserProfile(c *gin.Context) {
	var query userProfileQuery
	if !bindQuery(c, &query) {
		return
	}

	profile, err := h.svc.GetUserProfile(c.Request.Context(), query.TargetUserID)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, profile)
}

func (h *Handler) UnlimitedWXACode(c *gin.Context) {
	var req struct {
		Scene string `json:"scene" binding:"required"`
		Page  string `json:"page"`
	}
	if !bindJSON(c, &req) {
		return
	}

	data, err := h.svc.GenerateUnlimitedWXACode(c.Request.Context(), req.Scene, req.Page)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	c.String(http.StatusOK, data)
}

func currentUserID(c *gin.Context) int64 {
	claims := currentClaims(c)
	if claims == nil {
		return 0
	}
	return claims.UserID
}

func currentClaims(c *gin.Context) *jwtutil.Claims {
	v, ok := c.Get("claims")
	if !ok || v == nil {
		return nil
	}

	claims, ok := v.(*jwtutil.Claims)
	if !ok {
		return nil
	}
	return claims
}
