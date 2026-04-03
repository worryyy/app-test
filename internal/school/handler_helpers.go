package school

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/middleware"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/responses"
)

func bindJSON(c *gin.Context, req any) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		responses.Fail(c, bizerr.Param(errMsgInvalidParam))
		return false
	}
	return true
}

func writeJWCommonResponse(c *gin.Context, resp *JWCommonResp) {
	if resp == nil {
		responses.Fail(c, bizerr.Biz("获取失败"))
		return
	}
	responses.New(resp.Code, resp.Message, http.StatusOK).RespData(c, resp.Data)
}

func (h *Handler) currentUser(c *gin.Context) (*campusUser, bool) {
	userID := middleware.GetUserID(c)
	if userID <= 0 {
		responses.Fail(c, ErrUserNotLogin)
		return nil, false
	}

	user, err := h.svc.GetUserByID(c.Request.Context(), userID)
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

func (h *Handler) requireCertifiedUser(c *gin.Context) (*campusUser, bool) {
	user, ok := h.currentUser(c)
	if !ok {
		return nil, false
	}
	if !user.StuIsCheck {
		responses.Fail(c, ErrNeedCampusAuth)
		return nil, false
	}
	return user, true
}
