package user

import (
	"testing"
	"time"
)

func TestProvisionalGrantExpiresAt(t *testing.T) {
	loc := provisionalLocation()
	inWindow := time.Date(2026, time.June, 1, 0, 0, 0, 0, loc)
	want := time.Date(2026, time.October, 1, 0, 0, 0, 0, loc)

	got, ok := provisionalGrantExpiresAt(&User{}, inWindow)
	if !ok {
		t.Fatal("expected provisional grant in window")
	}
	if !got.Equal(want) {
		t.Fatalf("expected expiry %s, got %s", want, got)
	}

	if _, ok := provisionalGrantExpiresAt(&User{StuIsCheck: true}, inWindow); ok {
		t.Fatal("did not expect provisional grant for formally certified user")
	}

	beforeWindow := time.Date(2026, time.May, 31, 23, 59, 59, 0, loc)
	if _, ok := provisionalGrantExpiresAt(&User{}, beforeWindow); ok {
		t.Fatal("did not expect provisional grant before window")
	}

	atWindowEnd := time.Date(2026, time.October, 1, 0, 0, 0, 0, loc)
	if _, ok := provisionalGrantExpiresAt(&User{}, atWindowEnd); ok {
		t.Fatal("did not expect provisional grant at window end")
	}
}

func TestProvisionalGrantExpiresAtSkipsExistingCurrentExpiry(t *testing.T) {
	loc := provisionalLocation()
	inWindow := time.Date(2026, time.July, 10, 12, 0, 0, 0, loc)
	existing := time.Date(2026, time.October, 1, 0, 0, 0, 0, loc)

	if _, ok := provisionalGrantExpiresAt(&User{ProvisionalExpiresAt: &existing}, inWindow); ok {
		t.Fatal("did not expect provisional grant when existing expiry is current")
	}
}

func TestNewAnonymousUserInheritsRootCertificationFields(t *testing.T) {
	loc := provisionalLocation()
	expiresAt := time.Date(2026, time.October, 1, 0, 0, 0, 0, loc)
	root := &User{
		ID:                   1001,
		OpenID:               "openid-root",
		StuIsCheck:           false,
		ProvisionalExpiresAt: &expiresAt,
		Tag:                  "student",
		Gender:               "secret",
	}

	anonymous := newAnonymousUser(root, "avatar.png")
	if anonymous == nil {
		t.Fatal("expected anonymous user")
	}
	if anonymous.StuIsCheck {
		t.Fatal("expected anonymous user to inherit root StuIsCheck=false")
	}
	if anonymous.ProvisionalExpiresAt == nil || !anonymous.ProvisionalExpiresAt.Equal(expiresAt) {
		t.Fatalf("expected anonymous expiry %s, got %v", expiresAt, anonymous.ProvisionalExpiresAt)
	}
	if anonymous.RootUserID != root.ID {
		t.Fatalf("expected root user id %d, got %d", root.ID, anonymous.RootUserID)
	}
	if anonymous.OpenID != "openid-root:anon:1001" {
		t.Fatalf("unexpected anonymous openid %q", anonymous.OpenID)
	}
}
