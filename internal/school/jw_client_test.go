package school

import (
	"encoding/json"
	"testing"
)

func TestToJWLoginDataSupportsSnakeCase(t *testing.T) {
	data, err := toJWLoginData(map[string]interface{}{
		"is_login": true,
		"major":    "计算机科学与技术",
		"name":     "张三",
	})
	if err != nil {
		t.Fatalf("toJWLoginData returned error: %v", err)
	}
	if !data.IsLogin || data.Major != "计算机科学与技术" || data.Name != "张三" {
		t.Fatalf("unexpected jw login data: %+v", data)
	}
}

func TestJWLoginDataMarshalUsesSnakeCase(t *testing.T) {
	raw, err := json.Marshal(JWLoginData{
		IsLogin: true,
		Major:   "计算机科学与技术",
		Name:    "张三",
	})
	if err != nil {
		t.Fatalf("marshal JWLoginData: %v", err)
	}
	want := `{"is_login":true,"major":"计算机科学与技术","name":"张三"}`
	if string(raw) != want {
		t.Fatalf("unexpected json: got %s want %s", raw, want)
	}
}
