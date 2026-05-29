package middleware

import (
	"testing"
	"time"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/jwtutil"
	"github.com/Milchstrassse/Ecampus-go/internal/user"
)

func TestCertificationSubjectIDUsesRootForAnonymous(t *testing.T) {
	got := certificationSubjectID(&jwtutil.Claims{
		UserID:      1002,
		AccountType: "anonymous",
		RootUserID:  1001,
	})
	if got != 1001 {
		t.Fatalf("expected anonymous certification subject to be root user 1001, got %d", got)
	}
}

func TestCertificationSubjectIDUsesCurrentForBase(t *testing.T) {
	got := certificationSubjectID(&jwtutil.Claims{
		UserID:      1001,
		AccountType: "base",
		RootUserID:  1001,
	})
	if got != 1001 {
		t.Fatalf("expected base certification subject to be current user 1001, got %d", got)
	}
}

func TestHasCertifiedAccessAllowsFormalOrUnexpiredProvisional(t *testing.T) {
	now := time.Date(2026, time.July, 1, 12, 0, 0, 0, time.UTC)
	if !hasCertifiedAccess(user.User{StuIsCheck: true}, now) {
		t.Fatal("expected formally certified user to pass")
	}

	expiresAt := now.Add(time.Hour)
	if !hasCertifiedAccess(user.User{ProvisionalExpiresAt: &expiresAt}, now) {
		t.Fatal("expected unexpired provisional user to pass")
	}

	expiredAt := now.Add(-time.Hour)
	if hasCertifiedAccess(user.User{ProvisionalExpiresAt: &expiredAt}, now) {
		t.Fatal("expected expired provisional user to fail")
	}

	if hasCertifiedAccess(user.User{}, now) {
		t.Fatal("expected unauthenticated user to fail")
	}
}
