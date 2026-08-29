package prx

import (
	"encoding/json"
	"testing"
)

func TestVersionUsesManifestForDevelopmentBuild(t *testing.T) {
	previous := releaseVersion
	releaseVersion = ""
	t.Cleanup(func() { releaseVersion = previous })

	var manifest struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(packageManifest, &manifest); err != nil {
		t.Fatal(err)
	}
	if got, want := Version(), manifest.Version+"-dev"; got != want {
		t.Fatalf("Version() = %q, want %q", got, want)
	}
}

func TestVersionUsesInjectedReleaseVersion(t *testing.T) {
	previous := releaseVersion
	releaseVersion = "1.2.3"
	t.Cleanup(func() { releaseVersion = previous })

	if got, want := Version(), "1.2.3"; got != want {
		t.Fatalf("Version() = %q, want %q", got, want)
	}
}
