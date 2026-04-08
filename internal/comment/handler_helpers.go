package comment

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/responses"
)

func bindJSON(c *gin.Context, req any) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		responses.Fail(c, bizerr.Param(errMsgInvalidParam))
		return false
	}
	return true
}

func bindQuery(c *gin.Context, req any) bool {
	if err := c.ShouldBindQuery(req); err != nil {
		responses.Fail(c, bizerr.Param(errMsgInvalidParam))
		return false
	}
	return true
}

func bindURI(c *gin.Context, req any) bool {
	if err := c.ShouldBindUri(req); err != nil {
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
	}
	return page, size
}

func writeCommentListResult(c *gin.Context, data *PageResult[Comment], emptyAsList bool) {
	if emptyAsList && (data == nil || len(data.Data) == 0) {
		responses.Success.RespData(c, []Comment{})
		return
	}
	responses.Success.RespData(c, data)
}
