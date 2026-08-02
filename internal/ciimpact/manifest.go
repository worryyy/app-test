package ciimpact

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func LoadManifest(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("read service manifest: %w", err)
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode service manifest: %w", err)
	}
	if err := ValidateManifest(manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func ValidateManifest(manifest Manifest) error {
	if len(manifest.Services) == 0 {
		return fmt.Errorf("service manifest is empty")
	}
	seen := make(map[string]struct{}, len(manifest.Services))
	for _, service := range manifest.Services {
		name := strings.TrimSpace(service.Service)
		if name == "" || service.Entrypoint == "" || service.Image == "" || service.ConfigDir == "" || service.Port <= 0 {
			return fmt.Errorf("service manifest contains an incomplete entry for %q", name)
		}
		if _, exists := seen[name]; exists {
			return fmt.Errorf("service manifest contains duplicate service %q", name)
		}
		seen[name] = struct{}{}
	}
	return nil
}

func FindService(manifest Manifest, name string) (Service, error) {
	for _, service := range manifest.Services {
		if service.Service == name {
			return service, nil
		}
	}
	return Service{}, fmt.Errorf("service %q is not present in the manifest", name)
}
