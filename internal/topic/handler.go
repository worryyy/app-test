package topic

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/middleware"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/responses"

)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateTopicReq
	if !result.BindJSON(c, &req) {
		return
	}
	data, err := h.svc.Create(c.Request.Context(), middleware.GetClaims(c), &req)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Data(c, data)
}

func (h *Handler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("id"), middleware.GetUserID(c), false); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *Handler) GetByID(c *gin.Context) {
	queryUserID := userIDString(middleware.GetUserID(c))
	data, err := h.svc.GetByID(c.Request.Context(), c.Param("topic_id"), queryUserID)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Data(c, data)
}

func (h *Handler) Update(c *gin.Context) {
	var req UpdateTopicReq
	if !result.BindJSON(c, &req) {
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
	themeIDs := splitThemeIDs(firstNonEmpty(c.Query("themeIds"), c.Query("themeId")))
	content := firstNonEmpty(c.Query("content"), c.Query("keyword"))
	rawOrdCreated := strings.TrimSpace(c.Query("ord_created"))
	if rawOrdCreated == "" {
		rawOrdCreated = strings.TrimSpace(c.Query("orderBy"))
	}
	if rawOrdCreated == "" {
		rawOrdCreated = "0"
	}
	ordCreated, _ := strconv.Atoi(rawOrdCreated)
	if strings.EqualFold(c.Query("orderBy"), "created") {
		ordCreated = 1
	}
	if strings.EqualFold(c.Query("orderBy"), "hot") {
		ordCreated = 0
	}

	data, err := h.svc.Search(c.Request.Context(), userIDString(middleware.GetUserID(c)), themeIDs, content, page, size, ordCreated)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Data(c, data)
}

func (h *Handler) Mine(c *gin.Context) {
	page, size := pageSize(c)
	data, err := h.svc.ListMine(c.Request.Context(), middleware.GetUserID(c), page, size)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Data(c, data)
}

func (h *Handler) ThemeMine(c *gin.Context) {
	page, size := pageSize(c)
	data, err := h.svc.ListByTheme(c.Request.Context(), middleware.GetUserID(c), firstNonEmpty(c.Query("theme_id"), c.Query("themeId")), page, size)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	writeTopicListResult(c, data, true)
}

func (h *Handler) TargetUserTopics(c *gin.Context) {
	page, size := pageSize(c)
	targetUserID, err := strconv.ParseInt(firstNonEmpty(c.Query("target_user_id"), c.Query("targetUserId")), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, result.ErrParam.Error())
		return
	}
	data, err := h.svc.ListTargetUserTopics(c.Request.Context(), middleware.GetUserID(c), targetUserID, page, size)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	writeTopicListResult(c, data, true)
}

func (h *Handler) FollowTopics(c *gin.Context) {
	page, size := pageSize(c)
	data, err := h.svc.ListFollowTopics(c.Request.Context(), middleware.GetUserID(c), page, size)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Data(c, data)
}

func (h *Handler) Like(c *gin.Context) {
	if err := h.svc.LikeTopic(c.Request.Context(), middleware.GetClaims(c), c.Param("topic_id")); err != nil {
		if errors.Is(err, ErrTopicAlreadyLiked) {
			result.SuccessMsg(c, "已经点赞过了", nil)
			return
		}
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
	writeTopicListResult(c, data, true)
}

func (h *Handler) Collect(c *gin.Context) {
	if err := h.svc.CollectTopic(c.Request.Context(), middleware.GetClaims(c), c.Param("topic_id")); err != nil {
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
	writeTopicListResult(c, data, true)
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

func writeTopicListResult(c *gin.Context, data *result.CusPage[Topic], emptyAsList bool) {
	if emptyAsList && (data == nil || len(data.Data) == 0) {
		result.Data(c, []Topic{})
		return
	}
	result.Data(c, data)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func splitThemeIDs(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
