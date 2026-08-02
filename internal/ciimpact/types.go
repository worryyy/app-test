package ciimpact

type Service struct {
	Service    string `json:"service"`
	Entrypoint string `json:"entrypoint"`
	Image      string `json:"image"`
	ConfigDir  string `json:"config_dir"`
	Port       int    `json:"port"`
}

type Manifest struct {
	Services []Service `json:"services"`
}

type Matrix struct {
	Include []Service `json:"include"`
}

type Result struct {
	ChangedFiles       []string `json:"changed_files"`
	TestMatrix         Matrix   `json:"test_matrix"`
	BuildMatrix        Matrix   `json:"build_matrix"`
	HasTests           bool     `json:"has_tests"`
	HasBuilds          bool     `json:"has_builds"`
	RequiresProtoCheck bool     `json:"requires_proto_check"`
	FallbackReason     string   `json:"fallback_reason"`
}

type Change struct {
	Status  string
	OldPath string
	Path    string
}

type Graph struct {
	Packages map[string]string
	Runtime  map[string]map[string]struct{}
	Tests    map[string]map[string]struct{}
}
