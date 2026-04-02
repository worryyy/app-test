package user

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/responses"

)

type AdminHandler struct {
	svc *Service
}

func NewAdminHandler(svc *Service) *AdminHandler {
	return &AdminHandler{svc: svc}
}

func (h *AdminHandler) Login(c *gin.Context) {
	var req AdminLoginReq
	if !result.BindJSON(c, &req) {
		return
	}
	token, refreshToken, user, err := h.svc.AdminLogin(c.Request.Context(), &req)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, &AdminLoginResp{Token: token, RefreshToken: refreshToken, User: user})
}

func (h *AdminHandler) AddUser(c *gin.Context) {
	var req User
	if !result.BindJSON(c, &req) {
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
	if !result.BindJSON(c, &req) {
		return
	}
	if err := h.svc.AddAdmin(c.Request.Context(), req.UserID, req.Username, req.Password, req.Power); err != nil {
		result.HandleError(c, err)
		return
	}
	result.SuccessMsg(c, "添加成功", nil)
}

func (h *AdminHandler) DeleteUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, result.ErrParam.Error())
		return
	}
	if id < 1 {
		result.HandleError(c, result.ErrIDZero)
		return
	}
	if err := h.svc.DeleteUser(c.Request.Context(), id); err != nil {
		result.HandleError(c, err)
		return
	}
	result.SuccessMsg(c, "删除成功", nil)
}

func (h *AdminHandler) EditUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, result.ErrParam.Error())
		return
	}
	if id < 1 {
		result.HandleError(c, result.ErrIDZero)
		return
	}
	var req AdminEditUserReq
	if !result.BindJSON(c, &req) {
		return
	}
	if err := h.svc.EditAdminUser(c.Request.Context(), id, currentUserID(c), req); err != nil {
		result.HandleError(c, err)
		return
	}
	result.SuccessMsg(c, "更新成功", nil)
}

func (h *AdminHandler) GetUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, result.ErrParam.Error())
		return
	}
	if id < 1 {
		result.HandleError(c, result.ErrIDZero)
		return
	}
	user, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Data(c, h.svc.sanitizeUser(user))
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "0"))
	data, err := h.svc.ListUsers(c.Request.Context(), page, size, c.Query("nickName"))
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Data(c, data)
}

func (h *AdminHandler) ClearAuthentication(c *gin.Context) {
	var req UserIDReq
	if !result.BindJSON(c, &req) {
		return
	}
	if err := h.svc.ClearAuthentication(c.Request.Context(), req.UserID); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *AdminHandler) UserCourse(c *gin.Context) {
	key := strings.TrimSpace(c.Query("key"))
	if key == "" {
		result.Fail(c, result.CodeParamError, result.ErrParam.Error())
		return
	}
	course, err := h.svc.GetCourseFileByKey(c.Request.Context(), key)
	if err != nil {
		var bizErr *result.BizError
		if errors.As(err, &bizErr) && bizErr.Status == http.StatusNotFound {
			c.Data(http.StatusNotFound, "text/plain; charset=utf-8", []byte(bizErr.Msg))
			return
		}
		result.HandleError(c, err)
		return
	}
	c.Header("Content-Disposition", "attachment;filename="+key)
	c.Data(http.StatusOK, "application/msexcel", course.Val)
}

