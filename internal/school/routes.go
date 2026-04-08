package school

import "github.com/gin-gonic/gin"

func RegisterPublicRoutes(engine *gin.Engine, handler *Handler) {
	engine.GET("/api/term", handler.CurTerm)
}

func RegisterProtectedRoutes(api *gin.RouterGroup, handler *Handler) {
	term := api.Group("/user")
	term.POST("/authentication", handler.Authenticate)
	term.POST("/re_authentication", handler.ReAuthenticate)
	term.POST("/get_course_by_weeks", handler.GetCourseByWeeks)
	term.POST("/get_exam", handler.GetExam)
	term.POST("/get_exam_score", handler.GetExamScore)
}
