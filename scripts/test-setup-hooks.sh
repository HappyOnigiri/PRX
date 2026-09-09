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

repository="$root/repository"
mkdir -p "$repository/scripts/hooks"
cp "$script_directory/setup-hooks.sh" "$repository/scripts/setup-hooks.sh"
cp "$script_directory"/hooks/* "$repository/scripts/hooks/"

git_quiet -C "$repository" init --quiet --initial-branch=work
git_quiet -C "$repository" add -A
git_quiet -C "$repository" commit --quiet --no-verify -m 'initial'

output=$(cd "$repository" && scripts/setup-hooks.sh)
assert_contains "$output" "installed: $repository/.git/hooks/pre-commit"
for source in "$script_directory"/hooks/*; do
  destination="$repository/.git/hooks/$(basename "$source")"
  cmp "$source" "$destination"
  [ -x "$destination" ] || {
    echo "expected hook to be executable: $destination" >&2
    exit 1
  }
done

output=$(cd "$repository" && scripts/setup-hooks.sh)
assert_contains "$output" "unchanged: $repository/.git/hooks/pre-commit"

# worktree から実行しても、宛先は共通 Git ディレクトリの hooks/ である。
worktree="$root/worktree"
git_quiet -C "$repository" worktree add --quiet --detach "$worktree" HEAD
printf '%s\n' 'stale' > "$repository/.git/hooks/pre-commit"
output=$(cd "$worktree" && scripts/setup-hooks.sh)
assert_contains "$output" "installed: $repository/.git/hooks/pre-commit"
cmp "$script_directory/hooks/pre-commit" "$repository/.git/hooks/pre-commit"

echo 'setup-hooks tests passed'
