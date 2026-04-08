package user

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/responses"
)

const (
	errMsgInvalidParam   = "invalid param"
	errMsgUserNotLogin   = "user not logged in"
	errMsgNeedCampusAuth = "campus auth required"
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

func (h *Handler) requireCertifiedUser(c *gin.Context) (*User, bool) {
	user, ok := h.currentUser(c)
	if !ok {
		return nil, false
	}
	if !user.StuIsCheck {
		responses.Fail(c, bizerr.Forbidden(errMsgNeedCampusAuth))
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

func isBizErrCode(err error, code int) bool {
	var be *bizerr.Error
	return errors.As(err, &be) && be.Code == code
}
