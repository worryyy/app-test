package chat

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/responses"
)

func bindQuery(c *gin.Context, req any) bool {
	if err := c.ShouldBindQuery(req); err != nil {
		responses.Fail(c, bizerr.Param(errMsgInvalidParam))
		return false
	}
	return true
}

func pageSize(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "15"))
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 15
	} else if size > maxPageSize {
		size = maxPageSize
	}
	return page, size
}

func parsePositiveInt64(raw string) (int64, error) {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || value <= 0 {
		return 0, bizerr.Param(errMsgInvalidParam)
	}
	return value, nil
}

func parseOptionalPositiveInt64(raw string) (*int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	value, err := parsePositiveInt64(raw)
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func writeNotificationListResult(c *gin.Context, data *PageResult[Notification]) {
	if data == nil || len(data.Data) == 0 {
		responses.Success.RespData(c, []Notification{})
		return
	}
	responses.Success.RespData(c, data)
}
