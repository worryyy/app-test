package school

import (
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/bizerr"

	"github.com/gin-gonic/gin"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/responses"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) CurTerm(c *gin.Context) {
	data, err := h.svc.CurTerm(c.Request.Context())
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespData(c, data)
}

func (h *Handler) Authenticate(c *gin.Context) {
	var req AuthenticationReq
	if !bindJSON(c, &req) {
		return
	}

	currentUser, ok := h.currentUser(c)
	if !ok {
		return
	}
	if currentUser.StuIsCheck {
		responses.Fail(c, bizerr.Biz("当前用户已认证, 如需更换请重新认证"))
		return
	}

	loginResp, err := h.svc.Authenticate(c.Request.Context(), currentUser.ID, req)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespMessageData(c, "认证成功", loginResp)
}

func (h *Handler) ReAuthenticate(c *gin.Context) {
	var req AuthenticationReq
	if !bindJSON(c, &req) {
		return
	}

	currentUser, ok := h.currentUser(c)
	if !ok {
		return
	}
	if !currentUser.StuIsCheck {
		responses.Fail(c, bizerr.Biz("当前用户未认证，请先认证"))
		return
	}

	loginResp, err := h.svc.ReAuthentication(c.Request.Context(), currentUser.ID, req)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	responses.Success.RespMessageData(c, "认证成功", loginResp)
}

func (h *Handler) GetCourseByWeeks(c *gin.Context) {
	var req UserCourseReq
	if !bindJSON(c, &req) {
		return
	}
	if _, ok := h.requireCertifiedUser(c); !ok {
		return
	}

	resp, err := h.svc.GetCourseByWeeks(c.Request.Context(), req)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	writeJWCommonResponse(c, resp)
}

func (h *Handler) GetExam(c *gin.Context) {
	var req ExamReq
	if !bindJSON(c, &req) {
		return
	}
	if _, ok := h.requireCertifiedUser(c); !ok {
		return
	}

	resp, err := h.svc.GetExam(c.Request.Context(), req)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	writeJWCommonResponse(c, resp)
}

func (h *Handler) GetExamScore(c *gin.Context) {
	var req ExamScoreReq
	if !bindJSON(c, &req) {
		return
	}
	if _, ok := h.requireCertifiedUser(c); !ok {
		return
	}

	resp, err := h.svc.GetExamScore(c.Request.Context(), req)
	if err != nil {
		responses.Fail(c, err)
		return
	}
	writeJWCommonResponse(c, resp)
}
