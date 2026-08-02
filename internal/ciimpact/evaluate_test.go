package ciimpact

import (
	"strings"
	"testing"
)

func TestParseNameStatus(t *testing.T) {
	changes, err := ParseNameStatus(strings.NewReader("M\tinternal/topic/service.go\nR100\told.go\tnew.go\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 2 || changes[1].OldPath != "old.go" || changes[1].Path != "new.go" {
		t.Fatalf("unexpected changes: %#v", changes)
	}
}

func TestEvaluateRuntimeAndTestOnlyChanges(t *testing.T) {
	manifest, graph := fixture()
	result := Evaluate(manifest, []Change{{Status: "M", Path: "internal/topic/service.go"}}, graph)
	assertServices(t, result.TestMatrix.Include, "topic", "user")
	assertServices(t, result.BuildMatrix.Include, "topic", "user")

	result = Evaluate(manifest, []Change{{Status: "M", Path: "internal/testutil/helper.go"}}, graph)
	assertServices(t, result.TestMatrix.Include, "topic")
	assertServices(t, result.BuildMatrix.Include)

	result = Evaluate(manifest, []Change{{Status: "M", Path: "internal/topic/service_test.go"}}, graph)
	assertServices(t, result.TestMatrix.Include, "topic", "user")
	assertServices(t, result.BuildMatrix.Include)
}

func TestEvaluateRulesAndFallbacks(t *testing.T) {
	manifest, graph := fixture()
	tests := []struct {
		name       string
		change     Change
		wantTest   []string
		wantBuild  []string
		fallback   bool
		protoCheck bool
	}{
		{name: "shared config", change: Change{Status: "M", Path: "configs/ecampus/config-test.yaml"}, wantTest: []string{"agentchat", "topic", "user"}, wantBuild: []string{"agentchat", "topic", "user"}},
		{name: "go mod", change: Change{Status: "M", Path: "go.mod"}, wantTest: []string{"agentchat", "topic", "user"}, wantBuild: []string{"agentchat", "topic", "user"}},
		{name: "common dockerfile", change: Change{Status: "M", Path: "build/Dockerfile.go-service"}, wantTest: []string{"agentchat", "topic", "user"}, wantBuild: []string{"agentchat", "topic", "user"}},
		{name: "service manifest", change: Change{Status: "M", Path: "scripts/ci/services.json"}, wantTest: []string{"agentchat", "topic", "user"}, wantBuild: []string{"agentchat", "topic", "user"}},
		{name: "proto", change: Change{Status: "M", Path: "proto/agent/v1/agent.proto"}, wantTest: []string{"agentchat"}, wantBuild: []string{"agentchat"}, protoCheck: true},
		{name: "docs", change: Change{Status: "M", Path: "docs/ci.md"}},
		{name: "deleted docs", change: Change{Status: "D", Path: "docs/old.md"}},
		{name: "renamed docs", change: Change{Status: "R100", OldPath: "docs/old.md", Path: "docs/new.md"}},
		{name: "crm", change: Change{Status: "M", Path: "internal/app/ecampuscrm/app.go"}},
		{name: "crm config", change: Change{Status: "M", Path: "configs/ecampus-crm/config-prod.yaml"}},
		{name: "aggregate", change: Change{Status: "M", Path: "internal/app/ecampus/app.go"}},
		{name: "aggregate command", change: Change{Status: "M", Path: "cmd/ecampus/main.go"}},
		{name: "legacy deployment", change: Change{Status: "M", Path: "deployments/ecampus/Dockerfile"}},
		{name: "deleted", change: Change{Status: "D", Path: "internal/topic/service.go"}, wantTest: []string{"agentchat", "topic", "user"}, wantBuild: []string{"agentchat", "topic", "user"}, fallback: true},
		{name: "rename", change: Change{Status: "R100", OldPath: "a.go", Path: "b.go"}, wantTest: []string{"agentchat", "topic", "user"}, wantBuild: []string{"agentchat", "topic", "user"}, fallback: true},
		{name: "unknown", change: Change{Status: "M", Path: "unknown/file.go"}, wantTest: []string{"agentchat", "topic", "user"}, wantBuild: []string{"agentchat", "topic", "user"}, fallback: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := Evaluate(manifest, []Change{test.change}, graph)
			assertServices(t, result.TestMatrix.Include, test.wantTest...)
			assertServices(t, result.BuildMatrix.Include, test.wantBuild...)
			if (result.FallbackReason != "") != test.fallback {
				t.Fatalf("fallback_reason=%q", result.FallbackReason)
			}
			if result.RequiresProtoCheck != test.protoCheck {
				t.Fatalf("requires_proto_check=%v", result.RequiresProtoCheck)
			}
		})
	}
}

func fixture() (Manifest, Graph) {
	manifest := Manifest{Services: []Service{
		{Service: "agentchat", Entrypoint: "./cmd/ecampus-agentchat", Image: "ecampus-agentchat", ConfigDir: "configs/ecampus", Port: 8080},
		{Service: "topic", Entrypoint: "./cmd/ecampus-topic", Image: "ecampus-topic", ConfigDir: "configs/ecampus", Port: 8080},
		{Service: "user", Entrypoint: "./cmd/ecampus-user", Image: "ecampus-user", ConfigDir: "configs/ecampus", Port: 8080},
	}}
	graph := Graph{
		Packages: map[string]string{"internal/topic": "module/topic", "internal/testutil": "module/testutil"},
		Runtime: map[string]map[string]struct{}{
			"agentchat": {"module/agentchat": {}},
			"topic":     {"module/topic": {}},
			"user":      {"module/topic": {}, "module/user": {}},
		},
		Tests: map[string]map[string]struct{}{
			"agentchat": {"module/agentchat": {}},
			"topic":     {"module/topic": {}, "module/testutil": {}},
			"user":      {"module/topic": {}, "module/user": {}},
		},
	}
	return manifest, graph
}

func assertServices(t *testing.T, services []Service, want ...string) {
	t.Helper()
	if len(services) != len(want) {
		t.Fatalf("services=%v want=%v", serviceNames(services), want)
	}
	for index := range want {
		if services[index].Service != want[index] {
			t.Fatalf("services=%v want=%v", serviceNames(services), want)
		}
	}
}

func serviceNames(services []Service) []string {
	names := make([]string, len(services))
	for index, service := range services {
		names[index] = service.Service
	}
	return names
}
