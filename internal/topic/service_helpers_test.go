package topic

import (
	"testing"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/jwtutil"
)

func TestResolveTopicAuthorTargetUsesRequestedAnonymousAccount(t *testing.T) {
	target, err := resolveTopicAuthorTarget(&jwtutil.Claims{
		UserID:      11,
		AccountType: topicAccountTypeBase,
		RootUserID:  11,
	}, topicAccountTypeAnonymous)
	if err != nil {
		t.Fatalf("resolve target failed: %v", err)
	}

	if target.AccountType != topicAccountTypeAnonymous {
		t.Fatalf("unexpected account type: %q", target.AccountType)
	}
	if target.RootUserID != 11 {
		t.Fatalf("unexpected root user id: %d", target.RootUserID)
	}
	if target.UserID != 0 {
		t.Fatalf("anonymous target should not set direct user id, got %d", target.UserID)
	}
}

func TestResolveTopicAuthorTargetUsesRequestedBaseAccountFromAnonymousIdentity(t *testing.T) {
	target, err := resolveTopicAuthorTarget(&jwtutil.Claims{
		UserID:      22,
		AccountType: topicAccountTypeAnonymous,
		RootUserID:  11,
	}, topicAccountTypeBase)
	if err != nil {
		t.Fatalf("resolve target failed: %v", err)
	}

	if target.AccountType != topicAccountTypeBase {
		t.Fatalf("unexpected account type: %q", target.AccountType)
	}
	if target.RootUserID != 11 {
		t.Fatalf("unexpected root user id: %d", target.RootUserID)
	}
	if target.UserID != 11 {
		t.Fatalf("expected base identity user id 11, got %d", target.UserID)
	}
}

func TestResolveTopicAuthorTargetDefaultsToCurrentAnonymousIdentity(t *testing.T) {
	target, err := resolveTopicAuthorTarget(&jwtutil.Claims{
		UserID:      22,
		AccountType: topicAccountTypeAnonymous,
		RootUserID:  11,
	}, "")
	if err != nil {
		t.Fatalf("resolve target failed: %v", err)
	}

	if target.AccountType != topicAccountTypeAnonymous {
		t.Fatalf("unexpected account type: %q", target.AccountType)
	}
	if target.RootUserID != 11 {
		t.Fatalf("unexpected root user id: %d", target.RootUserID)
	}
}

func TestResolveTopicAuthorTargetInfersBaseWhenClaimsAccountTypeMissing(t *testing.T) {
	target, err := resolveTopicAuthorTarget(&jwtutil.Claims{
		UserID:     11,
		RootUserID: 11,
	}, "")
	if err != nil {
		t.Fatalf("resolve target failed: %v", err)
	}

	if target.AccountType != topicAccountTypeBase {
		t.Fatalf("unexpected inferred account type: %q", target.AccountType)
	}
	if target.UserID != 11 {
		t.Fatalf("unexpected inferred user id: %d", target.UserID)
	}
}

func TestResolveTopicAuthorTargetRejectsInvalidRequestedAccountType(t *testing.T) {
	_, err := resolveTopicAuthorTarget(&jwtutil.Claims{
		UserID:      11,
		AccountType: topicAccountTypeBase,
		RootUserID:  11,
	}, "merchant")
	if err == nil {
		t.Fatal("expected invalid account type error")
	}
}
