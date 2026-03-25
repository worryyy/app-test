package user

import "github.com/gin-gonic/gin"

import "github.com/Milchstrassse/Ecampus-go/internal/pkg/result"

func (h *Handler) CreateAnonymous(c *gin.Context) {
	var req IdentityAnonymousReq
	if !result.BindJSON(c, &req) {
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
	if !result.BindJSON(c, &req) {
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
	if !result.BindJSON(c, &req) {
		return
	}
	var (
		token        string
		refreshToken string
		target       *User
		rootUserID   int64
		err          error
	)
	if req.AccountType != "" {
		token, refreshToken, target, rootUserID, err = h.svc.SwitchIdentityByAccountType(c.Request.Context(), currentUserID(c), req.AccountType)
	} else if req.TargetUserID > 0 {
		token, refreshToken, target, rootUserID, err = h.svc.SwitchIdentity(c.Request.Context(), currentUserID(c), req.TargetUserID)
	} else {
		result.FailWithStatus(c, 400, result.CodeParamError, result.ErrParam.Error())
		return
	}
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, &SwitchIdentityResp{
		Token:           token,
		RefreshToken:    refreshToken,
		CurrentIdentity: buildIdentityVO(target),
		RootUserID:      rootUserID,
	})
}
