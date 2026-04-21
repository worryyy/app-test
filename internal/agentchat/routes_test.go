package agentchat

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRegisterProtectedRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	api := engine.Group("/api")
	RegisterProtectedRoutes(api, &Handler{})

	assertRouteExists(t, engine, "GET", "/api/agent/conversation")
	assertRouteExists(t, engine, "GET", "/api/agent/conversation/:id/history")
	assertRouteExists(t, engine, "DELETE", "/api/agent/conversation/:id")
	assertRouteExists(t, engine, "POST", "/api/agent/turn")
}

func TestRegisterInfraRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	RegisterInfraRoutes(engine, &Handler{})

	assertRouteExists(t, engine, "GET", "/agent/ws")
}

func assertRouteExists(t *testing.T, engine *gin.Engine, method, path string) {
	t.Helper()

	for _, route := range engine.Routes() {
		if route.Method == method && route.Path == path {
			return
		}
	}
	t.Fatalf("route %s %s not found", method, path)
}
