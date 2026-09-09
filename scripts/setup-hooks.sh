#!/bin/bash
set -euo pipefail

# scripts/hooks/ の内容をこのリポジトリの hooks ディレクトリへ複製する。
# 宛先は共通 Git ディレクトリ (--git-common-dir) なので、worktree から実行しても
# 全 worktree が共有する 1 か所へ入る。
# core.hooksPath は設定しない。user レベルの dispatcher を覆い隠さないためである。
root=$(git rev-parse --show-toplevel)
hooks_dir=$(git rev-parse --path-format=absolute --git-common-dir)/hooks

mkdir -p "$hooks_dir"

for source in "$root"/scripts/hooks/*; do
  name=$(basename "$source")
  destination="$hooks_dir/$name"
  if cmp -s "$source" "$destination"; then
    echo "unchanged: $destination"
    continue
  fi
  cp "$source" "$destination"
  chmod 755 "$destination"
  echo "installed: $destination"
done
