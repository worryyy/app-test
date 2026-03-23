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
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	token, refreshToken, user, err := h.svc.WechatLogin(c.Request.Context(), req.Code)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, gin.H{"token": token, "refreshToken": refreshToken, "user": user})
}

func (h *Handler) RefreshToken(c *gin.Context) {
	var req RefreshTokenReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	token, refreshToken, err := h.svc.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, gin.H{"token": token, "refreshToken": refreshToken})
}

func (h *Handler) PreAuth(c *gin.Context) {
	var req AuthenticationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	if err := h.svc.PreAuthentication(c.Request.Context(), req.StuNum, req.StuPwd); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *Handler) OfficialLogin(c *gin.Context) {
	var req OfficialLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	if req.SecondaryPassword == "" {
		req.SecondaryPassword = "required"
	}
	token, refreshToken, user, err := h.svc.OfficialLogin(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, gin.H{"token": token, "refreshToken": refreshToken, "user": user})
}

func (h *Handler) OfficialCert(c *gin.Context) {
	var req OfficialCertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	userID := currentUserID(c)
	if err := h.svc.SubmitOfficialCertification(c.Request.Context(), userID, req.Name, req.Reason); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *Handler) RandomNickname(c *gin.Context) {
	name := h.svc.RandomNickname()
	result.Success(c, name)
}

func (h *Handler) GetCurrent(c *gin.Context) {
	userID := currentUserID(c)
	u, err := h.svc.GetByID(c.Request.Context(), userID)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, u)
}

func (h *Handler) Edit(c *gin.Context) {
	var req User
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	if err := h.svc.Edit(c.Request.Context(), currentUserID(c), &req); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *Handler) Authenticate(c *gin.Context) {
	var req AuthenticationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	if err := h.svc.Authenticate(c.Request.Context(), currentUserID(c), req.StuNum, req.StuPwd); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *Handler) ReAuthenticate(c *gin.Context) {
	var req AuthenticationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	if err := h.svc.ReAuthentication(c.Request.Context(), currentUserID(c), req.StuNum, req.StuPwd); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *Handler) DelAuthentication(c *gin.Context) {
	if err := h.svc.DelAuthentication(c.Request.Context(), currentUserID(c)); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *Handler) CheckLogin(c *gin.Context) {
	ok, err := h.svc.CheckLogin(c.Request.Context(), currentUserID(c))
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, ok)
}

func (h *Handler) GetCourseByWeeks(c *gin.Context) {
	var req UserCourseReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	data, err := h.svc.GetCourseByWeeks(c.Request.Context(), currentUserID(c), req.Term, req.Weeks)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *Handler) GetExam(c *gin.Context) {
	data, err := h.svc.GetExam(c.Request.Context(), currentUserID(c))
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *Handler) GetExamScore(c *gin.Context) {
	data, err := h.svc.GetExamScore(c.Request.Context(), currentUserID(c))
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *Handler) GetUserProfile(c *gin.Context) {
	targetUserID, err := strconv.ParseInt(c.Query("targetUserId"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	profile, svcErr := h.svc.GetStats(c.Request.Context(), currentUserID(c), targetUserID)
	if svcErr != nil {
		result.HandleError(c, svcErr)
		return
	}
	result.Success(c, profile)
}

func (h *Handler) UnlimitedWXACode(c *gin.Context) {
	var req struct {
		Scene string `json:"scene" binding:"required"`
		Page  string `json:"page" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	data, err := h.svc.GenerateUnlimitedWXACode(c.Request.Context(), req.Scene, req.Page)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	c.Data(http.StatusOK, "image/png", data)
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
