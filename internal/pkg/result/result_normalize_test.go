package result

import (
	"reflect"
	"testing"
	"time"
)

type normalizeChild struct {
	Flag *bool `json:"flag"`
}

type normalizeSample struct {
	Name      *string         `json:"name"`
	Count     *int64          `json:"count"`
	Items     []int           `json:"items"`
	Child     *normalizeChild `json:"child"`
	CreatedAt time.Time       `json:"createdAt"`
}

func TestNormalizeData_RecursiveFields(t *testing.T) {
	ts := time.Date(2026, 3, 30, 12, 34, 56, 0, time.Local)
	got := normalizeData(normalizeSample{
		Name:      nil,
		Count:     nil,
		Items:     nil,
		Child:     &normalizeChild{Flag: nil},
		CreatedAt: ts,
	})

	m, ok := got.(map[string]interface{})
	if !ok {
		t.Fatalf("normalizeData type = %T, want map[string]interface{}", got)
	}

	if m["name"] != "" {
		t.Fatalf("name = %#v, want empty string", m["name"])
	}
	if m["count"] != int64(0) {
		t.Fatalf("count = %#v, want int64(0)", m["count"])
	}

	items, ok := m["items"].([]interface{})
	if !ok || len(items) != 0 {
		t.Fatalf("items = %#v, want empty []interface{}", m["items"])
	}

	child, ok := m["child"].(map[string]interface{})
	if !ok {
		t.Fatalf("child = %#v, want map[string]interface{}", m["child"])
	}
	if child["flag"] != false {
		t.Fatalf("child.flag = %#v, want false", child["flag"])
	}

	wantTime := ts.Format("2006-01-02 15:04:05")
	if m["createdAt"] != wantTime {
		t.Fatalf("createdAt = %#v, want %q", m["createdAt"], wantTime)
	}
}

func TestNormalizeData_NilSliceInput(t *testing.T) {
	var in []int
	got := normalizeData(in)
	want := []interface{}{}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeData(nil slice) = %#v, want %#v", got, want)
	}
}
