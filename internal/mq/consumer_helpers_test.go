package mq

import "testing"

func TestResolveWXOpenIDOwnerID(t *testing.T) {
	tests := []struct {
		name    string
		user    *wxUserRecord
		want    int64
		wantErr bool
	}{
		{
			name: "base user without root keeps self",
			user: &wxUserRecord{
				ID:          101,
				AccountType: "base",
			},
			want: 101,
		},
		{
			name: "base user with self root keeps self",
			user: &wxUserRecord{
				ID:          102,
				RootUserID:  102,
				AccountType: "base",
			},
			want: 102,
		},
		{
			name: "anonymous user resolves to root identity",
			user: &wxUserRecord{
				ID:          201,
				RootUserID:  11,
				AccountType: "anonymous",
			},
			want: 11,
		},
		{
			name: "anonymous user without root is rejected",
			user: &wxUserRecord{
				ID:          202,
				AccountType: "anonymous",
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveWXOpenIDOwnerID(tc.user)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("want %d, got %d", tc.want, got)
			}
		})
	}
}
