package user

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/bizerr"
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
	if !bindJSON(c, &req) {
		return
	}

	token, refreshToken, user, err := h.svc.AdminLogin(c.Request.Context(), &req)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, &AdminLoginResp{
		Token:        token,
		RefreshToken: refreshToken,
		User:         user,
	})
}

func (h *AdminHandler) AddUser(c *gin.Context) {
	var req User
	if !bindJSON(c, &req) {
		return
	}
	if err := h.svc.CreateUser(c.Request.Context(), &req); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.Resp(c)
}

func (h *AdminHandler) AddAdmin(c *gin.Context) {
	var req AddAdminReq
	if !bindJSON(c, &req) {
		return
	}
	if err := h.svc.AddAdmin(c.Request.Context(), req.UserID, req.Username, req.Password, req.Power); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespMessage(c, "添加成功")
}

func (h *AdminHandler) DeleteUser(c *gin.Context) {
	id, ok := pathPositiveInt64(c, "id")
	if !ok {
		return
	}
	if err := h.svc.DeleteUser(c.Request.Context(), id); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespMessage(c, "删除成功")
}

func (h *AdminHandler) EditUser(c *gin.Context) {
	id, ok := pathPositiveInt64(c, "id")
	if !ok {
		return
	}

	var req AdminEditUserReq
	if !bindJSON(c, &req) {
		return
	}
	if err := h.svc.EditAdminUser(c.Request.Context(), id, currentUserID(c), req); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespMessage(c, "更新成功")
}

func (h *AdminHandler) GetUser(c *gin.Context) {
	id, ok := pathPositiveInt64(c, "id")
	if !ok {
		return
	}

	user, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	if user == nil {
		responses.Fail(c, ErrUserNotFound)
		return
	}
	responses.Success.RespData(c, h.svc.sanitizeUser(user))
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "0"))

	data, err := h.svc.ListUsers(c.Request.Context(), page, size, c.Query("nickName"))
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, data)
}

func (h *AdminHandler) ClearAuthentication(c *gin.Context) {
	var req UserIDReq
	if !bindJSON(c, &req) {
		return
	}
	if err := h.svc.ClearAuthentication(c.Request.Context(), req.UserID); err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.Resp(c)
}

func (h *AdminHandler) UserCourse(c *gin.Context) {
	key := strings.TrimSpace(c.Query("key"))
	if key == "" {
		responses.Fail(c, bizerr.Param(errMsgInvalidParam))
		return
	}

	course, err := h.svc.GetCourseFileByKey(c.Request.Context(), key)
	if err != nil {
		var be *bizerr.Error
		if errors.As(err, &be) && be.Code == bizerr.CodeNotFound {
			c.Data(http.StatusNotFound, "text/plain; charset=utf-8", []byte(be.Message))
			return
		}
		responses.Fail(c, err)
		return
	}

	c.Header("Content-Disposition", "attachment;filename="+key)
	c.Data(http.StatusOK, "application/msexcel", course.Val)
}
