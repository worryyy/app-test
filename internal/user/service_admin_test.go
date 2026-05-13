package user

import (
	"testing"
)

func TestBuildAdminUserJWTUserUsesLinkedUserItself(t *testing.T) {
	lastSwitchID := int64(2002)
	rootUser := User{
		ID:           1001,
		OpenID:       "openid-root",
		Nickname:     "root",
		AccountType:  accountTypeBase,
		RootUserID:   1001,
		LastSwitchID: &lastSwitchID,
	}

	got := (&Service{}).buildAdminUserJWTUser(&rootUser)
	if got == nil {
		t.Fatal("expected token user")
	}
	if got.ID != rootUser.ID {
		t.Fatalf("expected linked user id %d, got %d", rootUser.ID, got.ID)
	}
	if got.RootUserID != rootUser.ID {
		t.Fatalf("expected root user id %d, got %d", rootUser.ID, got.RootUserID)
	}
	if got.AccountType != accountTypeBase {
		t.Fatalf("expected account type %s, got %s", accountTypeBase, got.AccountType)
	}
}
