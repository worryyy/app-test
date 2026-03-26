package user

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

func (h *Handler) Follow(c *gin.Context) {
	targetUserID, err := strconv.ParseInt(c.Query("following_id"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, result.ErrParam.Error())
		return
	}
	if err := h.svc.Follow(c.Request.Context(), currentUserID(c), targetUserID); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *Handler) Unfollow(c *gin.Context) {
	targetUserID, err := strconv.ParseInt(c.Query("following_id"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, result.ErrParam.Error())
		return
	}
	if err := h.svc.Unfollow(c.Request.Context(), currentUserID(c), targetUserID); err != nil {
		result.HandleError(c, err)
		return
	}
	result.SuccessMsg(c, "取关成功", nil)
}

func (h *Handler) Followers(c *gin.Context) {
	targetUserID, err := strconv.ParseInt(c.Query("targetId"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, result.ErrParam.Error())
		return
	}
	page, size := getPageSize(c, h.svc.defaultPageSize())
	data, err := h.svc.ListFollowers(c.Request.Context(), targetUserID, page, size)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	if data != nil && len(data.Data) == 0 {
		result.Data(c, data)
		return
	}
	result.Success(c, data)
}

func (h *Handler) Followings(c *gin.Context) {
	targetUserID, err := strconv.ParseInt(c.Query("targetId"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, result.ErrParam.Error())
		return
	}
	page, size := getPageSize(c, h.svc.defaultPageSize())
	data, err := h.svc.ListFollowings(c.Request.Context(), targetUserID, page, size)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	if data != nil && len(data.Data) == 0 {
		result.Data(c, data)
		return
	}
	result.Success(c, data)
}

func (h *Handler) Stats(c *gin.Context) {
	targetUserID, err := strconv.ParseInt(c.Query("user_id"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, result.ErrParam.Error())
		return
	}
	data, err := h.svc.GetStats(c.Request.Context(), targetUserID)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *Handler) IsFollowing(c *gin.Context) {
	targetUserID, err := strconv.ParseInt(c.Query("target_user_id"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, result.ErrParam.Error())
		return
	}
	ok, err := h.svc.IsFollowing(c.Request.Context(), currentUserID(c), targetUserID)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, ok)
}

func getPageSize(c *gin.Context, defaultSize int) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "0"))
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = defaultSize
	}
	return page, size
}
