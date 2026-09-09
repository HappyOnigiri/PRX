#!/bin/bash
set -euo pipefail

# hook から呼ばれると GIT_DIR などが環境に残り、一時リポジトリへの操作が
# 呼び出し元のリポジトリへ向いてしまうので、Git の環境変数を落としてから始める。
unset GIT_DIR GIT_WORK_TREE GIT_INDEX_FILE GIT_COMMON_DIR GIT_PREFIX \
  GIT_OBJECT_DIRECTORY GIT_ALTERNATE_OBJECT_DIRECTORIES

script_directory=$(cd "$(dirname "$0")" && pwd)
root=$(mktemp -d "${TMPDIR:-/tmp}/prx-setup-hooks-test.XXXXXX")
trap 'rm -rf "$root"' EXIT
# 宛先の表示は git が解決した実パスなので、期待値もシンボリックリンクを解いた形にそろえる。
root=$(cd "$root" && pwd -P)

git_quiet() {
  git -c user.name=prx -c user.email=prx@example.com -c commit.gpgsign=false "$@"
}

assert_contains() {
  case "$1" in
    *"$2"*) ;;
    *)
      echo "expected output to contain: $2" >&2
      printf '%s\n' "$1" >&2
      exit 1
      ;;
  esac
}

assert_executable() {
  [ -x "$1" ] || {
    echo "expected hook to be executable: $1" >&2
    exit 1
  }
}

repository="$root/repository"
hook="$repository/.git/hooks/pre-commit"
mkdir -p "$repository/scripts/hooks"
cp "$script_directory/setup-hooks.sh" "$repository/scripts/setup-hooks.sh"
cp "$script_directory"/hooks/* "$repository/scripts/hooks/"

git_quiet -C "$repository" init --quiet --initial-branch=work
git_quiet -C "$repository" add -A
git_quiet -C "$repository" commit --quiet --no-verify -m 'initial'

# 宛先が空なら、確認なしに複製して実行権限を付ける。
output=$(cd "$repository" && scripts/setup-hooks.sh < /dev/null)
assert_contains "$output" "installed: $hook"
for source in "$script_directory"/hooks/*; do
  destination="$repository/.git/hooks/$(basename "$source")"
  cmp "$source" "$destination"
  assert_executable "$destination"
done

output=$(cd "$repository" && scripts/setup-hooks.sh < /dev/null)
assert_contains "$output" "unchanged: $hook"

# 内容が同じでも実行権限が落ちていれば入れ直す。dispatcher は実行できない hook を無視する。
chmod 644 "$hook"
output=$(cd "$repository" && scripts/setup-hooks.sh < /dev/null)
assert_contains "$output" "fixed mode: $hook"
assert_executable "$hook"

# 差分があるときは diff を見せ、答えが得られなければ既存の hook を残して失敗する。
printf '%s\n' 'stale' > "$hook"
status=0
output=$(cd "$repository" && scripts/setup-hooks.sh < /dev/null 2>&1) || status=$?
[ "$status" -ne 0 ] || {
  echo 'expected setup-hooks.sh to fail when an existing hook is kept' >&2
  exit 1
}
assert_contains "$output" "differs: $hook"
assert_contains "$output" '-stale'
assert_contains "$output" "kept: $hook"
[ "$(cat "$hook")" = 'stale' ] || {
  echo 'expected the existing hook to be left untouched' >&2
  exit 1
}

# 答えが no でも残す。
output=$(cd "$repository" && printf '%s\n' n | scripts/setup-hooks.sh 2>&1) || status=$?
assert_contains "$output" "kept: $hook"
[ "$(cat "$hook")" = 'stale' ] || {
  echo 'expected a declined overwrite to leave the hook untouched' >&2
  exit 1
}

# 答えが yes なら上書きする。
output=$(cd "$repository" && printf '%s\n' y | scripts/setup-hooks.sh)
assert_contains "$output" "installed: $hook"
cmp "$script_directory/hooks/pre-commit" "$hook"

# --yes は確認を省いて上書きする。
printf '%s\n' 'stale' > "$hook"
output=$(cd "$repository" && scripts/setup-hooks.sh --yes < /dev/null)
assert_contains "$output" "installed: $hook"
cmp "$script_directory/hooks/pre-commit" "$hook"

# repository の core.hooksPath があると hooks/ は読まれないので、置かずに失敗する。
git_quiet -C "$repository" config --local core.hooksPath .githooks
status=0
output=$(cd "$repository" && scripts/setup-hooks.sh --yes < /dev/null 2>&1) || status=$?
[ "$status" -ne 0 ] || {
  echo 'expected setup-hooks.sh to fail while core.hooksPath is set for the repository' >&2
  exit 1
}
assert_contains "$output" 'core.hooksPath is set for this repository'
git_quiet -C "$repository" config --local --unset core.hooksPath

# worktree から実行しても、宛先は共通 Git ディレクトリの hooks/ である。
worktree="$root/worktree"
git_quiet -C "$repository" worktree add --quiet --detach "$worktree" HEAD
printf '%s\n' 'stale' > "$hook"
output=$(cd "$worktree" && scripts/setup-hooks.sh --yes < /dev/null)
assert_contains "$output" "installed: $hook"
cmp "$script_directory/hooks/pre-commit" "$hook"

echo 'setup-hooks tests passed'
