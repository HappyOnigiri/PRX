#!/bin/bash
set -euo pipefail

script_directory=$(cd "$(dirname "$0")" && pwd)
root=$(mktemp -d "${TMPDIR:-/tmp}/prx-uninstall-test.XXXXXX")
trap 'rm -rf "$root"' EXIT

tools="$root/tools"
mkdir -p "$tools"
cat > "$tools/uname" <<'EOF'
#!/bin/bash
case "${1-}" in
  -s) echo Darwin ;;
  *) exit 1 ;;
esac
EOF
chmod 0755 "$tools/uname"

write_prx_stub() {
  local path=$1
  cat > "$path" <<'EOF'
#!/bin/bash
set -euo pipefail
marker=unset
[ -n "${PRX_RUN_DIR+x}" ] && marker=set
printf '%s:%s\n' "$*" "$marker" >> "$PRX_UNINSTALL_TEST_LOG"
case "${1-}:${2-}" in
  daemon:stop)
    if [ "${PRX_UNINSTALL_TEST_STOP:-ok}" = fail ]; then exit 1; fi
    ;;
  daemon:uninstall)
    if [ "${PRX_UNINSTALL_TEST_UNINSTALL:-ok}" = fail ]; then exit 1; fi
    ;;
  --json:daemon)
    [ "${PRX_UNINSTALL_TEST_STATUS:-}" != '' ] || exit 1
    printf '%s\n' "$PRX_UNINSTALL_TEST_STATUS"
    ;;
  *)
    exit 1
    ;;
esac
EOF
  chmod 0755 "$path"
}

assert_file() {
  [ -e "$1" ] || {
    echo "expected file: $1" >&2
    exit 1
  }
}

assert_missing() {
  [ ! -e "$1" ] || {
    echo "expected path to be missing: $1" >&2
    exit 1
  }
}

assert_contains() {
  grep -Fq -- "$2" "$1" || {
    echo "expected $1 to contain: $2" >&2
    sed -n '1,240p' "$1" >&2
    exit 1
  }
}

assert_not_contains() {
  [ -e "$1" ] || return 0
  if grep -Fq -- "$2" "$1"; then
    echo "expected $1 not to contain: $2" >&2
    sed -n '1,240p' "$1" >&2
    exit 1
  fi
}

run_script() {
  local name=$1 home=$2 path=$3 output=$4 error=$5
  shift 5
  HOME="$home" PATH="$tools:$path:/usr/bin:/bin" PRX_RUN_DIR="$home/custom-run" \
    PRX_UNINSTALL_TEST_LOG="$home/events" \
    PRX_UNINSTALL_TEST_STOP="${PRX_UNINSTALL_TEST_STOP:-ok}" \
    PRX_UNINSTALL_TEST_UNINSTALL="${PRX_UNINSTALL_TEST_UNINSTALL:-ok}" \
    PRX_UNINSTALL_TEST_STATUS="${PRX_UNINSTALL_TEST_STATUS:-}" \
    bash "$script_directory/uninstall.sh" "$@" \
    > "$output" 2> "$error"
}

new_home() {
  local name=$1 home
  home="$root/$name/home"
  mkdir -p "$home/Library/Application Support/prx" "$home/Library/Logs/prx"
  printf 'keep\n' > "$home/Library/Application Support/prx/config.yaml"
  printf 'keep\n' > "$home/Library/Logs/prx/serve.log"
  echo "$home"
}

test_success_removes_standard_binary_and_keeps_data() {
  local home output error path="$root/empty" home_bin
  home=$(new_home standard)
  home_bin="$home/.local/bin"
  mkdir -p "$home_bin" "$path"
  write_prx_stub "$home_bin/prx"
  output="$home/output"
  error="$home/error"
  PRX_UNINSTALL_TEST_STATUS='{"running":false,"installed":false}' \
    PRX_UNINSTALL_TEST_STOP=ok PRX_UNINSTALL_TEST_UNINSTALL=ok \
    run_script standard "$home" "$path" "$output" "$error" --yes
  assert_missing "$home_bin/prx"
  assert_file "$home/Library/Application Support/prx/config.yaml"
  assert_file "$home/Library/Logs/prx/serve.log"
  assert_contains "$home/events" 'daemon stop:unset'
  assert_contains "$home/events" 'daemon uninstall:unset'
  assert_contains "$home/events" '--json daemon:unset'
  assert_contains "$output" 'Removed'
  assert_contains "$output" 'rm -rf'
}

