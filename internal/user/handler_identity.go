package user

import (
	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/responses"
)

func (h *Handler) CreateAnonymous(c *gin.Context) {
	rootUserID, ok := requireCurrentRootUserID(c)
	if !ok {
		return
	}

	data, err := h.svc.CreateAnonymousIdentity(c.Request.Context(), rootUserID)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, data)
}

func (h *Handler) UpdateAnonymousNickname(c *gin.Context) {
	var req UpdateAnonymousNicknameReq
	if !bindJSON(c, &req) {
		return
	}

	rootUserID, ok := requireCurrentRootUserID(c)
	if !ok {
		return
	}

	if err := h.svc.UpdateAnonymousNickname(c.Request.Context(), rootUserID, req.Nickname); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespMessage(c, "nickname updated")
}

func (h *Handler) ListIdentity(c *gin.Context) {
	rootUserID, ok := requireCurrentRootUserID(c)
	if !ok {
		return
	}

	data, err := h.svc.ListIdentities(c.Request.Context(), rootUserID)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, data)
}

func (h *Handler) SwitchIdentity(c *gin.Context) {
	var req IdentitySwitchReq
	if !bindJSON(c, &req) {
		return
	}

	rootUserID, ok := requireCurrentRootUserID(c)
	if !ok {
		return
	}

	token, refreshToken, target, rootUserID, err := h.svc.SwitchIdentityByAccountType(
		c.Request.Context(),
		rootUserID,
		req.AccountType,
	)
	if err != nil {
		responses.Fail(c, err)
		return
	}

	responses.Success.RespData(c, &SwitchIdentityResp{
		Token:           token,
		RefreshToken:    refreshToken,
		CurrentIdentity: buildIdentity(target),
		RootUserID:      rootUserID,
	})
}
