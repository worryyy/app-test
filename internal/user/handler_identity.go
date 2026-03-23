package user

import "github.com/gin-gonic/gin"

import "github.com/Milchstrassse/Ecampus-go/internal/pkg/result"

func (h *Handler) CreateAnonymous(c *gin.Context) {
	var req IdentityAnonymousReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	u, err := h.svc.CreateAnonymousIdentity(c.Request.Context(), currentUserID(c), req.Nickname)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, u)
}

func (h *Handler) UpdateAnonymousNickname(c *gin.Context) {
	var req IdentityAnonymousReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	if err := h.svc.UpdateAnonymousNickname(c.Request.Context(), currentUserID(c), req.Nickname); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *Handler) ListIdentity(c *gin.Context) {
	data, err := h.svc.ListIdentities(c.Request.Context(), currentUserID(c))
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *Handler) SwitchIdentity(c *gin.Context) {
	var req IdentitySwitchReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	token, refreshToken, err := h.svc.SwitchIdentity(c.Request.Context(), currentUserID(c), req.TargetUserID)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, gin.H{"token": token, "refreshToken": refreshToken})
}
