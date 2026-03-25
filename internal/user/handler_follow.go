package user

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

func (h *Handler) Follow(c *gin.Context) {
	var req FollowReq
	if !result.BindJSON(c, &req) {
		return
	}
	if err := h.svc.Follow(c.Request.Context(), currentUserID(c), req.TargetUserID); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *Handler) Unfollow(c *gin.Context) {
	targetUserID, err := strconv.ParseInt(c.Query("targetUserId"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	if err := h.svc.Unfollow(c.Request.Context(), currentUserID(c), targetUserID); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *Handler) Followers(c *gin.Context) {
	page, size := getPageSize(c)
	data, err := h.svc.ListFollowers(c.Request.Context(), currentUserID(c), page, size)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *Handler) Followings(c *gin.Context) {
	page, size := getPageSize(c)
	data, err := h.svc.ListFollowings(c.Request.Context(), currentUserID(c), page, size)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *Handler) Stats(c *gin.Context) {
	targetUserID, err := strconv.ParseInt(c.Query("targetUserId"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	data, err := h.svc.GetStats(c.Request.Context(), currentUserID(c), targetUserID)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *Handler) IsFollowing(c *gin.Context) {
	targetUserID, err := strconv.ParseInt(c.Query("targetUserId"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	ok, err := h.svc.IsFollowing(c.Request.Context(), currentUserID(c), targetUserID)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, ok)
}

func getPageSize(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "15"))
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 15
	}
	return page, size
}
