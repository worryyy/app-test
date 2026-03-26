package user

import "github.com/gin-gonic/gin"

import "github.com/Milchstrassse/Ecampus-go/internal/pkg/result"

func (h *Handler) CreateAnonymous(c *gin.Context) {
	data, err := h.svc.CreateAnonymousIdentity(c.Request.Context(), currentRootUserID(c))
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *Handler) UpdateAnonymousNickname(c *gin.Context) {
	var req UpdateAnonymousNicknameReq
	if !result.BindJSON(c, &req) {
		return
	}
	if err := h.svc.UpdateAnonymousNickname(c.Request.Context(), currentRootUserID(c), req.Nickname); err != nil {
		result.HandleError(c, err)
		return
	}
	result.SuccessMsg(c, "昵称修改成功", nil)
}

func (h *Handler) ListIdentity(c *gin.Context) {
	data, err := h.svc.ListIdentities(c.Request.Context(), currentRootUserID(c))
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

	token, refreshToken, target, rootUserID, err := h.svc.SwitchIdentityByAccountType(
		c.Request.Context(),
		currentRootUserID(c),
		req.AccountType,
	)
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
