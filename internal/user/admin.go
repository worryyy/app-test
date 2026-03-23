package user

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

type AdminHandler struct {
	svc *Service
}

func NewAdminHandler(svc *Service) *AdminHandler {
	return &AdminHandler{svc: svc}
}

func (h *AdminHandler) Login(c *gin.Context) {
	var req AdminLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	token, refreshToken, user, err := h.svc.AdminLogin(c.Request.Context(), &req)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, gin.H{"token": token, "refreshToken": refreshToken, "user": user})
}

func (h *AdminHandler) AddUser(c *gin.Context) {
	var req User
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	if err := h.svc.CreateUser(c.Request.Context(), &req); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *AdminHandler) AddAdmin(c *gin.Context) {
	var req AddAdminReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	if err := h.svc.AddAdmin(c.Request.Context(), req.UserID, req.Username, req.Password); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *AdminHandler) DeleteUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	if err := h.svc.DeleteUser(c.Request.Context(), id); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *AdminHandler) EditUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	var req User
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	if err := h.svc.Edit(c.Request.Context(), id, &req); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *AdminHandler) GetUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	u, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, u)
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "15"))
	name := c.Query("name")
	data, err := h.svc.ListUsers(c.Request.Context(), page, size, name)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *AdminHandler) ClearAuthentication(c *gin.Context) {
	var req UserIDReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	if err := h.svc.ClearAuthentication(c.Request.Context(), req.UserID); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *AdminHandler) UserCourse(c *gin.Context) {
	var req CourseFetchReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	if err := h.svc.RequestCourseByKey(c.Request.Context(), req.Key); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *AdminHandler) AddBlackList(c *gin.Context) {
	var req UserIDsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	if err := h.svc.AddBlackList(c.Request.Context(), req.UserIDs); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *AdminHandler) DelBlackList(c *gin.Context) {
	var req UserIDsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	if err := h.svc.DelBlackList(c.Request.Context(), req.UserIDs); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *AdminHandler) BlackList(c *gin.Context) {
	data, err := h.svc.ListBlackList(c.Request.Context())
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *AdminHandler) CertificationList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "15"))
	data, err := h.svc.ListOfficialCertifications(c.Request.Context(), page, size)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *AdminHandler) CertificationReview(c *gin.Context) {
	var req CertReviewReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	if err := h.svc.ReviewCertification(c.Request.Context(), req.CertID, req.Approved); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}
