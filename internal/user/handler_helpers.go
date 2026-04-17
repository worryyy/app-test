package user

import (
	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/responses"
)

const (
	errMsgInvalidParam = "invalid param"
	errMsgUserNotLogin = "user not logged in"
)

func bindJSON(c *gin.Context, req any) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		responses.Fail(c, bizerr.Param(errMsgInvalidParam))
		return false
	}
	return true
}

func bindQuery(c *gin.Context, req any) bool {
	if err := c.ShouldBindQuery(req); err != nil {
		responses.Fail(c, bizerr.Param(errMsgInvalidParam))
		return false
	}
	return true
}

func bindURI(c *gin.Context, req any) bool {
	if err := c.ShouldBindUri(req); err != nil {
		responses.Fail(c, bizerr.Param(errMsgInvalidParam))
		return false
	}
	return true
}

func (h *Handler) currentUser(c *gin.Context) (*User, bool) {
	userID := currentUserID(c)
	if userID <= 0 {
		responses.Fail(c, bizerr.Unauthorized(errMsgUserNotLogin))
		return nil, false
	}

	user, err := h.svc.GetByID(c.Request.Context(), userID)
	if err != nil {
		responses.Fail(c, err)
		return nil, false
	}
	if user == nil {
		responses.Fail(c, ErrUserNotFound)
		return nil, false
	}
	return user, true
}

func requireCurrentRootUserID(c *gin.Context) (int64, bool) {
	claims := currentClaims(c)
	if claims == nil || claims.RootUserID <= 0 {
		responses.Fail(c, bizerr.Unauthorized(errMsgUserNotLogin))
		return 0, false
	}
	return claims.RootUserID, true
}
