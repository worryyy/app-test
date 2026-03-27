package user

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/jwtutil"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginReq
	if !result.BindJSON(c, &req) {
		return
	}

	token, refreshToken, user, activeIdentity, isNew, err := h.svc.WechatLogin(c.Request.Context(), req.Code)
	if err != nil {
		result.HandleError(c, err)
		return
	}

	result.Success(c, &LoginResp{
		Token:           token,
		RefreshToken:    refreshToken,
		User:            user,
		IsNew:           isNew,
		CurrentIdentity: buildIdentityVO(activeIdentity),
		RootUserID:      rootUserID(user),
	})
}

func (h *Handler) RefreshToken(c *gin.Context) {
	var req RefreshTokenReq
	if !result.BindJSON(c, &req) {
		return
	}
	token, refreshToken, user, err := h.svc.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		result.HandleError(c, err)
		return
	}

	result.Success(c, &RefreshTokenResp{
		Token:           token,
		RefreshToken:    refreshToken,
		CurrentIdentity: buildIdentityVO(user),
	})
}

func (h *Handler) PreAuth(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Query("user_id"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, result.ErrParam.Error())
		return
	}
	if err := h.svc.PreAuthentication(c.Request.Context(), userID, c.Query("nick_name"), c.Query("pwd")); err != nil {
		result.HandleError(c, err)
		return
	}
	result.SuccessMsg(c, "预认证成功", nil)
}

func (h *Handler) OfficialLogin(c *gin.Context) {
	var req OfficialLoginReq
	if !result.BindJSON(c, &req) {
		return
	}

	token, refreshToken, user, err := h.svc.OfficialLogin(c.Request.Context(), req.LoginAccount, req.LoginPassword)
	if err != nil {
		result.HandleError(c, err)
		return
	}

	result.SuccessMsg(c, "登录成功", &LoginResp{
		Token:        token,
		RefreshToken: refreshToken,
		User:         user,
		IsNew:        false,
	})
}

func (h *Handler) OfficialCert(c *gin.Context) {
	var req OfficialCertReq
	if !result.BindJSON(c, &req) {
		return
	}

	data, err := h.svc.SubmitOfficialCertification(c.Request.Context(), req)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.SuccessMsg(c, "认证申请提交成功，请等待审核", data)
}

func (h *Handler) RandomNickname(c *gin.Context) {
	name, err := h.svc.RandomNickname(c.Query("type"))
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Data(c, name)
}

func (h *Handler) GetCurrent(c *gin.Context) {
	userID := currentUserID(c)
	user, err := h.svc.GetByID(c.Request.Context(), userID)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Data(c, h.svc.sanitizeUser(user))
}

func (h *Handler) Edit(c *gin.Context) {
	var req UserEditReq
	if !result.BindJSON(c, &req) {
		return
	}

	updatedUser, err := h.svc.Edit(c.Request.Context(), currentUserID(c), req)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, updatedUser)
}

func (h *Handler) Authenticate(c *gin.Context) {
	var req AuthenticationReq
	if !result.BindJSON(c, &req) {
		return
	}

	currentUser, err := h.svc.GetByID(c.Request.Context(), currentUserID(c))
	if err != nil {
		result.HandleError(c, err)
		return
	}
	if currentUser != nil && currentUser.StuIsCheck {
		result.Fail(c, result.CodeFail, "当前用户已认证, 如需更换请重新认证")
		return
	}

	loginResp, err := h.svc.Authenticate(c.Request.Context(), currentUserID(c), req)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.SuccessMsg(c, "认证成功", loginResp)
}

func (h *Handler) ReAuthenticate(c *gin.Context) {
	var req AuthenticationReq
	if !result.BindJSON(c, &req) {
		return
	}

	currentUser, err := h.svc.GetByID(c.Request.Context(), currentUserID(c))
	if err != nil {
		result.HandleError(c, err)
		return
	}
	if currentUser == nil || !currentUser.StuIsCheck {
		result.Fail(c, result.CodeFail, "当前用户未认证，请先认证")
		return
	}

	loginResp, err := h.svc.ReAuthentication(c.Request.Context(), currentUserID(c), req)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.SuccessMsg(c, "认证成功", loginResp)
}

