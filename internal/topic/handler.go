package topic

import (
	"errors"

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
	if !bindJSON(c, &req) {
		return
	}

	data, err := h.svc.Create(c.Request.Context(), middleware.GetClaims(c), &req)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, data)
}

func (h *Handler) Delete(c *gin.Context) {
	var uri resourceIDURI
	if !bindURI(c, &uri) {
		return
	}

	if err := h.svc.Delete(c.Request.Context(), uri.ID, middleware.GetUserID(c), false); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.Resp(c)
}

func (h *Handler) GetByID(c *gin.Context) {
	var uri topicURI
	if !bindURI(c, &uri) {
		return
	}

	queryUserID := userIDString(middleware.GetUserID(c))
	data, err := h.svc.GetByID(c.Request.Context(), uri.TopicID, queryUserID)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, data)
}

func (h *Handler) Update(c *gin.Context) {
	var uri topicURI
	if !bindURI(c, &uri) {
		return
	}

	var req UpdateTopicReq
	if !bindJSON(c, &req) {
		return
	}

	if err := h.svc.Update(c.Request.Context(), uri.TopicID, middleware.GetUserID(c), &req); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.Resp(c)
}

func (h *Handler) Search(c *gin.Context) {
	var query topicSearchQuery
	if !bindQuery(c, &query) {
		return
	}

	page, size := pageSize(c)
	data, err := h.svc.Search(
		c.Request.Context(),
		userIDString(middleware.GetUserID(c)),
		splitThemeIDs(query.ThemeIDs),
		query.Content,
		page,
		size,
		query.OrdCreated,
	)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, data)
}

func (h *Handler) Mine(c *gin.Context) {
	page, size := pageSize(c)
	data, err := h.svc.ListMine(c.Request.Context(), middleware.GetUserID(c), page, size)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, data)
}

func (h *Handler) TargetUserTopics(c *gin.Context) {
	var query targetUserTopicsQuery
	if !bindQuery(c, &query) {
		return
	}

	page, size := pageSize(c)
	data, err := h.svc.ListTargetUserTopics(c.Request.Context(), middleware.GetUserID(c), query.TargetUserID, page, size)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	writeTopicListResult(c, data, true)
}

func (h *Handler) Like(c *gin.Context) {
	var uri topicURI
	if !bindURI(c, &uri) {
		return
	}

	if err := h.svc.LikeTopic(c.Request.Context(), middleware.GetClaims(c), uri.TopicID); err != nil {
		if errors.Is(err, ErrTopicAlreadyLiked) {
			responses.Success.RespMessage(c, ErrTopicAlreadyLiked.Error())
			return
		}
		responses.Fail(c, err)
		return
	}
	responses.Success.Resp(c)
}

func (h *Handler) Unlike(c *gin.Context) {
	var uri topicURI
	if !bindURI(c, &uri) {
		return
	}

	if err := h.svc.UnlikeTopic(c.Request.Context(), middleware.GetUserID(c), uri.TopicID); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.Resp(c)
}

func (h *Handler) LikedTopics(c *gin.Context) {
	page, size := pageSize(c)
	data, err := h.svc.ListLikedTopics(c.Request.Context(), middleware.GetUserID(c), page, size)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	writeTopicListResult(c, data, true)
}

func (h *Handler) Collect(c *gin.Context) {
	var uri topicURI
	if !bindURI(c, &uri) {
		return
	}

	if err := h.svc.CollectTopic(c.Request.Context(), middleware.GetClaims(c), uri.TopicID); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.Resp(c)
}

func (h *Handler) Uncollect(c *gin.Context) {
	var uri topicURI
	if !bindURI(c, &uri) {
		return
	}

	if err := h.svc.UncollectTopic(c.Request.Context(), middleware.GetUserID(c), uri.TopicID); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.Resp(c)
}

func (h *Handler) CollectionTopics(c *gin.Context) {
	page, size := pageSize(c)
	data, err := h.svc.ListCollectedTopics(c.Request.Context(), middleware.GetUserID(c), page, size)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	writeTopicListResult(c, data, true)
}
