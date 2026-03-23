package topic

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/middleware"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateTopicReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	id, err := h.svc.Create(c.Request.Context(), middleware.GetClaims(c), &req)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, id)
}

func (h *Handler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("id"), middleware.GetUserID(c), false); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *Handler) GetByID(c *gin.Context) {
	queryUserID := strconv.FormatInt(middleware.GetUserID(c), 10)
	data, err := h.svc.GetByID(c.Request.Context(), c.Param("topic_id"), queryUserID)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *Handler) Update(c *gin.Context) {
	var req CreateTopicReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	if err := h.svc.Update(c.Request.Context(), c.Param("topic_id"), middleware.GetUserID(c), &req); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *Handler) Search(c *gin.Context) {
	page, size := pageSize(c)
	themeID := c.Query("themeId")
	keyword := c.Query("keyword")
	orderBy := c.Query("orderBy")
	data, err := h.svc.Search(c.Request.Context(), strconv.FormatInt(middleware.GetUserID(c), 10), themeID, keyword, page, size, orderBy)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *Handler) Mine(c *gin.Context) {
	page, size := pageSize(c)
	data, err := h.svc.ListMine(c.Request.Context(), middleware.GetUserID(c), page, size)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *Handler) ThemeMine(c *gin.Context) {
	page, size := pageSize(c)
	data, err := h.svc.ListByTheme(c.Request.Context(), middleware.GetUserID(c), c.Query("themeId"), page, size)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *Handler) TargetUserTopics(c *gin.Context) {
	page, size := pageSize(c)
	targetUserID, err := strconv.ParseInt(c.Query("targetUserId"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	data, err := h.svc.ListTargetUserTopics(c.Request.Context(), targetUserID, page, size)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *Handler) FollowTopics(c *gin.Context) {
	page, size := pageSize(c)
	data, err := h.svc.ListFollowTopics(c.Request.Context(), middleware.GetUserID(c), page, size)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *Handler) Like(c *gin.Context) {
	if err := h.svc.LikeTopic(c.Request.Context(), middleware.GetUserID(c), c.Param("topic_id")); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *Handler) Unlike(c *gin.Context) {
	if err := h.svc.UnlikeTopic(c.Request.Context(), middleware.GetUserID(c), c.Param("topic_id")); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *Handler) LikedTopics(c *gin.Context) {
	page, size := pageSize(c)
	data, err := h.svc.ListLikedTopics(c.Request.Context(), middleware.GetUserID(c), page, size)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *Handler) Collect(c *gin.Context) {
	if err := h.svc.CollectTopic(c.Request.Context(), middleware.GetUserID(c), c.Param("topic_id")); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *Handler) Uncollect(c *gin.Context) {
	if err := h.svc.UncollectTopic(c.Request.Context(), middleware.GetUserID(c), c.Param("topic_id")); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *Handler) CollectionTopics(c *gin.Context) {
	page, size := pageSize(c)
	data, err := h.svc.ListCollectedTopics(c.Request.Context(), middleware.GetUserID(c), page, size)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func pageSize(c *gin.Context) (int, int) {
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