test_path_binary_is_kept() {
  local home output error path="$root/path-bin"
  home=$(new_home path)
  mkdir -p "$path"
  write_prx_stub "$path/prx"
  output="$home/output"
  error="$home/error"
  PRX_UNINSTALL_TEST_STATUS='{"running":false,"installed":false}' \
    run_script path "$home" "$path" "$output" "$error" --yes
  assert_file "$path/prx"
  assert_contains "$output" 'outside the standard installation path'
}

test_symlink_removes_only_link() {
  local home output error path="$root/symlink-bin" target
  home=$(new_home symlink)
  mkdir -p "$home/.local/bin" "$path"
  target="$home/elsewhere/prx"
  mkdir -p "$(dirname "$target")"
  write_prx_stub "$target"
  ln -s "$target" "$home/.local/bin/prx"
  output="$home/output"
  error="$home/error"
  PRX_UNINSTALL_TEST_STATUS='{"running":false,"installed":false}' \
    run_script symlink "$home" "$path" "$output" "$error" --yes
  assert_missing "$home/.local/bin/prx"
  assert_file "$target"
  assert_contains "$output" 'target was kept'
  assert_contains "$output" "$target"
}

test_failures_keep_standard_binary() {
  local mode home output error path="$root/failure-bin"
  for mode in stop uninstall status running; do
    home=$(new_home "failure-$mode")
    mkdir -p "$home/.local/bin" "$path"
    write_prx_stub "$home/.local/bin/prx"
    output="$home/output"
    error="$home/error"
    PRX_UNINSTALL_TEST_STOP=ok PRX_UNINSTALL_TEST_UNINSTALL=ok \
      PRX_UNINSTALL_TEST_STATUS='{"running":false,"installed":false}'
    case "$mode" in
      stop) PRX_UNINSTALL_TEST_STOP=fail ;;
      uninstall) PRX_UNINSTALL_TEST_UNINSTALL=fail ;;
      status) PRX_UNINSTALL_TEST_STATUS='' ;;
      running) PRX_UNINSTALL_TEST_STATUS='{"running":true,"installed":false}' ;;
    esac
    if PRX_UNINSTALL_TEST_STOP="$PRX_UNINSTALL_TEST_STOP" \
      PRX_UNINSTALL_TEST_UNINSTALL="$PRX_UNINSTALL_TEST_UNINSTALL" \
      PRX_UNINSTALL_TEST_STATUS="$PRX_UNINSTALL_TEST_STATUS" \
      run_script "$mode" "$home" "$path" "$output" "$error" --yes; then
      echo "expected $mode to fail" >&2
      exit 1
    fi
    assert_file "$home/.local/bin/prx"
  done
}

test_preflight_rejects_directory_and_missing_cli() {
  local home output error path="$root/no-cli"
  home=$(new_home directory)
  mkdir -p "$home/.local/bin/prx" "$path"
  output="$home/output"
  error="$home/error"
  if run_script directory "$home" "$path" "$output" "$error" --yes; then
    echo 'directory target unexpectedly succeeded' >&2
    exit 1
  fi
  assert_file "$home/.local/bin/prx"

  home=$(new_home missing)
  output="$home/output"
  error="$home/error"
  if run_script missing "$home" "$path" "$output" "$error" --yes; then
    echo 'missing CLI unexpectedly succeeded' >&2
    exit 1
  fi
  assert_not_contains "$home/events" 'daemon stop'
}

test_noninteractive_confirmation_is_safe() {
  local home output error path="$root/confirm-bin"
  home=$(new_home confirmation)
  mkdir -p "$home/.local/bin" "$path"
  write_prx_stub "$home/.local/bin/prx"
  output="$home/output"
  error="$home/error"
  if PRX_UNINSTALL_TEST_STATUS='{"running":false,"installed":false}' \
    run_script confirmation "$home" "$path" "$output" "$error"; then
    echo 'noninteractive confirmation unexpectedly succeeded' >&2
    exit 1
  fi
  assert_file "$home/.local/bin/prx"
  assert_not_contains "$home/events" 'daemon stop'
}

bash -n "$script_directory/uninstall.sh"
test_success_removes_standard_binary_and_keeps_data
test_path_binary_is_kept
test_symlink_removes_only_link
test_failures_keep_standard_binary
test_preflight_rejects_directory_and_missing_cli
test_noninteractive_confirmation_is_safe
echo 'uninstall tests passed'
