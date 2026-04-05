package file

import (
	"net/http"
	"strconv"

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

func (h *Handler) Download(c *gin.Context) {
	var uri fileMD5URI
	if !bindURI(c, &uri) {
		return
	}
	var query fileDownloadQuery
	if !bindQuery(c, &query) {
		return
	}

	url, err := h.svc.GetDownloadURL(c.Request.Context(), uri.MD5, query.ShowOrigin > 0)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	c.Redirect(http.StatusFound, url)
}

func (h *Handler) ListPublic(c *gin.Context) {
	page, size := pageSize(c)
	data, err := h.svc.ListPublic(c.Request.Context(), page, size)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, data)
}

func (h *Handler) Upload(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		responses.ParamErr.RespMessage(c, "file参数错误")
		return
	}

	userID := strconv.FormatInt(middleware.GetUserID(c), 10)
	md5Value, err := h.svc.Upload(c.Request.Context(), file, header, userID)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, UploadResp{Path: md5Value})
}

func (h *Handler) Delete(c *gin.Context) {
	var uri fileMD5URI
	if !bindURI(c, &uri) {
		return
	}

	userID := strconv.FormatInt(middleware.GetUserID(c), 10)
	if err := h.svc.Delete(c.Request.Context(), uri.MD5, userID, false); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.Resp(c)
}
