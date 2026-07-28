package marketplace

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRoutesExposeMarketplaceEndpoints(t *testing.T) {
	engine := gin.New()
	RegisterProtectedRoutes(engine.Group("/api"), NewHandler(nil))
	RegisterAdminRoutes(engine.Group("/admin"), NewAdminHandler(nil))
	routes := map[string]bool{}
	for _, route := range engine.Routes() {
		routes[route.Method+" "+route.Path] = true
	}
	expected := []string{
		"GET /api/marketplace/categories",
		"GET /api/marketplace/items",
		"POST /api/marketplace/items",
		"POST /api/marketplace/orders",
		"POST /api/marketplace/orders/:id/pay",
		"PUT /api/marketplace/orders/:id/delivered",
		"PUT /api/marketplace/orders/:id/received",
		"POST /api/marketplace/orders/:id/refunds",
		"POST /api/marketplace/orders/:id/disputes",
		"POST /api/marketplace/payments/test/callback",
		"POST /admin/marketplace/categories",
		"GET /admin/marketplace/orders",
		"PUT /admin/marketplace/refunds/:id/decision",
		"PUT /admin/marketplace/disputes/:id/decision",
		"GET /admin/marketplace/settlements",
	}
	for _, route := range expected {
		if !routes[route] {
			t.Fatalf("missing route %s", route)
		}
	}
}
