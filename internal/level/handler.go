package level

import (
	"strconv"
	"strings"

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

func (h *Handler) GetUserSignDetail(c *gin.Context) {
	data, err := h.svc.GetSignDetail(c.Request.Context(), middleware.GetUserID(c))
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *Handler) SignIn(c *gin.Context) {
	if err := h.svc.SignIn(c.Request.Context(), middleware.GetUserID(c)); err != nil {
		result.HandleError(c, err)
		return
	}
	result.SuccessMsg(c, "签到成功", nil)
}

func (h *Handler) TestAOP(c *gin.Context) {
	result.Data(c, h.svc.TestAOP(c.Request.Context()))
}

func (h *Handler) ExpPlus3(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, result.ErrParam.Error())
		return
	}
	h.svc.ExpPlus3(c.Request.Context(), id)
	result.SuccessMsg(c, "经验+3，告辞", nil)
}

func (h *Handler) UserExp(c *gin.Context) {
	rawIDs := c.QueryArray("userIdList")
	if len(rawIDs) == 0 {
		raw := strings.TrimSpace(c.Query("userIdList"))
		if raw != "" {
			rawIDs = strings.Split(raw, ",")
		}
	}
	if len(rawIDs) == 0 {
		result.Fail(c, result.CodeFail, "无用户id信息")
		return
	}
	ids := make([]int64, 0, len(rawIDs))
	for _, p := range rawIDs {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := strconv.ParseInt(p, 10, 64)
		if err == nil {
			ids = append(ids, id)
		}
	}
	data, err := h.svc.GetExpBatch(c.Request.Context(), ids)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	out := make([]UserExpItem, 0, len(data))
	for _, id := range ids {
		out = append(out, UserExpItem{UserID: id, UserExp: data[id]})
	}
	result.Success(c, out)
}
