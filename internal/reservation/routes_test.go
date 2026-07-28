package reservation

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRoutesExposeUserAndAdminEndpoints(t *testing.T) {
	engine := gin.New()
	RegisterProtectedRoutes(engine.Group("/api"), NewHandler(nil))
	RegisterAdminRoutes(engine.Group("/admin"), NewAdminHandler(nil))

	routes := make(map[string]bool)
	for _, route := range engine.Routes() {
		routes[route.Method+" "+route.Path] = true
	}
	expected := []string{
		"GET /api/reservation/venues",
		"GET /api/reservation/venues/:id/resources",
		"GET /api/reservation/resources/:id/slots",
		"POST /api/reservation/bookings",
		"GET /api/reservation/bookings",
		"DELETE /api/reservation/bookings/:id",
		"POST /admin/reservation/venues",
		"POST /admin/reservation/venues/:id/resources",
		"POST /admin/reservation/resources/:id/rules",
		"POST /admin/reservation/closures",
		"POST /admin/reservation/checkin",
	}
	for _, route := range expected {
		if !routes[route] {
			t.Fatalf("missing route %s", route)
		}
	}
}
