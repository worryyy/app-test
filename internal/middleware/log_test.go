package middleware

import (
	"strings"
	"testing"
)

func TestMaskedRequestPathMasksSensitiveQueryValues(t *testing.T) {
	got := maskedRequestPath("/admin/user/pre_authentication", "user_id=1&pwd=secret&token=abc123&nickname=tester")
	if strings.Contains(got, "secret") {
		t.Fatalf("maskedRequestPath() leaked pwd: %s", got)
	}
	if strings.Contains(got, "abc123") {
		t.Fatalf("maskedRequestPath() leaked token: %s", got)
	}
	if !strings.Contains(got, "nickname=tester") {
		t.Fatalf("maskedRequestPath() should keep non-sensitive params: %s", got)
	}
	if !strings.Contains(got, "pwd=%2A%2A%2A") || !strings.Contains(got, "token=%2A%2A%2A") {
		t.Fatalf("maskedRequestPath() should replace sensitive values: %s", got)
	}
}
