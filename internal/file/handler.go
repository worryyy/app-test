package file

import (
	"net/http"
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

func (h *Handler) Download(c *gin.Context) {
	url, err := h.svc.GetDownloadURL(c.Request.Context(), c.Param("md5"))
	if err != nil {
		result.HandleError(c, err)
		return
	}
	c.Redirect(http.StatusFound, url)
}

func (h *Handler) ListPublic(c *gin.Context) {
	page, size := pageSize(c)
	data, err := h.svc.ListPublic(c.Request.Context(), page, size)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *Handler) Upload(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	userID := strconv.FormatInt(middleware.GetUserID(c), 10)
	md5Value, url, err := h.svc.Upload(c.Request.Context(), file, header, userID)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, gin.H{"md5": md5Value, "url": url})
}

func (h *Handler) Delete(c *gin.Context) {
	userID := strconv.FormatInt(middleware.GetUserID(c), 10)
	if err := h.svc.Delete(c.Request.Context(), c.Param("md5"), userID, false); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
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
