#!/bin/bash
set -euo pipefail

root=$(mktemp -d "${TMPDIR:-/tmp}/prx-install-test.XXXXXX")
trap 'rm -rf "$root"' EXIT

asset="$root/prx-darwin-arm64"
cat > "$asset" <<'EOF'
#!/bin/bash
case "${1-}:${2-}" in
  --version:)
    echo 'prx version 0.0.0'
    ;;
  daemon:--json)
    if [ "$PRX_INSTALL_TEST_DAEMON_STATUS" = fail ]; then
      exit 1
    fi
    printf '%s\n' "$PRX_INSTALL_TEST_DAEMON_STATUS"
    ;;
  *)
    exit 1
    ;;
esac
EOF
chmod 0755 "$asset"

tool_directory="$root/tools"
mkdir -p "$tool_directory"
cat > "$tool_directory/uname" <<'EOF'
#!/bin/bash
case "${1-}" in
  -s) echo Darwin ;;
  -m) echo arm64 ;;
  *) exit 1 ;;
esac
EOF
cat > "$tool_directory/curl" <<'EOF'
#!/bin/bash
set -euo pipefail

output=''
url=''
while [ "$#" -gt 0 ]; do
  case "$1" in
    --output)
      output=$2
      shift 2
      ;;
    *)
      url=$1
      shift
      ;;
  esac
done
case "$url" in
  */prx-darwin-arm64)
    cp "$PRX_INSTALL_TEST_ASSET" "$output"
    ;;
  */checksums.txt)
    checksum=$(shasum -a 256 "$PRX_INSTALL_TEST_ASSET")
    printf '%s  prx-darwin-arm64\n' "${checksum%% *}" > "$output"
    ;;
  *)
    exit 1
    ;;
esac
EOF
chmod 0755 "$tool_directory/uname" "$tool_directory/curl"

installer="$root/install.sh"
sed "s/@PRX_RELEASE_VERSION@/v0.0.0/g" scripts/install.sh > "$installer"

assert_contains() {
  local file=$1 text=$2
  grep -Fq "$text" "$file" || {
    echo "expected output to contain: $text" >&2
    cat "$file" >&2
    exit 1
  }
}

assert_not_contains() {
  local file=$1 text=$2
  if grep -Fq "$text" "$file"; then
    echo "expected output not to contain: $text" >&2
    cat "$file" >&2
    exit 1
  fi
}

run_case() {
  local name=$1 existing_binary=$2 daemon_status=$3 expected_startup=$4
  local home="$root/$name/home" output="$root/$name/output"
  mkdir -p "$home"
  if [ "$existing_binary" = true ]; then
    mkdir -p "$home/.local/bin"
    cp "$asset" "$home/.local/bin/prx"
  fi

  HOME="$home" PATH="$tool_directory:$PATH" PRX_INSTALL_TEST_ASSET="$asset" \
    PRX_INSTALL_TEST_DAEMON_STATUS="$daemon_status" bash "$installer" > "$output"
  assert_contains "$output" 'To use prx in this terminal, run:'
  if [ "$expected_startup" = daemon ]; then
    assert_contains "$output" 'To start PRX at login, run:'
    assert_contains "$output" '  prx daemon install'
    assert_contains "$output" 'Then run prx open to open the server in your browser.'
    assert_not_contains "$output" 'Then run prx serve to start the server at http://127.0.0.1:7331.'
    return
  fi
  assert_contains "$output" 'Then run prx serve to start the server at http://127.0.0.1:7331.'
  assert_not_contains "$output" 'To start PRX at login, run:'
}

run_case initial-not-installed false '{"supported":true,"installed":false}' daemon
run_case initial-installed false '{"supported":true,"installed":true}' serve
run_case initial-status-failure false fail serve
run_case update-not-installed true '{"supported":true,"installed":false}' serve
