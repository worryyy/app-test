package event

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

type AdminHandler struct {
	svc *Service
}

func NewAdminHandler(svc *Service) *AdminHandler {
	return &AdminHandler{svc: svc}
}

func (h *AdminHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, result.ErrParam.Error())
		return
	}
	if id < 1 {
		result.HandleError(c, result.ErrIDZero)
		return
	}
	if err := h.svc.DeleteEvent(c.Request.Context(), id); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *AdminHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, result.ErrParam.Error())
		return
	}
	if id < 1 {
		result.HandleError(c, result.ErrIDZero)
		return
	}
	var req EventUpdateReq
	if !result.BindJSON(c, &req) {
		return
	}
	ok, err := h.svc.UpdateEvent(c.Request.Context(), id, &req)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Data(c, ok)
}

func (h *AdminHandler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		result.Fail(c, result.CodeParamError, result.ErrParam.Error())
		return
	}
	if id < 1 {
		result.HandleError(c, result.ErrIDZero)
		return
	}
	data, err := h.svc.GetEvent(c.Request.Context(), id)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Data(c, data)
}

func (h *AdminHandler) List(c *gin.Context) {
	startTime, err := time.ParseInLocation(
		"2006-01-02 15:04:05",
		c.DefaultQuery("start_time", "2023-09-18 12:00:00"),
		time.Local,
	)
	if err != nil {
		result.Fail(c, result.CodeParamError, result.ErrParam.Error())
		return
	}

	prevID, _ := strconv.ParseInt(c.DefaultQuery("prev_id", "1"), 10, 64)
	size, _ := strconv.Atoi(c.DefaultQuery("size", "0"))
	data, err := h.svc.ListEvents(c.Request.Context(), EventListReq{
		PrevID:    prevID,
		Size:      size,
		StartTime: startTime,
		UserID:    c.Query("user_id"),
		EventType: c.Query("event_type"),
		KeyWord:   c.Query("key_word"),
	})
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Data(c, data)
}
