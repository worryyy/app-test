package topic

import (
	"testing"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/jwtutil"
)

func TestNormalizeTopicAccountType(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "base", input: "base", want: topicAccountTypeBase},
		{name: "anonymous", input: "anonymous", want: topicAccountTypeAnonymous},
		{name: "trimmed", input: " anonymous ", want: topicAccountTypeAnonymous},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeTopicAccountType(tt.input)
			if err != nil {
				t.Fatalf("normalizeTopicAccountType returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("normalizeTopicAccountType = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeTopicAccountTypeRejectsInvalidValue(t *testing.T) {
	_, err := normalizeTopicAccountType("merchant")
	if err == nil {
		t.Fatal("expected invalid account type error")
	}
}

func TestValidateTopicClaimsAcceptsValidClaims(t *testing.T) {
	err := validateTopicClaims(&jwtutil.Claims{
		UserID:      22,
		RootUserID:  11,
		AccountType: topicAccountTypeAnonymous,
	})
	if err != nil {
		t.Fatalf("validateTopicClaims returned error: %v", err)
	}
}

func TestValidateTopicClaimsRejectsNilClaimsAsUnauthorized(t *testing.T) {
	err := validateTopicClaims(nil)
	if err == nil {
		t.Fatal("expected nil claims error")
	}

	bizErr, ok := err.(*bizerr.Error)
	if !ok {
		t.Fatalf("expected bizerr.Error, got %T", err)
	}
	if bizErr.Code != bizerr.CodeUnauthorized {
		t.Fatalf("unexpected error code: got %d want %d", bizErr.Code, bizerr.CodeUnauthorized)
	}
}

func TestValidateTopicClaimsRejectsMissingRootUserIDAsUnauthorized(t *testing.T) {
	err := validateTopicClaims(&jwtutil.Claims{
		UserID:      11,
		RootUserID:  0,
		AccountType: topicAccountTypeBase,
	})
	if err == nil {
		t.Fatal("expected missing root user id error")
	}

	bizErr, ok := err.(*bizerr.Error)
	if !ok {
		t.Fatalf("expected bizerr.Error, got %T", err)
	}
	if bizErr.Code != bizerr.CodeUnauthorized {
		t.Fatalf("unexpected error code: got %d want %d", bizErr.Code, bizerr.CodeUnauthorized)
	}
}
