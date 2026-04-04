package school

import "testing"

func TestDecodeJWLoginMetaSupportsSnakeCase(t *testing.T) {
	loggedIn, name, major, err := decodeJWLoginMeta(map[string]any{
		"is_login": true,
		"name":     "张三",
		"major":    "计算机科学与技术",
	})
	if err != nil {
		t.Fatalf("decodeJWLoginMeta returned error: %v", err)
	}
	if !loggedIn {
		t.Fatalf("expected login success")
	}
	if name != "张三" {
		t.Fatalf("unexpected name: %s", name)
	}
	if major != "计算机科学与技术" {
		t.Fatalf("unexpected major: %s", major)
	}
}

func TestDecodeJWLoginMetaSupportsCamelCase(t *testing.T) {
	loggedIn, name, major, err := decodeJWLoginMeta(map[string]any{
		"isLogin": true,
		"name":    "李四",
		"major":   "软件工程",
	})
	if err != nil {
		t.Fatalf("decodeJWLoginMeta returned error: %v", err)
	}
	if !loggedIn || name != "李四" || major != "软件工程" {
		t.Fatalf("unexpected decoded login meta: loggedIn=%v name=%s major=%s", loggedIn, name, major)
	}
}
