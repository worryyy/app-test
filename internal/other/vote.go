package other

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/middleware"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

func (h *Handler) VoteList(c *gin.Context) {
	page, size := pageSize(c)
	data, err := h.svc.ListVotes(c.Request.Context(), page, size)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *Handler) VoteDraft(c *gin.Context) {
	infoID, err := strconv.ParseInt(c.Param("info_id"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	data, err := h.svc.GetVoteOptions(c.Request.Context(), infoID)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *Handler) VoteDraftAccept(c *gin.Context) {
	infoID, err := strconv.ParseInt(c.Param("info_id"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, "参数错误")
		return
	}
	var req VoteAcceptReq
	if !result.BindJSON(c, &req) {
		return
	}
	if err := h.svc.AcceptVoteOptions(c.Request.Context(), infoID, req.OptionIDs); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *Handler) VoteCreate(c *gin.Context) {
	var req VoteInfo
	if !result.BindJSON(c, &req) {
		return
	}
	req.UserID = middleware.GetUserID(c)
	if err := h.svc.CreateVoteInfo(c.Request.Context(), &req); err != nil {
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
	req.VoteInfoID = infoID
	if err := h.svc.AddVoteOption(c.Request.Context(), &req); err != nil {
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
	var req VoteReq
	if !result.BindJSON(c, &req) {
		return
	}
	if err := h.svc.Vote(c.Request.Context(), infoID, middleware.GetUserID(c), req.OptionIDs); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}
