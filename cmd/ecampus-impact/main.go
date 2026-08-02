package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Milchstrassse/Ecampus-go/internal/ciimpact"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	base := flag.String("base", "", "base Git SHA")
	head := flag.String("head", "", "head Git SHA")
	changedFile := flag.String("changed-files", "", "git name-status input file")
	all := flag.Bool("all", false, "select all delivery services")
	manifestPath := flag.String("manifest", "scripts/ci/services.json", "service manifest path")
	flag.Parse()

	repoRoot, err := repositoryRoot(ctx)
	if err != nil {
		return err
	}
	manifestFile := *manifestPath
	if !filepath.IsAbs(manifestFile) {
		manifestFile = filepath.Join(repoRoot, manifestFile)
	}
	manifest, err := ciimpact.LoadManifest(manifestFile)
	if err != nil {
		return err
	}
	modeCount := 0
	if *all {
		modeCount++
	}
	if *changedFile != "" {
		modeCount++
	}
	if *base != "" || *head != "" {
		if *base == "" || *head == "" {
			return fmt.Errorf("--base and --head must be provided together")
		}
		modeCount++
	}
	if modeCount != 1 {
		return fmt.Errorf("provide exactly one of --all, --changed-files, or --base/--head")
	}

	if *all {
		return writeResult(ciimpact.FullResult(manifest, nil, ""))
	}
	changes, reason, err := readChanges(ctx, repoRoot, *changedFile, *base, *head)
	if err != nil {
		return writeResult(ciimpact.FullResult(manifest, nil, reason+": "+err.Error()))
	}
	graph, err := ciimpact.BuildGraph(ctx, repoRoot, manifest)
	if err != nil {
		paths := make([]string, 0, len(changes))
		for _, change := range changes {
			paths = append(paths, change.Path)
		}
		return writeResult(ciimpact.FullResult(manifest, paths, err.Error()))
	}
	return writeResult(ciimpact.Evaluate(manifest, changes, graph))
}

func readChanges(ctx context.Context, repoRoot, changedFile, base, head string) ([]ciimpact.Change, string, error) {
	if changedFile != "" {
		file, err := os.Open(changedFile)
		if err != nil {
			return nil, "read changed-files", err
		}
		defer func() { _ = file.Close() }()
		changes, err := ciimpact.ParseNameStatus(file)
		return changes, "parse changed-files", err
	}
	command := exec.CommandContext(ctx, "git", "diff", "--name-status", "--find-renames", base, head)
	command.Dir = repoRoot
	output, err := command.Output()
	if err != nil {
		return nil, "git diff failed", err
	}
	changes, err := ciimpact.ParseNameStatus(bytes.NewReader(output))
	return changes, "parse git diff", err
}

func repositoryRoot(ctx context.Context) (string, error) {
	command := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("resolve repository root: %w", err)
	}
	return filepath.Clean(string(bytes.TrimSpace(output))), nil
}

func writeResult(result ciimpact.Result) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}
