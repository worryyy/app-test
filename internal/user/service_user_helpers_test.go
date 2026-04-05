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

func TestNormalizedUserReturnsCopyWithDefaults(t *testing.T) {
	svc := &Service{}
	raw := &User{ID: 42, StuPwd: "secret"}

	normalized := svc.normalizedUser(raw)
	if normalized == nil {
		t.Fatal("expected normalized user")
	}
	if normalized == raw {
		t.Fatal("expected normalized user to be a copy")
	}
	if normalized.RootUserID != 42 {
		t.Fatalf("expected root user id 42, got %d", normalized.RootUserID)
	}
	if normalized.AccountType != accountTypeBase {
		t.Fatalf("expected account type %q, got %q", accountTypeBase, normalized.AccountType)
	}
	if normalized.LastSwitchID == nil || *normalized.LastSwitchID != 42 {
		t.Fatalf("expected last switch id 42, got %#v", normalized.LastSwitchID)
	}
	if raw.RootUserID != 0 || raw.AccountType != "" || raw.LastSwitchID != nil {
		t.Fatalf("expected raw user to stay unchanged, got %+v", raw)
	}
}

func TestSanitizeUserDoesNotMutateInput(t *testing.T) {
	svc := &Service{}
	raw := &User{ID: 7, StuPwd: "secret"}

	sanitized := svc.sanitizeUser(raw)
	if sanitized == nil {
		t.Fatal("expected sanitized user")
	}
	if sanitized.StuPwd != "" {
		t.Fatalf("expected empty password, got %q", sanitized.StuPwd)
	}
	if raw.StuPwd != "secret" {
		t.Fatalf("expected raw password to stay unchanged, got %q", raw.StuPwd)
	}
	if raw.RootUserID != 0 || raw.AccountType != "" || raw.LastSwitchID != nil {
		t.Fatalf("expected sanitize to avoid mutating raw user defaults, got %+v", raw)
	}
}
