package user

import (
	"testing"
	"time"
)

func TestIdentitySwitchReqResolvedAccountType(t *testing.T) {
	tests := []struct {
		name string
		req  IdentitySwitchReq
		want string
	}{
		{
			name: "prefers camel case field",
			req: IdentitySwitchReq{
				AccountType:       "base",
				LegacyAccountType: "anonymous",
			},
			want: "base",
		},
		{
			name: "falls back to legacy field",
			req: IdentitySwitchReq{
				LegacyAccountType: "anonymous",
			},
			want: "anonymous",
		},
		{
			name: "trims blank values",
			req: IdentitySwitchReq{
				AccountType:       "   ",
				LegacyAccountType: " official ",
			},
			want: "official",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.req.ResolvedAccountType(); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestNormalizeUserEditReqIgnoresBlankOnlyFields(t *testing.T) {
	req := normalizeUserEditReq(UserEditReq{
		Nickname:  "   ",
		Avatar:    " avatar.png ",
		Gender:    "\n",
		Signature: "\t",
	})

	if req.Nickname != "" {
		t.Fatalf("expected blank nickname to be dropped, got %q", req.Nickname)
	}
	if req.Avatar != " avatar.png " {
		t.Fatalf("expected avatar value to be preserved, got %q", req.Avatar)
	}
	if req.Gender != "" {
		t.Fatalf("expected blank gender to be dropped, got %q", req.Gender)
	}
	if req.Signature != "" {
		t.Fatalf("expected blank signature to be dropped, got %q", req.Signature)
	}
}

func TestBuildUserProfileUpdatesIncludesAuditFields(t *testing.T) {
	now := time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC)
	updates := buildUserProfileUpdates(101, UserEditReq{
		Nickname:  "Alice2",
		Avatar:    "   ",
		Signature: "new sig",
	}, now)

	if got := updates["nickname"]; got != "Alice2" {
		t.Fatalf("expected nickname update, got %#v", got)
	}
	if _, exists := updates["avatar"]; exists {
		t.Fatal("expected blank avatar to be omitted")
	}
	if got := updates["signature"]; got != "new sig" {
		t.Fatalf("expected signature update, got %#v", got)
	}
	if got := updates["updated_by"]; got != int64(101) {
		t.Fatalf("expected updated_by=101, got %#v", got)
	}
	if got := updates["updated_at"]; got != now {
		t.Fatalf("expected updated_at=%v, got %#v", now, got)
	}
}
