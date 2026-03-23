package monitor

import "github.com/gin-gonic/gin"

import "github.com/Milchstrassse/Ecampus-go/internal/pkg/result"

type AdminHandler struct {
	svc *Service
}

func NewAdminHandler(svc *Service) *AdminHandler {
	return &AdminHandler{svc: svc}
}

func (h *AdminHandler) CacheNames(c *gin.Context) {
	data, err := h.svc.ListCacheNames(c.Request.Context())
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *AdminHandler) CacheStats(c *gin.Context) {
	data, err := h.svc.GetCacheStats(c.Request.Context(), c.Query("cacheName"))
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}
