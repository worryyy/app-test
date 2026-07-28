package academic

import "github.com/gin-gonic/gin"

func RegisterProtectedRoutes(api *gin.RouterGroup, h *Handler) {
	g := api.Group("/academic")
	g.GET("/courses", h.Search)
	g.POST("/courses", h.CreateCourse)
	g.GET("/courses/:id", h.CourseDetail)
	g.GET("/courses/:id/reviews", h.Reviews)
	g.PUT("/courses/:id/review", h.SaveReview)
	g.DELETE("/courses/:id/review", h.DeleteReview)
	g.POST("/courses/:id/materials", h.UploadMaterial)
	g.GET("/courses/:id/materials", h.Materials)
	g.GET("/materials/mine", h.MyMaterials)
	g.GET("/materials/:id/download", h.DownloadMaterial)
	g.DELETE("/materials/:id", h.DeleteMaterial)
}

func RegisterAdminRoutes(admin *gin.RouterGroup, h *AdminHandler) {
	g := admin.Group("/academic")
	g.PUT("/courses/:id/merge", h.Merge)
	g.PUT("/courses/:id/hidden", h.HideCourse)
	g.PUT("/reviews/:id/hidden", h.HideReview)
	g.PUT("/materials/:id/hidden", h.HideMaterial)
}
