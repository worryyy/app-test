package ciimpact

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type listedModule struct {
	Main bool
}

type listedPackage struct {
	ImportPath string
	Dir        string
	Module     *listedModule
}

func BuildGraph(ctx context.Context, repoRoot string, manifest Manifest) (Graph, error) {
	all, err := goList(ctx, repoRoot, "-json", "./...")
	if err != nil {
		return Graph{}, fmt.Errorf("go list packages: %w", err)
	}
	graph := Graph{
		Packages: make(map[string]string),
		Runtime:  make(map[string]map[string]struct{}, len(manifest.Services)),
		Tests:    make(map[string]map[string]struct{}, len(manifest.Services)),
	}
	for _, pkg := range all {
		if !isMainModule(pkg) || pkg.Dir == "" || pkg.ImportPath == "" {
			continue
		}
		dir, err := filepath.Rel(repoRoot, pkg.Dir)
		if err != nil || strings.HasPrefix(dir, "..") {
			continue
		}
		graph.Packages[cleanPath(dir)] = pkg.ImportPath
	}

	for _, service := range manifest.Services {
		runtimePackages, err := goList(ctx, repoRoot, "-deps", "-json", service.Entrypoint)
		if err != nil {
			return Graph{}, fmt.Errorf("resolve runtime dependencies for %s: %w", service.Service, err)
		}
		runtime := localImports(runtimePackages)
		if len(runtime) == 0 {
			return Graph{}, fmt.Errorf("resolve runtime dependencies for %s: no main-module packages", service.Service)
		}
		graph.Runtime[service.Service] = runtime

		imports := make([]string, 0, len(runtime))
		for importPath := range runtime {
			imports = append(imports, importPath)
		}
		sort.Strings(imports)
		args := append([]string{"-deps", "-test", "-json"}, imports...)
		testPackages, err := goList(ctx, repoRoot, args...)
		if err != nil {
			return Graph{}, fmt.Errorf("resolve test dependencies for %s: %w", service.Service, err)
		}
		graph.Tests[service.Service] = localImports(testPackages)
	}
	return graph, nil
}

func RuntimePackages(ctx context.Context, repoRoot string, service Service) ([]string, error) {
	packages, err := goList(ctx, repoRoot, "-deps", "-json", service.Entrypoint)
	if err != nil {
		return nil, err
	}
	imports := localImports(packages)
	result := make([]string, 0, len(imports))
	for importPath := range imports {
		result = append(result, importPath)
	}
	sort.Strings(result)
	return result, nil
}

func goList(ctx context.Context, repoRoot string, args ...string) ([]listedPackage, error) {
	command := exec.CommandContext(ctx, "go", append([]string{"list"}, args...)...)
	command.Dir = repoRoot
	var stderr bytes.Buffer
	command.Stderr = &stderr
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("%v: %s: %w", command.Args, strings.TrimSpace(stderr.String()), err)
	}
	decoder := json.NewDecoder(bytes.NewReader(output))
	packages := make([]listedPackage, 0)
	for decoder.More() {
		var pkg listedPackage
		if err := decoder.Decode(&pkg); err != nil {
			return nil, fmt.Errorf("decode go list output: %w", err)
		}
		packages = append(packages, pkg)
	}
	return packages, nil
}

func localImports(packages []listedPackage) map[string]struct{} {
	imports := make(map[string]struct{})
	for _, pkg := range packages {
		if isMainModule(pkg) && pkg.ImportPath != "" && !strings.HasSuffix(pkg.ImportPath, ".test") {
			imports[pkg.ImportPath] = struct{}{}
		}
	}
	return imports
}

func isMainModule(pkg listedPackage) bool {
	return pkg.Module != nil && pkg.Module.Main
}
