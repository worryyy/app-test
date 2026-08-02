package ciimpact

import "testing"

func TestProtoCheckSurvivesFullFallback(t *testing.T) {
	manifest, graph := fixture()
	result := Evaluate(manifest, []Change{
		{Status: "M", Path: "proto/agent/v1/agent.proto"},
		{Status: "M", Path: "unknown/file.go"},
	}, graph)
	if result.FallbackReason == "" || !result.RequiresProtoCheck {
		t.Fatalf("fallback=%q proto=%v", result.FallbackReason, result.RequiresProtoCheck)
	}
}
