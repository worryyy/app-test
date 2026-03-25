package other

import (
	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

func (h *AdminHandler) SensitiveGetAllList(c *gin.Context) {
	data, err := h.svc.GetAllSensitiveWords(c.Request.Context())
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *AdminHandler) SensitiveGetByWord(c *gin.Context) {
	data, err := h.svc.GetSensitiveWordByWord(c.Request.Context(), c.Query("word"))
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *AdminHandler) SensitiveDeleteByWord(c *gin.Context) {
	if err := h.svc.DeleteSensitiveWordByWord(c.Request.Context(), c.Query("word")); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *AdminHandler) SensitiveBatchDelete(c *gin.Context) {
	var req WordsReq
	if !result.BindJSON(c, &req) {
		return
	}
	if err := h.svc.BatchDeleteSensitiveWords(c.Request.Context(), req.Words); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *AdminHandler) SensitiveAdd(c *gin.Context) {
	var req SensitiveWord
	if !result.BindJSON(c, &req) {
		return
	}
	if err := h.svc.AddSensitiveWord(c.Request.Context(), &req); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *AdminHandler) SensitiveBatchAdd(c *gin.Context) {
	var req WordsReq
	if !result.BindJSON(c, &req) {
		return
	}
	if err := h.svc.BatchAddSensitiveWords(c.Request.Context(), req.Words); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *AdminHandler) SensitivePage(c *gin.Context) {
	page, size := pageSize(c)
	data, err := h.svc.SensitiveWordPage(c.Request.Context(), page, size)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *AdminHandler) SensitiveSearchLike(c *gin.Context) {
	data, err := h.svc.SearchSensitiveWordLike(c.Request.Context(), c.Query("word"))
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *AdminHandler) SensitiveUpdate(c *gin.Context) {
	var req SensitiveWord
	if !result.BindJSON(c, &req) {
		return
	}
	if err := h.svc.UpdateSensitiveWord(c.Request.Context(), &req); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}
