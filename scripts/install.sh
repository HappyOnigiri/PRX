#!/bin/bash
set -euo pipefail

# Release CI がこの値を埋め込み、インストーラーとバイナリを同じタグへ固定する。
release_version='@PRX_RELEASE_VERSION@'

fail() {
  echo "prx install: $*" >&2
  exit 1
}

# curl | bash の途中切断では配置処理を始めないよう、全体を読み込んでから呼ぶ。
main() {
  [[ "$release_version" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]] ||
    fail "use install.sh from a GitHub Release"
  [ "$(uname -s)" = Darwin ] && [ "$(uname -m)" = arm64 ] ||
    fail "macOS arm64 is required; use a native Apple Silicon terminal"
  [[ "${HOME:-}" = /* ]] || fail "HOME must be an absolute path"
  for tool in curl shasum mktemp install mv; do
    command -v "$tool" >/dev/null 2>&1 || fail "$tool is required"
  done

  local install_dir="$HOME/.local/bin" destination="$HOME/.local/bin/prx"
  local asset=prx-darwin-arm64
  local base_url="https://github.com/HappyOnigiri/PRX/releases/download/$release_version"
  local checksum checksum_name actual daemon_status='' initial_install=false
  [ ! -d "$destination" ] || fail "$destination is a directory"
  if [ ! -e "$destination" ]; then
    initial_install=true
  fi

  scratch=$(mktemp -d "${TMPDIR:-/tmp}/prx-install.XXXXXX")
  staged=''
  trap 'rm -rf "$scratch"; if [ -n "$staged" ]; then rm -f "$staged"; fi' EXIT
  trap 'exit 130' INT
  trap 'exit 143' TERM
  echo "Downloading prx $release_version..."
  curl --fail --silent --show-error --location "$base_url/$asset" --output "$scratch/$asset" ||
    fail "download failed; the installed binary was not changed"
  curl --fail --silent --show-error --location "$base_url/checksums.txt" --output "$scratch/checksums.txt" ||
    fail "checksum download failed; the installed binary was not changed"
  read -r checksum checksum_name < "$scratch/checksums.txt" || fail "invalid checksums.txt"
  [[ "$checksum" =~ ^[0-9a-fA-F]{64}$ ]] && [ "$checksum_name" = "$asset" ] ||
    fail "invalid checksum entry for $asset"
  # 検証する名前を固定し、checksums.txt の別行や相対パスを shasum に渡さない。
  (cd "$scratch" && printf '%s  %s\n' "$checksum" "$asset" | shasum -a 256 -c -) ||
    fail "checksum verification failed; the installed binary was not changed"
  chmod 0755 "$scratch/$asset"
  actual=$("$scratch/$asset" --version) || fail "the downloaded binary could not run"
  [ "$actual" = "prx version ${release_version#v}" ] || fail "unexpected binary version: $actual"

  # 同じファイルシステム上で組み立ててから置き換え、途中で失敗しても既存バイナリを壊さない。
  install -d "$install_dir"
  staged=$(mktemp "$install_dir/.prx-install.XXXXXX")
  install -m 0755 "$scratch/$asset" "$staged"
  mv -f "$staged" "$destination"
  staged=''
  echo "Installed prx $release_version to $destination"

  echo 'To use prx in this terminal, run:'
  # 利用者が実行するコマンドを展開せず表示する。
  # shellcheck disable=SC2016
  echo '  export PATH="$HOME/.local/bin:$PATH"'
  echo 'Add that line to your shell configuration (for example, ~/.zshrc) for new terminals.'
  if [ "$initial_install" = true ]; then
    daemon_status=$(PATH="$install_dir:$PATH" "$destination" daemon --json 2>/dev/null) || daemon_status=''
  fi
  if [[ "$daemon_status" == *'"installed":false'* ]]; then
    echo 'To start PRX at login, run:'
    echo '  prx daemon install'
    echo 'Then run prx open to open the server in your browser.'
  else
    echo 'Then run prx serve to start the server at http://127.0.0.1:7331.'
  fi
}

main "$@"
