package academic

import (
	"github.com/gin-gonic/gin"
	"testing"
)

func TestRegistersAcademicRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	RegisterProtectedRoutes(engine.Group("/api"), NewHandler(&Service{}))
	RegisterAdminRoutes(engine.Group("/admin"), NewAdminHandler(&Service{}))
	want := map[string]bool{"POST /api/academic/courses": false, "PUT /api/academic/courses/:id/review": false, "POST /api/academic/courses/:id/materials": false, "GET /api/academic/materials/:id/download": false, "PUT /admin/academic/courses/:id/merge": false}
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
