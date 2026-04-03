package user

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/bizerr"
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

func (h *Handler) PreAuth(c *gin.Context) {
	userID, ok := queryPositiveInt64(c, "user_id")
	if !ok {
		return
	}

	if err := h.svc.PreAuthentication(c.Request.Context(), userID, c.Query("nick_name"), c.Query("pwd")); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespMessage(c, "预认证成功")
}

func (h *Handler) RandomNickname(c *gin.Context) {
	name, err := h.svc.RandomNickname(c.Query("type"))
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, name)
}

func (h *Handler) GetCurrent(c *gin.Context) {
	user, ok := h.currentUser(c)
	if !ok {
		return
	}
	responses.Success.RespData(c, h.svc.sanitizeUser(user))
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

func (h *Handler) Authenticate(c *gin.Context) {
	var req AuthenticationReq
	if !bindJSON(c, &req) {
		return
	}

	currentUser, ok := h.currentUser(c)
	if !ok {
		return
	}
	if currentUser.StuIsCheck {
		responses.Fail(c, bizerr.Biz("当前用户已认证, 如需更换请重新认证"))
		return
	}

	loginResp, err := h.svc.Authenticate(c.Request.Context(), currentUser.ID, req)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespMessageData(c, "认证成功", loginResp)
}

func (h *Handler) ReAuthenticate(c *gin.Context) {
	var req AuthenticationReq
	if !bindJSON(c, &req) {
		return
	}

	currentUser, ok := h.currentUser(c)
	if !ok {
		return
	}
	if !currentUser.StuIsCheck {
		responses.Fail(c, bizerr.Biz("当前用户未认证，请先认证"))
		return
	}

	loginResp, err := h.svc.ReAuthentication(c.Request.Context(), currentUser.ID, req)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespMessageData(c, "认证成功", loginResp)
}

func (h *Handler) DelAuthentication(c *gin.Context) {
	if err := h.svc.DelAuthentication(c.Request.Context(), currentUserID(c)); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.Resp(c)
}

func (h *Handler) CheckLogin(c *gin.Context) {
	var req CheckLoginReq
	if !bindJSON(c, &req) {
		return
	}
	if _, ok := h.requireCertifiedUser(c); !ok {
		return
	}

	loginResp, err := h.svc.CheckLogin(c.Request.Context(), req)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespMessageData(c, "认证成功", loginResp)
}

func (h *Handler) GetCourseByWeeks(c *gin.Context) {
	var req UserCourseReq
	if !bindJSON(c, &req) {
		return
	}
	if _, ok := h.requireCertifiedUser(c); !ok {
		return
	}

	resp, err := h.svc.GetCourseByWeeks(c.Request.Context(), req)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	writeJWCommonResponse(c, resp)
}

func (h *Handler) GetExam(c *gin.Context) {
	var req ExamReq
	if !bindJSON(c, &req) {
		return
	}
	if _, ok := h.requireCertifiedUser(c); !ok {
		return
	}

	resp, err := h.svc.GetExam(c.Request.Context(), req)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	writeJWCommonResponse(c, resp)
}

func (h *Handler) GetExamScore(c *gin.Context) {
	var req ExamScoreReq
	if !bindJSON(c, &req) {
		return
	}
	if _, ok := h.requireCertifiedUser(c); !ok {
		return
	}

	resp, err := h.svc.GetExamScore(c.Request.Context(), req)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	writeJWCommonResponse(c, resp)
}

func (h *Handler) GetUserProfile(c *gin.Context) {
	targetUserID, ok := queryPositiveInt64(c, "target_user_id")
	if !ok {
		return
	}

	profile, err := h.svc.GetUserProfile(c.Request.Context(), targetUserID)
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

func currentRootUserID(c *gin.Context) int64 {
	claims := currentClaims(c)
	if claims == nil {
		return 0
	}
	if claims.RootUserID > 0 {
		return claims.RootUserID
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
