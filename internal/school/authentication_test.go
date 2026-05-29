package school

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/jwtutil"
)

func TestCurrentRootUserIDUsesClaimsRootForAnonymous(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("claims", &jwtutil.Claims{
		UserID:      1002,
		AccountType: "anonymous",
		RootUserID:  1001,
	})

	if got := currentRootUserID(c); got != 1001 {
		t.Fatalf("expected root user id 1001, got %d", got)
	}
}

func TestCurrentRootUserIDFallsBackToCurrentUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("claims", &jwtutil.Claims{
		UserID:      1001,
		AccountType: "base",
	})

	if got := currentRootUserID(c); got != 1001 {
		t.Fatalf("expected current user id 1001, got %d", got)
	}
}
