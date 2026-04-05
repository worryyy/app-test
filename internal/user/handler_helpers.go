package user

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/responses"
)

const (
	errMsgInvalidParam   = "参数错误"
	errMsgUserNotLogin   = "用户未登录"
	errMsgNeedCampusAuth = "请先进行校园认证"
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

func queryPositiveInt64(c *gin.Context, key string) (int64, bool) {
	value, err := parsePositiveInt64(c.Query(key))
	if err != nil {
		responses.Fail(c, err)
		return 0, false
	}
	return value, true
}

func pathPositiveInt64(c *gin.Context, key string) (int64, bool) {
	value, err := parsePositiveInt64(c.Param(key))
	if err != nil {
		responses.Fail(c, err)
		return 0, false
	}
	return value, true
}

func parsePositiveInt64(raw string) (int64, error) {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || value <= 0 {
		return 0, bizerr.Param(errMsgInvalidParam)
	}
	return value, nil
}

func (h *Handler) currentUser(c *gin.Context) (*User, bool) {
	userID := currentUserID(c)
	if userID <= 0 {
		responses.Fail(c, bizerr.Biz(errMsgUserNotLogin))
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
		responses.Fail(c, bizerr.Biz(errMsgNeedCampusAuth))
		return nil, false
	}
	return user, true
}

func isBizErrCode(err error, code int) bool {
	var be *bizerr.Error
	return errors.As(err, &be) && be.Code == code
}
