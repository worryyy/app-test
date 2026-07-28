package academic

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/middleware"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/pagination"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/responses"
)

type Handler struct{ svc *Service }
type AdminHandler struct{ svc *Service }

func NewHandler(svc *Service) *Handler           { return &Handler{svc: svc} }
func NewAdminHandler(svc *Service) *AdminHandler { return &AdminHandler{svc: svc} }

func (h *Handler) Search(c *gin.Context) {
	page, size := pagination.PageSize(c)
	data, err := h.svc.SearchCourses(c.Request.Context(), rootID(c), c.Query("keyword"), page, size)
	write(c, data, err)
}
func (h *Handler) CreateCourse(c *gin.Context) {
	var req CreateCourseReq
	if !bind(c, &req) {
		return
	}
	data, err := h.svc.CreateCourse(c.Request.Context(), middleware.GetUserID(c), rootID(c), req)
	write(c, data, err)
}
func (h *Handler) CourseDetail(c *gin.Context) {
	data, err := h.svc.CourseDetail(c.Request.Context(), c.Param("id"))
	write(c, data, err)
}
func (h *Handler) Reviews(c *gin.Context) {
	page, size := pagination.PageSize(c)
	data, err := h.svc.ListReviews(c.Request.Context(), c.Param("id"), page, size)
	write(c, data, err)
}
func (h *Handler) SaveReview(c *gin.Context) {
	var req ReviewReq
	if !bind(c, &req) {
		return
	}
	data, err := h.svc.SaveReview(c.Request.Context(), middleware.GetUserID(c), rootID(c), c.Param("id"), req)
	write(c, data, err)
}
func (h *Handler) DeleteReview(c *gin.Context) {
	if err := h.svc.DeleteReview(c.Request.Context(), rootID(c), c.Param("id")); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.Resp(c)
}
func (h *Handler) UploadMaterial(c *gin.Context) {
	stream, header, err := c.Request.FormFile("file")
	if err != nil {
		responses.Fail(c, bizerr.Param("file参数错误"))
		return
	}
	data, err := h.svc.UploadMaterial(c.Request.Context(), middleware.GetUserID(c), rootID(c), c.Param("id"), c.PostForm("semester"), c.PostForm("title"), c.PostForm("description"), stream, header)
	write(c, data, err)
}
func (h *Handler) Materials(c *gin.Context) {
	page, size := pagination.PageSize(c)
	data, err := h.svc.ListMaterials(c.Request.Context(), c.Param("id"), page, size)
	write(c, data, err)
}
func (h *Handler) MyMaterials(c *gin.Context) {
	page, size := pagination.PageSize(c)
	data, err := h.svc.ListMyMaterials(c.Request.Context(), rootID(c), page, size)
	write(c, data, err)
}
func (h *Handler) DownloadMaterial(c *gin.Context) {
	url, err := h.svc.MaterialDownloadURL(c.Request.Context(), rootID(c), c.Param("id"))
	if err != nil {
		responses.Fail(c, err)
		return
	}
	c.Redirect(http.StatusFound, url)
}
func (h *Handler) DeleteMaterial(c *gin.Context) {
	if err := h.svc.DeleteMaterial(c.Request.Context(), rootID(c), c.Param("id")); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.Resp(c)
}

func (h *AdminHandler) Merge(c *gin.Context) {
	var req MergeCourseReq
	if !bind(c, &req) {
		return
	}
	if err := h.svc.MergeCourses(c.Request.Context(), c.Param("id"), req.TargetCourseID); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.Resp(c)
}
func (h *AdminHandler) HideCourse(c *gin.Context) {
	var req HideReq
	if !bind(c, &req) {
		return
	}
	if err := h.svc.HideCourse(c.Request.Context(), c.Param("id"), req.Hidden); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.Resp(c)
}
func (h *AdminHandler) HideReview(c *gin.Context) {
	var req HideReq
	if !bind(c, &req) {
		return
	}
	if err := h.svc.HideReview(c.Request.Context(), c.Param("id"), req.Hidden); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.Resp(c)
}
func (h *AdminHandler) HideMaterial(c *gin.Context) {
	var req HideReq
	if !bind(c, &req) {
		return
	}
	if err := h.svc.HideMaterial(c.Request.Context(), c.Param("id"), req.Hidden); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.Resp(c)
}

func rootID(c *gin.Context) int64 {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return 0
	}
	if claims.RootUserID > 0 {
		return claims.RootUserID
	}
	return claims.UserID
}
func bind(c *gin.Context, req any) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		responses.Fail(c, bizerr.Param("请求参数错误"))
		return false
	}
	return true
}
func write(c *gin.Context, data any, err error) {
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, data)
}
