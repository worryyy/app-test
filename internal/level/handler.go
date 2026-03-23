package level

import (
	"strconv"
	"strings"

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

func (h *Handler) GetUserSignDetail(c *gin.Context) {
	data, err := h.svc.GetSignDetail(c.Request.Context(), middleware.GetUserID(c))
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *Handler) SignIn(c *gin.Context) {
	if err := h.svc.SignIn(c.Request.Context(), middleware.GetUserID(c)); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *Handler) UserExp(c *gin.Context) {
	raw := c.Query("userIds")
	if strings.TrimSpace(raw) == "" {
		result.Success(c, []map[string]interface{}{})
		return
	}
	parts := strings.Split(raw, ",")
	ids := make([]int64, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := strconv.ParseInt(p, 10, 64)
		if err == nil {
			ids = append(ids, id)
		}
	}
	data, err := h.svc.GetExpBatch(c.Request.Context(), ids)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	out := make([]map[string]interface{}, 0, len(data))
	for _, id := range ids {
		out = append(out, map[string]interface{}{"userId": id, "exp": data[id]})
	}
	result.Success(c, out)
}