func (h *Handler) DelAuthentication(c *gin.Context) {
	if err := h.svc.DelAuthentication(c.Request.Context(), currentUserID(c)); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *Handler) CheckLogin(c *gin.Context) {
	var req CheckLoginReq
	if !result.BindJSON(c, &req) {
		return
	}

	currentUser, err := h.svc.GetByID(c.Request.Context(), currentUserID(c))
	if err != nil {
		result.HandleError(c, err)
		return
	}
	if currentUser != nil && !currentUser.StuIsCheck {
		result.Fail(c, result.CodeFail, "请先进行校园认证")
		return
	}

	loginResp, err := h.svc.CheckLogin(c.Request.Context(), req)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.SuccessMsg(c, "认证成功", loginResp)
}

func (h *Handler) GetCourseByWeeks(c *gin.Context) {
	var req UserCourseReq
	if !result.BindJSON(c, &req) {
		return
	}

	currentUser, err := h.svc.GetByID(c.Request.Context(), currentUserID(c))
	if err != nil {
		result.HandleError(c, err)
		return
	}
	if currentUser != nil && !currentUser.StuIsCheck {
		result.Fail(c, result.CodeFail, "请先进行校园认证")
		return
	}

	resp, err := h.svc.GetCourseByWeeks(c.Request.Context(), req)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	if resp == nil {
		result.Fail(c, result.CodeFail, "获取失败")
		return
	}
	result.Write(c, http.StatusOK, true, resp.Code, resp.Message, resp.Data)
}

func (h *Handler) GetExam(c *gin.Context) {
	var req ExamReq
	if !result.BindJSON(c, &req) {
		return
	}

	currentUser, err := h.svc.GetByID(c.Request.Context(), currentUserID(c))
	if err != nil {
		result.HandleError(c, err)
		return
	}
	if currentUser != nil && !currentUser.StuIsCheck {
		result.Fail(c, result.CodeFail, "请先进行校园认证")
		return
	}

	resp, err := h.svc.GetExam(c.Request.Context(), req)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	if resp == nil {
		result.Fail(c, result.CodeFail, "获取失败")
		return
	}
	result.Write(c, http.StatusOK, true, resp.Code, resp.Message, resp.Data)
}

func (h *Handler) GetExamScore(c *gin.Context) {
	var req ExamScoreReq
	if !result.BindJSON(c, &req) {
		return
	}

	currentUser, err := h.svc.GetByID(c.Request.Context(), currentUserID(c))
	if err != nil {
		result.HandleError(c, err)
		return
	}
	if currentUser != nil && !currentUser.StuIsCheck {
		result.Fail(c, result.CodeFail, "请先进行校园认证")
		return
	}

	resp, err := h.svc.GetExamScore(c.Request.Context(), req)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	if resp == nil {
		result.Fail(c, result.CodeFail, "获取失败")
		return
	}
	result.Write(c, http.StatusOK, true, resp.Code, resp.Message, resp.Data)
}

func (h *Handler) GetUserProfile(c *gin.Context) {
	targetUserID, err := strconv.ParseInt(c.Query("target_user_id"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, result.ErrParam.Error())
		return
	}

	profile, svcErr := h.svc.GetUserProfile(c.Request.Context(), targetUserID)
	if svcErr != nil {
		result.HandleError(c, svcErr)
		return
	}
	result.Success(c, profile)
}

func (h *Handler) UnlimitedWXACode(c *gin.Context) {
	var req struct {
		Scene string `json:"scene" binding:"required"`
		Page  string `json:"page"`
	}
	if !result.BindJSON(c, &req) {
		return
	}
	data, err := h.svc.GenerateUnlimitedWXACode(c.Request.Context(), req.Scene, req.Page)
	if err != nil {
		result.HandleError(c, err)
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
