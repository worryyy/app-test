package sensitive

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/pagination"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

type AdminHandler struct {
	svc *Service
}

func NewAdminHandler(svc *Service) *AdminHandler {
	return &AdminHandler{svc: svc}
}

func (h *AdminHandler) SensitiveGetAllList(c *gin.Context) {
	data, err := h.svc.GetAllSensitiveWords(c.Request.Context())
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Data(c, data)
}

func (h *AdminHandler) SensitiveGetByWord(c *gin.Context) {
	word := strings.TrimSpace(c.Query("word"))
	if word == "" {
		result.Fail(c, result.CodeFail, "参数为NULL，请重试")
		return
	}
	data, err := h.svc.GetSensitiveWordByWord(c.Request.Context(), word)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Data(c, data)
}

func (h *AdminHandler) SensitiveDeleteByWord(c *gin.Context) {
	word := strings.TrimSpace(c.Query("word"))
	if word == "" {
		result.Fail(c, result.CodeFail, "参数为NULL，请重试")
		return
	}
	if err := h.svc.DeleteSensitiveWordByWord(c.Request.Context(), word); err != nil {
		result.HandleError(c, err)
		return
	}
	result.SuccessMsg(c, fmt.Sprintf("删除关键词：%s成功", word), nil)
}

func (h *AdminHandler) SensitiveBatchDelete(c *gin.Context) {
	var words []string
	if !result.BindJSON(c, &words) {
		return
	}
	if len(words) == 0 {
		result.Fail(c, result.CodeFail, "参数为NULL，请重试")
		return
	}
	if err := h.svc.BatchDeleteSensitiveWords(c.Request.Context(), words); err != nil {
		result.HandleError(c, err)
		return
	}
	result.SuccessMsg(c, "批量删除关键词成功", nil)
}

func (h *AdminHandler) SensitiveAdd(c *gin.Context) {
	word := strings.TrimSpace(c.Query("word"))
	if word == "" {
		result.Fail(c, result.CodeFail, "参数为NULL，请重试")
		return
	}
	if err := h.svc.AddSensitiveWord(c.Request.Context(), &SensitiveWord{Word: word}); err != nil {
		result.HandleError(c, err)
		return
	}
	result.SuccessMsg(c, fmt.Sprintf("关键词：%s添加成功", word), nil)
}

func (h *AdminHandler) SensitiveBatchAdd(c *gin.Context) {
	var words []string
	if !result.BindJSON(c, &words) {
		return
	}
	if len(words) == 0 {
		result.Fail(c, result.CodeFail, "参数为NULL，请重试")
		return
	}
	if err := h.svc.BatchAddSensitiveWords(c.Request.Context(), words); err != nil {
		result.HandleError(c, err)
		return
	}
	result.SuccessMsg(c, "关键词批量添加成功", nil)
}

func (h *AdminHandler) SensitivePage(c *gin.Context) {
	page, size := pagination.PageSize(c)
	data, err := h.svc.SensitiveWordPage(c.Request.Context(), page, size)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Data(c, data)
}

func (h *AdminHandler) SensitiveSearchLike(c *gin.Context) {
	word := strings.TrimSpace(c.Query("word"))
	if word == "" {
		result.Fail(c, result.CodeFail, "参数为NULL，请重试")
		return
	}
	data, err := h.svc.SearchSensitiveWordLike(c.Request.Context(), word)
	if err != nil {
		result.HandleError(c, err)
		return
	}
	result.Success(c, data)
}

func (h *AdminHandler) SensitiveUpdate(c *gin.Context) {
	word := strings.TrimSpace(c.Query("word"))
	updateWord := strings.TrimSpace(c.Query("updateWord"))
	if word == "" || updateWord == "" {
		result.Fail(c, result.CodeFail, "参数为NULL，请重试")
		return
	}
	if err := h.svc.UpdateSensitiveWordByWord(c.Request.Context(), word, updateWord); err != nil {
		result.HandleError(c, err)
		return
	}
	result.SuccessMsg(c, "更新成功", nil)
}
