package moderation

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRegistersUserAndAdminRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	RegisterProtectedRoutes(engine.Group("/api"), NewHandler(&Service{}))
	RegisterAdminRoutes(engine.Group("/admin"), NewAdminHandler(&Service{}))
	want := map[string]bool{
		"POST /api/moderation/reports":                 false,
		"GET /api/moderation/punishments":              false,
		"POST /api/moderation/punishments/:id/appeals": false,
		"PUT /admin/moderation/reports/:id/claim":      false,
		"PUT /admin/moderation/reports/:id/decision":   false,
		"PUT /admin/moderation/appeals/:id/decision":   false,
	}
	for _, route := range engine.Routes() {
		key := route.Method + " " + route.Path
		if _, ok := want[key]; ok {
			want[key] = true
		}
	}
	for route, ok := range want {
		if !ok {
			t.Fatalf("route %s missing", route)
		}
	}
}
