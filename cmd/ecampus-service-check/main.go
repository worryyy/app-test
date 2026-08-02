package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Milchstrassse/Ecampus-go/internal/ciimpact"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	serviceName := flag.String("service", "", "delivery service name")
	build := flag.Bool("build", false, "also build the service binary")
	output := flag.String("output", "", "binary output path")
	checkGenerated := flag.Bool("check-generated", false, "regenerate agent protobufs and verify a clean diff")
	manifestPath := flag.String("manifest", "scripts/ci/services.json", "service manifest path")
	flag.Parse()
	if *serviceName == "" {
		return fmt.Errorf("--service is required")
	}
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
	service, err := ciimpact.FindService(manifest, *serviceName)
	if err != nil {
		return err
	}
	packages, err := ciimpact.RuntimePackages(ctx, repoRoot, service)
	if err != nil {
		return fmt.Errorf("resolve %s packages: %w", service.Service, err)
	}
	if err := runCommand(ctx, repoRoot, "go", append([]string{"vet"}, packages...)...); err != nil {
		return err
	}
	if err := runCommand(ctx, repoRoot, "go", append([]string{"test"}, packages...)...); err != nil {
		return err
	}
	if *checkGenerated {
		if service.Service != "agentchat" {
			return fmt.Errorf("--check-generated is only valid for agentchat")
		}
		if err := runCommand(ctx, repoRoot, "make", "proto-agent"); err != nil {
			return err
		}
		if err := runCommand(ctx, repoRoot, "git", "diff", "--exit-code", "--", "proto/agent", "internal/agentchat/agentv1"); err != nil {
			return fmt.Errorf("agent protobuf generated code is not synchronized: %w", err)
		}
	}
	if *build {
		args := []string{"build"}
		if *output != "" {
			args = append(args, "-o", *output)
		}
		args = append(args, service.Entrypoint)
		if err := runCommand(ctx, repoRoot, "go", args...); err != nil {
			return err
		}
	}
	return nil
}

func repositoryRoot(ctx context.Context) (string, error) {
	command := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("resolve repository root: %w", err)
	}
	return filepath.Clean(string(bytes.TrimSpace(output))), nil
}

func runCommand(ctx context.Context, dir, name string, args ...string) error {
	command := exec.CommandContext(ctx, name, args...)
	command.Dir = dir
	command.Stdout, command.Stderr = os.Stdout, os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}
