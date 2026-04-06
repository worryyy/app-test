package user

import (
	"errors"
	"net/http"

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
		Token:              token,
		RefreshToken:       refreshToken,
		LegacyRefreshToken: refreshToken,
		User:               user,
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
	var uri userIDURI
	if !bindURI(c, &uri) {
		return
	}

	if err := h.svc.DeleteUser(c.Request.Context(), uri.ID); err != nil {
		responses.Fail(c, err)
		return
	}

	responses.Success.RespMessage(c, "删除成功")
}

func (h *AdminHandler) EditUser(c *gin.Context) {
	var uri userIDURI
	if !bindURI(c, &uri) {
		return
	}

	var req AdminEditUserReq
	if !bindJSON(c, &req) {
		return
	}

	if err := h.svc.EditAdminUser(c.Request.Context(), uri.ID, currentUserID(c), req); err != nil {
		responses.Fail(c, err)
		return
	}

	responses.Success.RespMessage(c, "更新成功")
}

func (h *AdminHandler) GetUser(c *gin.Context) {
	var uri userIDURI
	if !bindURI(c, &uri) {
		return
	}

	user, err := h.svc.GetByID(c.Request.Context(), uri.ID)
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
	var query adminListUsersQuery
	if !bindQuery(c, &query) {
		return
	}

	data, err := h.svc.ListUsers(c.Request.Context(), query.Page, query.Size, query.NickName)
	if err != nil {
		responses.Fail(c, err)
		return
	}

	responses.Success.RespData(c, data)
}

func (h *Handler) PreAuth(c *gin.Context) {
	var query preAuthQuery
	if !bindQuery(c, &query) {
		return
	}

	if err := h.svc.PreAuthentication(c.Request.Context(), query.UserID, query.NickName, query.Pwd); err != nil {
		responses.Fail(c, err)
		return
	}

	responses.Success.RespMessage(c, "预认证成功")
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
	var query courseKeyQuery
	if !bindQuery(c, &query) {
		return
	}

	course, err := h.svc.GetCourseFileByKey(c.Request.Context(), query.Key)
	if err != nil {
		var be *bizerr.Error
		if errors.As(err, &be) && be.Code == bizerr.CodeNotFound {
			c.Data(http.StatusNotFound, "text/plain; charset=utf-8", []byte(be.Message))
			return
		}
		responses.Fail(c, err)
		return
	}

	c.Header("Content-Disposition", "attachment;filename="+query.Key)
	c.Data(http.StatusOK, "application/msexcel", course.Val)
}
