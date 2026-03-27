package vote

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/Milchstrassse/Ecampus-go/internal/middleware"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/pagination"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) VoteList(c *gin.Context) {
	page, size := pagination.PageSize(c)
	data, err := h.svc.ListVotes(c.Request.Context(), page, size)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Data(c, data)
}

func (h *Handler) VoteDraft(c *gin.Context) {
	infoID, err := strconv.ParseInt(c.Param("info_id"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	page, size := pagination.PageSize(c)
	isOK, _ := strconv.Atoi(c.DefaultQuery("is_ok", "1"))
	data, err := h.svc.GetVoteOptions(c.Request.Context(), infoID, middleware.GetUserID(c), page, size, isOK)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Data(c, data)
}

func (h *Handler) VoteDraftAccept(c *gin.Context) {
	infoID, err := strconv.ParseInt(c.Param("info_id"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	optionIDs, ok := bindVoteOptionIDs(c)
	if !ok {
		return
	}
	if err := h.svc.AcceptVoteOptions(c.Request.Context(), infoID, middleware.GetUserID(c), optionIDs); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *Handler) VoteCreate(c *gin.Context) {
	var req VoteCreateReq
	if !result.BindJSON(c, &req) {
		return
	}
	if err := h.svc.CreateVoteInfo(c.Request.Context(), middleware.GetUserID(c), &req); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *Handler) VoteAddOption(c *gin.Context) {
	infoID, err := strconv.ParseInt(c.Param("info_id"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	var req VoteOption
	if !result.BindJSON(c, &req) {
		return
	}
	if err := h.svc.AddVoteOption(c.Request.Context(), infoID, middleware.GetUserID(c), &req); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *Handler) VoteDo(c *gin.Context) {
	infoID, err := strconv.ParseInt(c.Param("info_id"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	optionIDs, ok := bindVoteOptionIDs(c)
	if !ok {
		return
	}
	if err := h.svc.Vote(c.Request.Context(), infoID, middleware.GetUserID(c), optionIDs); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func bindVoteOptionIDs(c *gin.Context) ([]int64, bool) {
	var rawIDs []int64
	if err := c.ShouldBindBodyWith(&rawIDs, binding.JSON); err == nil {
		return rawIDs, true
	}

	var req VoteReq
	if err := c.ShouldBindBodyWith(&req, binding.JSON); err == nil {
		return req.OptionIDs, true
	}

	result.FailWithStatus(c, http.StatusBadRequest, result.CodeParamError, result.ErrParam.Error())
	return nil, false
}
