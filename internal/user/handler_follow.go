package user

import (
	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/responses"
	"github.com/gin-gonic/gin"
)

func (h *Handler) Follow(c *gin.Context) {
	if err := ensureLoggedInUser(c); err != nil {
		responses.Fail(c, err)
		return
	}

	var query followActionQuery
	if !bindQuery(c, &query) {
		return
	}

	if err := h.svc.Follow(c.Request.Context(), currentUserID(c), query.FollowingID); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.Resp(c)
}

func (h *Handler) Unfollow(c *gin.Context) {
	if err := ensureLoggedInUser(c); err != nil {
		responses.Fail(c, err)
		return
	}

	var query followActionQuery
	if !bindQuery(c, &query) {
		return
	}

	if err := h.svc.Unfollow(c.Request.Context(), currentUserID(c), query.FollowingID); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.Resp(c)
}

func (h *Handler) GetUserStats(c *gin.Context) {
	var query userStatsQuery
	if !bindQuery(c, &query) {
		return
	}

	data, err := h.svc.GetUserStats(c.Request.Context(), query.UserID)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, data)
}

func ensureLoggedInUser(c *gin.Context) error {
	if currentUserID(c) > 0 {
		return nil
	}
	return bizerr.Unauthorized(errMsgUserNotLogin)
}
