package ecampuscrm

import (
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/comment"
	"github.com/Milchstrassse/Ecampus-go/internal/school"
	"github.com/Milchstrassse/Ecampus-go/internal/sensitive"
	"github.com/Milchstrassse/Ecampus-go/internal/topic"
	"github.com/Milchstrassse/Ecampus-go/internal/user"
)

func TestRegisterRoutesExposesExpectedAdminEndpoints(t *testing.T) {
	engine := gin.New()
	registerRoutes(engine, zap.NewNop(), nil, nil, nil, adminHandlers{
		User:      user.NewAdminHandler(nil),
		School:    school.NewAdminHandler(nil),
		Sensitive: sensitive.NewAdminHandler(nil),
		Topic:     topic.NewAdminHandler(nil),
		Comment:   comment.NewAdminHandler(nil),
	})

	routes := routeSet(engine.Routes())

	expected := []routeKey{
		{method: "POST", path: "/admin/user/login"},
		{method: "POST", path: "/admin/user/refresh"},
		{method: "POST", path: "/admin/user/logout"},
		{method: "POST", path: "/admin/user/user_token"},
		{method: "PUT", path: "/admin/user/pre_authentication"},
		{method: "PUT", path: "/admin/user/pre_authentication/batch"},
		{method: "POST", path: "/admin/user"},
		{method: "PUT", path: "/admin/user/:id"},
		{method: "GET", path: "/admin/user/list"},
		{method: "POST", path: "/admin/user/clear"},
		{method: "GET", path: "/admin/term"},
		{method: "GET", path: "/admin/term/list"},
		{method: "POST", path: "/admin/term"},
		{method: "DELETE", path: "/admin/term/:id"},
		{method: "POST", path: "/admin/term/cur"},
		{method: "GET", path: "/admin/sensitive/page"},
		{method: "GET", path: "/admin/sensitive/search_like"},
		{method: "POST", path: "/admin/sensitive/add"},
		{method: "DELETE", path: "/admin/sensitive/deleteByWord"},
		{method: "GET", path: "/admin/topic"},
		{method: "POST", path: "/admin/topic"},
		{method: "PATCH", path: "/admin/topic/:topic_id"},
		{method: "DELETE", path: "/admin/topic/:topic_id"},
		{method: "GET", path: "/admin/topic/:topic_id/comment"},
		{method: "DELETE", path: "/admin/topic/:topic_id/comment/:comment_id"},
	}

	for _, want := range expected {
		if !routes[want] {
			t.Fatalf("missing route %s %s", want.method, want.path)
		}
	}
}

func TestRegisterRoutesRemovesDeprecatedAdminEndpoints(t *testing.T) {
	engine := gin.New()
	registerRoutes(engine, zap.NewNop(), nil, nil, nil, adminHandlers{
		User:      user.NewAdminHandler(nil),
		School:    school.NewAdminHandler(nil),
		Sensitive: sensitive.NewAdminHandler(nil),
		Topic:     topic.NewAdminHandler(nil),
		Comment:   comment.NewAdminHandler(nil),
	})

	routes := routeSet(engine.Routes())
	deprecated := []routeKey{
		{method: "POST", path: "/admin/user/add"},
		{method: "DELETE", path: "/admin/user/:id"},
		{method: "GET", path: "/admin/user/:id"},
		{method: "POST", path: "/admin/user/course"},
		{method: "DELETE", path: "/admin/comment/:topic_id/:comment_id"},
		{method: "GET", path: "/admin/theme"},
		{method: "GET", path: "/admin/file"},
		{method: "GET", path: "/admin/sensitive/getAllList"},
		{method: "GET", path: "/admin/sensitive/getByWord"},
		{method: "GET", path: "/admin/sensitive/getByWord/"},
		{method: "DELETE", path: "/admin/sensitive/batchDelete"},
		{method: "POST", path: "/admin/sensitive/batchAdd"},
		{method: "PUT", path: "/admin/sensitive/update"},
	}

	for _, route := range deprecated {
		if routes[route] {
			t.Fatalf("deprecated route still registered: %s %s", route.method, route.path)
		}
	}
}

type routeKey struct {
	method string
	path   string
}

func routeSet(routes gin.RoutesInfo) map[routeKey]bool {
	out := make(map[routeKey]bool, len(routes))
	for _, route := range routes {
		out[routeKey{method: route.Method, path: route.Path}] = true
	}
	return out
}
