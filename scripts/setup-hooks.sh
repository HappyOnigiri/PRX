#!/bin/bash
set -euo pipefail

# scripts/hooks/ の内容をこのリポジトリの hooks ディレクトリへ複製する。
# 宛先は共通 Git ディレクトリ (--git-common-dir) なので、worktree から実行しても
# 全 worktree が共有する 1 か所へ入る。
# core.hooksPath は設定しない。user レベルの dispatcher を覆い隠さないためである。
root=$(git rev-parse --show-toplevel)
hooks_dir=$(git rev-parse --path-format=absolute --git-common-dir)/hooks

# repository か worktree の core.hooksPath があると Git はこの hooks ディレクトリを見ないので、
# 動かない hook を置かずに中断する。global 設定は dispatcher なので許容する。
scoped_hooks_path=$(git config --show-scope --get core.hooksPath 2>/dev/null || true)
case "$scoped_hooks_path" in
  local*|worktree*)
    echo "Error: core.hooksPath is set for this repository ($scoped_hooks_path)." >&2
    echo "Hooks in $hooks_dir would be ignored; unset it before running make setup-hooks." >&2
    exit 1
    ;;
esac

assume_yes=${SETUP_HOOKS_ASSUME_YES:-0}
if [ "${1-}" = '--yes' ]; then
  assume_yes=1
fi

mkdir -p "$hooks_dir"

kept=0

# 既存の hook を書き換えるときだけ、差分を見せて許可を求める。
# 答えが得られないとき (パイプ実行や CI) は既存の hook を残す。
confirm_overwrite() {
  local destination=$1
  if [ "$assume_yes" = 1 ]; then
    return 0
  fi
  local reply=''
  printf 'Overwrite %s? [y/N] ' "$destination"
  if ! read -r reply; then
    echo
    return 1
  fi
  case "$reply" in
    y | Y | yes | YES) return 0 ;;
    *) return 1 ;;
  esac
}

for source in "$root"/scripts/hooks/*; do
  name=$(basename "$source")
  destination="$hooks_dir/$name"

  if cmp -s "$source" "$destination"; then
    # 同じ内容でも実行権限が落ちていると hook は呼ばれないので、権限だけ入れ直す。
    if [ -x "$destination" ]; then
      echo "unchanged: $destination"
    else
      chmod 755 "$destination"
      echo "fixed mode: $destination"
    fi
    continue
  fi

  if [ -e "$destination" ]; then
    echo "differs: $destination"
    diff -u "$destination" "$source" || true
    if ! confirm_overwrite "$destination"; then
      echo "kept: $destination"
      kept=1
      continue
    fi
  fi

  cp "$source" "$destination"
  chmod 755 "$destination"
  echo "installed: $destination"
done

if [ "$kept" = 1 ]; then
  echo "Error: some hooks were left as they are; re-run with --yes to overwrite them." >&2
  exit 1
fi
