package wxutil

import "testing"

func TestStringifyWXLabel(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  string
	}{
		{name: "nil", input: nil, want: ""},
		{name: "string", input: "100", want: "100"},
		{name: "number", input: 200.0, want: "200"},
	}

	for _, tt := range tests {
		if got := stringifyWXLabel(tt.input); got != tt.want {
			t.Fatalf("%s: got %q want %q", tt.name, got, tt.want)
		}
	}
}
