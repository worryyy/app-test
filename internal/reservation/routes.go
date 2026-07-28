package reservation

import "github.com/gin-gonic/gin"

func RegisterProtectedRoutes(api *gin.RouterGroup, h *Handler) {
	g := api.Group("/reservation")
	g.GET("/venues", h.Venues)
	g.GET("/venues/:id/resources", h.Resources)
	g.GET("/resources/:id/slots", h.Slots)
	g.POST("/bookings", h.Create)
	g.GET("/bookings", h.Mine)
	g.DELETE("/bookings/:id", h.Cancel)
}
func RegisterAdminRoutes(admin *gin.RouterGroup, h *AdminHandler) {
	g := admin.Group("/reservation")
	g.POST("/venues", h.CreateVenue)
	g.POST("/venues/:id/resources", h.CreateResource)
	g.POST("/resources/:id/rules", h.CreateRule)
	g.POST("/closures", h.CreateClosure)
	g.POST("/checkin", h.Checkin)
}
