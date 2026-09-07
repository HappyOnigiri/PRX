// Package prx は PRX ビルドのバージョンを公開する。
package prx

import (
	_ "embed"
	"encoding/json"
)

//go:embed package.json
var packageManifest []byte

// releaseVersion は正式なプロダクションビルドで埋め込まれる。埋め込みのないビルドは、
// そのリリースそのものだとは主張せず、基にしたリリースを示す。
var releaseVersion string

// Version は、プロダクションビルドではリリースバージョンを返し、埋め込みのない
// 開発ビルドでは基にしたリリースに -dev を付けた文字列を返す。
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
