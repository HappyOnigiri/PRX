// Package prx exposes the version of the PRX build.
package prx

import (
	_ "embed"
	"encoding/json"
)

//go:embed package.json
var packageManifest []byte

// releaseVersion is populated by the supported production build. An unstamped
// build identifies the release it is based on without claiming to be that release.
var releaseVersion string

// Version returns the release version for a production build and the base
// release with a -dev suffix for an unstamped development build.
func Version() string {
	if releaseVersion != "" {
		return releaseVersion
	}
	var manifest struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(packageManifest, &manifest); err != nil {
		panic("decode embedded package.json: " + err.Error())
	}
	if manifest.Version == "" {
		panic("embedded package.json has no version")
	}
	return manifest.Version + "-dev"
}
