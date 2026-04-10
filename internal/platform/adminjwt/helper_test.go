package adminjwt

import (
	"context"
	"errors"
	"testing"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/config"
)

func TestGenerateAndVerifyAdminTokenPair(t *testing.T) {
	helper := NewHelper(config.AdminJWTConfig{
		Secret:              "admin-secret",
		TokenMinutes:        30,
		RefreshTokenMinutes: 120,
		Issue:               "campus-admin",
	}, nil)

	token, refreshToken, err := helper.GenerateTokenPair(&TokenUser{
		AdminID:  1,
		UserID:   2,
		Username: "root",
		Power:    2,
	})
	if err != nil {
		t.Fatalf("GenerateTokenPair() error = %v", err)
	}

	claims, err := helper.ParseAndVerifyAccess(context.Background(), token, nil)
	if err != nil {
		t.Fatalf("ParseAndVerifyAccess() error = %v", err)
	}
	if claims.AdminID != 1 || claims.UserID != 2 || claims.Username != "root" {
		t.Fatalf("unexpected access claims: %+v", claims)
	}

	refreshClaims, err := helper.ParseAndVerifyRefresh(context.Background(), refreshToken, nil)
	if err != nil {
		t.Fatalf("ParseAndVerifyRefresh() error = %v", err)
	}
	if refreshClaims.Subject != tokenSubjectRefresh {
		t.Fatalf("unexpected refresh subject: %s", refreshClaims.Subject)
	}

	if _, err := helper.ParseAndVerifyRefresh(context.Background(), token, nil); !errors.Is(err, ErrTokenInvalid) {
		t.Fatalf("expected access token to be rejected by refresh verifier, got %v", err)
	}
}

func TestGenerateTokenPairCreatesNewSessionIDByDefault(t *testing.T) {
	helper := NewHelper(config.AdminJWTConfig{
		Secret:              "admin-secret",
		TokenMinutes:        30,
		RefreshTokenMinutes: 120,
		Issue:               "campus-admin",
	}, nil)

	oldToken, _, err := helper.GenerateTokenPair(&TokenUser{
		AdminID:  9,
		UserID:   11,
		Username: "root",
		Power:    8,
	})
	if err != nil {
		t.Fatalf("GenerateTokenPair(old) error = %v", err)
	}
	newToken, _, err := helper.GenerateTokenPair(&TokenUser{
		AdminID:  9,
		UserID:   11,
		Username: "root",
		Power:    8,
	})
	if err != nil {
		t.Fatalf("GenerateTokenPair(new) error = %v", err)
	}

	oldClaims, err := helper.ParseAndVerifyAccess(context.Background(), oldToken, nil)
	if err != nil {
		t.Fatalf("ParseAndVerifyAccess(old) error = %v", err)
	}
	newClaims, err := helper.ParseAndVerifyAccess(context.Background(), newToken, nil)
	if err != nil {
		t.Fatalf("ParseAndVerifyAccess(new) error = %v", err)
	}
	if oldClaims.SessionID == "" || newClaims.SessionID == "" {
		t.Fatalf("session ids should not be empty: old=%q new=%q", oldClaims.SessionID, newClaims.SessionID)
	}
	if oldClaims.SessionID == newClaims.SessionID {
		t.Fatalf("expected distinct session ids for separate logins, got %q", oldClaims.SessionID)
	}
}
