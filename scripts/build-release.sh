#!/bin/bash
set -euo pipefail

# 開発ビルドの -dev 付きバージョンは使わず、CI がリリースブランチから決めたタグだけを受け取る。
release_version=${RELEASE_VERSION:-}
[[ "$release_version" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]] || {
  echo 'release: RELEASE_VERSION must be vX.Y.Z' >&2
  exit 1
}
go_command=${GO:-go}
release_dir=${RELEASE_DIR:-artifacts/release}
script_directory=$(cd "$(dirname "$0")" && pwd)
scratch=$(mktemp -d "${TMPDIR:-/tmp}/prx-release.XXXXXX")
trap 'rm -rf "$scratch"' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

# WebUI は呼び出し側の web-build が生成済みで、この Go ビルドが internal/webui へ埋め込む。
# CLI の表示は v なしなので、タグから接頭辞を落として埋め込む。
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 "$go_command" build -trimpath \
  -ldflags "-s -w -X github.com/HappyOnigiri/PRX.releaseVersion=${release_version#v}" \
  -o "$scratch/prx-darwin-arm64" ./cmd/prx
sed "s/@PRX_RELEASE_VERSION@/$release_version/g" "$script_directory/install.sh" > "$scratch/install.sh"
cp "$script_directory/uninstall.sh" "$scratch/uninstall.sh"
(cd "$scratch" && shasum -a 256 prx-darwin-arm64 > checksums.txt)
mkdir -p "$release_dir"
cp "$scratch/prx-darwin-arm64" "$scratch/install.sh" "$scratch/uninstall.sh" "$scratch/checksums.txt" "$release_dir/"
