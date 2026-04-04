package user

import "testing"

func TestBuildTokenUserUsesEndUserClaims(t *testing.T) {
	svc := &Service{}
	identity := &User{
		ID:          205,
		AccountType: accountTypeAnonymous,
		Power:       6,
	}
	rootUser := &User{
		ID:          101,
		OpenID:      "wx-root-openid",
		AccountType: accountTypeBase,
		Power:       6,
	}

	tokenUser := svc.buildTokenUser(identity, rootUser)
	if tokenUser == nil {
		t.Fatal("expected token user")
	}
	if tokenUser.ID != identity.ID {
		t.Fatalf("expected identity id %d, got %d", identity.ID, tokenUser.ID)
	}
	if tokenUser.OpenID != rootUser.OpenID {
		t.Fatalf("expected openid %q, got %q", rootUser.OpenID, tokenUser.OpenID)
	}
	if tokenUser.Power != 0 {
		t.Fatalf("expected end-user token power 0, got %d", tokenUser.Power)
	}
	if tokenUser.AccountType != identity.AccountType {
		t.Fatalf("expected account type %q, got %q", identity.AccountType, tokenUser.AccountType)
	}
	if tokenUser.RootUserID != rootUser.ID {
		t.Fatalf("expected root user id %d, got %d", rootUser.ID, tokenUser.RootUserID)
	}
}

func TestBuildAdminTokenUserPreservesAdminPower(t *testing.T) {
	svc := &Service{}
	adminUser := &User{
		ID:          301,
		AccountType: accountTypeBase,
		Power:       10,
	}
	rootUser := &User{
		ID:          301,
		OpenID:      "wx-admin-openid",
		AccountType: accountTypeBase,
	}

	tokenUser := svc.buildAdminTokenUser(adminUser, rootUser)
	if tokenUser == nil {
		t.Fatal("expected token user")
	}
	if tokenUser.Power != adminUser.Power {
		t.Fatalf("expected admin power %d, got %d", adminUser.Power, tokenUser.Power)
	}
	if tokenUser.RootUserID != rootUser.ID {
		t.Fatalf("expected root user id %d, got %d", rootUser.ID, tokenUser.RootUserID)
	}
}
