package user

import (
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/responses"
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

	followingID, err := parsePositiveInt64(query.ResolvedFollowingID())
	if err != nil {
		responses.Fail(c, err)
		return
	}
	if err := h.svc.Follow(c.Request.Context(), currentUserID(c), followingID); err != nil {
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

	followingID, err := parsePositiveInt64(query.ResolvedFollowingID())
	if err != nil {
		responses.Fail(c, err)
		return
	}
	if err := h.svc.Unfollow(c.Request.Context(), currentUserID(c), followingID); err != nil {
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

	userID, err := parsePositiveInt64(query.ResolvedUserID())
	if err != nil {
		responses.Fail(c, err)
		return
	}
	data, err := h.svc.GetUserStats(c.Request.Context(), userID)
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
