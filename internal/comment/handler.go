package comment

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/jwtutil"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateCommentReq
	if !result.BindJSON(c, &req) {
		return
	}
	_, err := h.svc.AddComment(c.Request.Context(), c.Param("topic_id"), getUserID(c), req.Comment, req.ParentCmtID)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *Handler) Delete(c *gin.Context) {
	if err := h.svc.DeleteComment(c.Request.Context(), c.Param("topic_id"), c.Param("comment_id"), getUserID(c), false); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *Handler) ListByTopic(c *gin.Context) {
	page, size := getPageSize(c)
	rootID := firstNonEmpty(c.Query("root_id"), c.Query("rootId"))
	data, err := h.svc.ListByTopic(c.Request.Context(), c.Param("topic_id"), rootID, getUserID(c), page, size)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Data(c, data)
}

func (h *Handler) Mine(c *gin.Context) {
	page, size := getPageSize(c)
	data, err := h.svc.ListMine(c.Request.Context(), getUserID(c), page, size)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Data(c, data)
}

func (h *Handler) TargetUserComments(c *gin.Context) {
	page, size := getPageSize(c)
	targetUserID := firstNonEmpty(c.Query("target_user_id"), c.Query("targetUserId"))
	data, err := h.svc.ListTargetUserComments(c.Request.Context(), targetUserID, page, size)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Data(c, data)
}

func (h *Handler) Like(c *gin.Context) {
	if err := h.svc.LikeComment(c.Request.Context(), c.Param("comment_id"), getUserID(c)); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *Handler) Unlike(c *gin.Context) {
	if err := h.svc.UnlikeComment(c.Request.Context(), c.Param("comment_id"), getUserID(c)); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func getClaims(c *gin.Context) *jwtutil.Claims {
	v, ok := c.Get("claims")
	if !ok || v == nil {
		return nil
	}
	claims, ok := v.(*jwtutil.Claims)
	if !ok {
		return nil
	}
	return claims
}

func getUserID(c *gin.Context) int64 {
	claims := getClaims(c)
	if claims == nil {
		return 0
	}
	return claims.UserID
}
