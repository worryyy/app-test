package ciimpact

import (
	"fmt"
	"path"
	"sort"
	"strings"
)

var fullBuildFiles = map[string]struct{}{
	"go.mod": {}, "go.sum": {}, "go.work": {}, "go.work.sum": {}, ".dockerignore": {},
	"build/Dockerfile.go-service": {}, "scripts/ci/services.json": {},
}

func FullResult(manifest Manifest, changed []string, reason string) Result {
	if changed == nil {
		changed = []string{}
	}
	requiresProtoCheck := false
	for _, file := range changed {
		if strings.HasPrefix(cleanPath(file), "proto/agent/") {
			requiresProtoCheck = true
			break
		}
	}
	services := append([]Service(nil), manifest.Services...)
	return Result{
		ChangedFiles: changed, TestMatrix: Matrix{Include: services}, BuildMatrix: Matrix{Include: services},
		HasTests: len(services) > 0, HasBuilds: len(services) > 0,
		RequiresProtoCheck: requiresProtoCheck, FallbackReason: reason,
	}
}

func Evaluate(manifest Manifest, changes []Change, graph Graph) Result {
	changed := changedPaths(changes)
	for _, change := range changes {
		if changeIsIgnorable(change) {
			continue
		}
		if strings.HasPrefix(change.Status, "D") {
			return FullResult(manifest, changed, "deleted file cannot be mapped safely: "+change.Path)
		}
		if strings.HasPrefix(change.Status, "R") || strings.HasPrefix(change.Status, "C") {
			return FullResult(manifest, changed, "renamed or copied file cannot be mapped safely: "+change.Path)
		}
		if change.Status != "A" && change.Status != "M" && change.Status != "T" {
			return FullResult(manifest, changed, "unsupported git change status: "+change.Status)
		}
	}

	tests := make(map[string]struct{})
	builds := make(map[string]struct{})
	requiresProtoCheck := false
	for _, change := range changes {
		file := cleanPath(change.Path)
		if ignorable(file) {
			continue
		}
		if _, ok := fullBuildFiles[file]; ok || strings.HasPrefix(file, "configs/ecampus/") || ciContractFile(file) {
			return FullResult(manifest, changed, "")
		}
		if strings.HasPrefix(file, "proto/agent/") {
			add(tests, "agentchat")
			add(builds, "agentchat")
			requiresProtoCheck = true
			continue
		}

		importPath, ok := graph.Packages[path.Dir(file)]
		if !ok {
			return FullResult(manifest, changed, "file cannot be mapped to a Go package: "+file)
		}
		matched := false
		for _, service := range manifest.Services {
			if _, ok := graph.Runtime[service.Service][importPath]; ok {
				add(tests, service.Service)
				if !strings.HasSuffix(file, "_test.go") {
					add(builds, service.Service)
				}
				matched = true
				continue
			}
			if _, ok := graph.Tests[service.Service][importPath]; ok {
				add(tests, service.Service)
				matched = true
			}
		}
		if !matched {
			return FullResult(manifest, changed, "package is outside every service dependency closure: "+importPath)
		}
	}

	result := Result{
		ChangedFiles:       changed,
		TestMatrix:         Matrix{Include: selectedServices(manifest, tests)},
		BuildMatrix:        Matrix{Include: selectedServices(manifest, builds)},
		RequiresProtoCheck: requiresProtoCheck,
	}
	result.HasTests = len(result.TestMatrix.Include) > 0
	result.HasBuilds = len(result.BuildMatrix.Include) > 0
	return result
}

func changeIsIgnorable(change Change) bool {
	if !ignorable(cleanPath(change.Path)) {
		return false
	}
	return change.OldPath == "" || ignorable(cleanPath(change.OldPath))
}

func changedPaths(changes []Change) []string {
	set := make(map[string]struct{})
	for _, change := range changes {
		if change.OldPath != "" {
			set[cleanPath(change.OldPath)] = struct{}{}
		}
		if change.Path != "" {
			set[cleanPath(change.Path)] = struct{}{}
		}
	}
	result := make([]string, 0, len(set))
	for file := range set {
		result = append(result, file)
	}
	sort.Strings(result)
	return result
}

func selectedServices(manifest Manifest, selected map[string]struct{}) []Service {
	services := make([]Service, 0, len(selected))
	for _, service := range manifest.Services {
		if _, ok := selected[service.Service]; ok {
			services = append(services, service)
		}
	}
	return services
}

func add(set map[string]struct{}, value string) { set[value] = struct{}{} }

func cleanPath(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	value = strings.TrimPrefix(value, "./")
	return path.Clean(value)
}

func ignorable(file string) bool {
	if file == ".github/workflows/deploy.yml" || strings.HasPrefix(file, "docs/") || strings.HasPrefix(file, ".github/ISSUE_TEMPLATE/") {
		return true
	}
	if strings.HasPrefix(file, "configs/ecampus-crm/") || strings.HasPrefix(file, "cmd/ecampus-crm/") || strings.HasPrefix(file, "internal/app/ecampuscrm/") {
		return true
	}
	if strings.HasPrefix(file, "cmd/ecampus/") || strings.HasPrefix(file, "deployments/ecampus/") || strings.HasPrefix(file, "deployments/ecampus-crm/") {
		return true
	}
	if path.Dir(file) == "internal/app/ecampus" {
		return true
	}
	ext := strings.ToLower(path.Ext(file))
	return ext == ".md" || ext == ".mdx" || file == "LICENSE"
}

func ciContractFile(file string) bool {
	return strings.HasPrefix(file, "internal/ciimpact/") || strings.HasPrefix(file, "cmd/ecampus-impact/") ||
		strings.HasPrefix(file, "cmd/ecampus-service-check/") || strings.HasPrefix(file, "scripts/ci/run-service-checks")
}

func ValidateGraph(manifest Manifest, graph Graph) error {
	for _, service := range manifest.Services {
		if len(graph.Runtime[service.Service]) == 0 || len(graph.Tests[service.Service]) == 0 {
			return fmt.Errorf("dependency graph is incomplete for %s", service.Service)
		}
	}
	return nil
}
