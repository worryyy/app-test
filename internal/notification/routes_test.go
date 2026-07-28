package notification

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRegistersNotificationAndLegacyRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	handler := NewHandler(&Service{}, nil)
	RegisterInfraRoutes(engine, handler)
	RegisterProtectedRoutes(engine.Group("/api"), handler)

	want := map[string]bool{
		"GET /notification/ws":                false,
		"GET /api/notification":               false,
		"GET /api/notification/unread-counts": false,
		"PUT /api/notification/:id/read":      false,
		"PUT /api/notification/read":          false,
		"GET /api/notify":                     false,
		"GET /api/notify/:type/haveUnread":    false,
		"GET /api/notify/:type":               false,
	}
	for _, route := range engine.Routes() {
		key := route.Method + " " + route.Path
		if _, ok := want[key]; ok {
			want[key] = true
		}
	}
	for route, found := range want {
		if !found {
			t.Fatalf("route %s not registered", route)
		}
	}
}
