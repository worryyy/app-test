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
	var words []string
	if !result.BindJSON(c, &words) {
		return
	}
	if err := h.svc.BatchDeleteSensitiveWords(c.Request.Context(), words); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *AdminHandler) SensitiveAdd(c *gin.Context) {
	word := c.Query("word")
	if word == "" {
		result.Fail(c, result.CodeParamError, result.ErrParam.Error())
		return
	}
	if err := h.svc.AddSensitiveWord(c.Request.Context(), &SensitiveWord{Word: word}); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}

func (h *AdminHandler) SensitiveBatchAdd(c *gin.Context) {
	var words []string
	if !result.BindJSON(c, &words) {
		return
	}
	if err := h.svc.BatchAddSensitiveWords(c.Request.Context(), words); err != nil {
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
	word := c.Query("word")
	updateWord := c.Query("updateWord")
	if word == "" || updateWord == "" {
		result.Fail(c, result.CodeParamError, result.ErrParam.Error())
		return
	}
	if err := h.svc.UpdateSensitiveWordByWord(c.Request.Context(), word, updateWord); err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, nil)
}
