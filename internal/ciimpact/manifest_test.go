package ciimpact

import "testing"

func TestDeliveryManifestContainsExactlyThirteenServices(t *testing.T) {
	manifest, err := LoadManifest("../../scripts/ci/services.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Services) != 13 {
		t.Fatalf("service count=%d want=13", len(manifest.Services))
	}
	expected := []string{"academic", "agentchat", "chat", "comment", "file", "marketplace", "moderation", "notification", "reservation", "school", "theme", "topic", "user"}
	for index, service := range manifest.Services {
		if service.Service != expected[index] {
			t.Fatalf("service[%d]=%q want=%q", index, service.Service, expected[index])
		}
	}
}
