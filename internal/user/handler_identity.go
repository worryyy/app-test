package user

import (
	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/responses"
)

func (h *Handler) CreateAnonymous(c *gin.Context) {
	data, err := h.svc.CreateAnonymousIdentity(c.Request.Context(), currentRootUserID(c))
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

	if err := h.svc.UpdateAnonymousNickname(c.Request.Context(), currentRootUserID(c), req.Nickname); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespMessage(c, "昵称修改成功")
}

func (h *Handler) ListIdentity(c *gin.Context) {
	data, err := h.svc.ListIdentities(c.Request.Context(), currentRootUserID(c))
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

	accountType := req.ResolvedAccountType()
	if accountType == "" {
		responses.Fail(c, bizerr.Param(errMsgInvalidParam))
		return
	}

	token, refreshToken, target, rootUserID, err := h.svc.SwitchIdentityByAccountType(
		c.Request.Context(),
		currentRootUserID(c),
		accountType,
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
