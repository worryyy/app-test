package ginutil

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/responses"
)

const defaultPageSize = 15

func BindJSON(c *gin.Context, req any, invalidParamMessage string) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		responses.Fail(c, bizerr.Param(invalidParamMessage))
		return false
	}
	return true
}

func BindQuery(c *gin.Context, req any, invalidParamMessage string) bool {
	if err := c.ShouldBindQuery(req); err != nil {
		responses.Fail(c, bizerr.Param(invalidParamMessage))
		return false
	}
	return true
}

func BindURI(c *gin.Context, req any, invalidParamMessage string) bool {
	if err := c.ShouldBindUri(req); err != nil {
		responses.Fail(c, bizerr.Param(invalidParamMessage))
		return false
	}
	return true
}

func PageSize(c *gin.Context, maxSize int) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", strconv.Itoa(defaultPageSize)))
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = defaultPageSize
	} else if maxSize > 0 && size > maxSize {
		size = maxSize
	}
	return page, size
}

func ParseOptionalPositiveInt64(raw string, invalidParamMessage string) (*int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return nil, bizerr.Param(invalidParamMessage)
	}
	return &value, nil
}
