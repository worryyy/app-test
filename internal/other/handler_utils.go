package other

import "github.com/gin-gonic/gin"
import "strconv"

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
