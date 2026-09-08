#!/bin/bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: uninstall.sh [--help] [-y|--yes]

Stop PRX, remove its LaunchAgent, and remove the binary installed at
~/.local/bin/prx. Configuration, databases, run state, and logs are kept.

Use --yes when running without an interactive terminal.
EOF
}

fail() {
  echo "prx uninstall: $*" >&2
  exit 1
}

print_retained_data() {
  local home=$1
  echo ''
  echo 'PRX will keep these files:'
  printf '  %q  (configuration, database, and run state)\n' "$home/Library/Application Support/prx/"
  printf '  %q  (server logs)\n' "$home/Library/Logs/prx/"
  echo 'Keeping these files lets a later installation reuse the existing data.'
}

print_manual_process_advice() {
  echo 'Stop any PRX process started with prx serve --addr, prx serve --demo,'
  echo 'or a custom PRX_RUN_DIR manually; this script only manages the standard daemon.'
}

run_prx() {
  env -u PRX_RUN_DIR "$prx" "$@"
}

main() {
  local assume_yes=false help_requested=false
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --help)
        help_requested=true
        ;;
      -y | --yes)
        assume_yes=true
        ;;
      *)
        fail "unknown option $1; the only options are --help and --yes"
        ;;
    esac
    shift
  done
  if [ "$help_requested" = true ]; then
    usage
    return 0
  fi

  [ "$(uname -s)" = Darwin ] || fail "prx only runs on macOS"
  [[ "${HOME:-}" = /* ]] || fail "HOME must be an absolute path"

  local home=$HOME installed="$HOME/.local/bin/prx" standard_entry=false standard_symlink_target=''
  if [ -d "$installed" ] && [ ! -L "$installed" ]; then
    print_retained_data "$home" >&2
    fail "$installed is a directory; it was not changed"
  fi
  if [ -L "$installed" ] && [ -d "$installed" ]; then
    print_retained_data "$home" >&2
    fail "$installed points to a directory; it was not changed"
  fi
  if [ -e "$installed" ] && [ ! -L "$installed" ] && [ ! -f "$installed" ]; then
    print_retained_data "$home" >&2
    fail "$installed is not a regular file; it was not changed"
  fi
  if [ -e "$installed" ] || [ -L "$installed" ]; then
    standard_entry=true
  fi
  if [ -L "$installed" ]; then
    standard_symlink_target=$(readlink "$installed" 2>/dev/null || true)
  fi

  if [ -x "$installed" ]; then
    prx=$installed
  else
    prx=$(command -v prx 2>/dev/null || true)
    [ -n "$prx" ] && [ -x "$prx" ] || {
      print_retained_data "$home" >&2
      print_manual_process_advice >&2
      fail "prx was not found on PATH or at $installed; automatic cleanup was not run"
    }
  fi

  echo 'PRX uninstall will:'
  echo '  stop the standard PRX server and remove its LaunchAgent'
  if [ "$standard_entry" = true ]; then
    if [ -L "$installed" ]; then
      printf '  remove the symlink %q (its target will be kept)\n' "$installed"
    elif [ -x "$installed" ]; then
      printf '  remove the installed binary %q\n' "$installed"
    else
      printf '  leave the non-executable file %q in place\n' "$installed"
    fi
  else
    echo '  leave the standard binary in place because it is not installed'
  fi
  print_retained_data "$home"
  print_manual_process_advice
  if [ -n "${PRX_RUN_DIR:-}" ]; then
    printf '  The current PRX_RUN_DIR is %q; it will not be inspected or changed.\n' "$PRX_RUN_DIR"
  fi

  if [ "$assume_yes" != true ]; then
    local answer=''
    echo ''
    printf 'Continue? [y/N] '
    [ -r /dev/tty ] || fail 'no terminal to confirm on; rerun with --yes to skip the question'
    read -r answer < /dev/tty || answer=''
    case "$answer" in
      y | Y | yes | YES) ;;
      *) fail 'cancelled; nothing was changed' ;;
    esac
  fi

  if ! run_prx daemon stop; then
    fail 'daemon stop failed; the binary was left in place; rerun this script after stopping PRX'
  fi
  if ! run_prx daemon uninstall; then
    fail 'daemon uninstall failed; the binary was left in place; rerun this script'
  fi

  local status compact_status
  if ! status=$(run_prx --json daemon); then
    fail 'could not verify the daemon state; the binary was left in place; rerun this script'
  fi
  compact_status=$(printf '%s' "$status" | tr -d '[:space:]')
  case "$compact_status" in
    *'"running":false'*) ;;
    *)
      fail 'the PRX server is still running or its state is unknown; the binary was left in place'
      ;;
  esac
  case "$compact_status" in
    *'"installed":false'*) ;;
    *)
      fail 'the LaunchAgent is still installed or its state is unknown; the binary was left in place'
      ;;
  esac

  if [ -L "$installed" ]; then
    rm -f "$installed" || fail "could not remove $installed; delete it after stopping PRX"
    printf 'Removed symlink %q (the target was kept)\n' "$installed"
    if [ -n "$standard_symlink_target" ]; then
      printf 'Kept symlink target %q\n' "$standard_symlink_target"
    fi
  elif [ -x "$installed" ] && [ ! -d "$installed" ]; then
    rm -f "$installed" || fail "could not remove $installed; delete it after stopping PRX"
    printf 'Removed %q\n' "$installed"
  elif [ "$standard_entry" = true ]; then
    printf 'Kept %q because it is not an executable regular file\n' "$installed"
  fi
  if [ "$prx" != "$installed" ]; then
    printf 'Kept %q because it is outside the standard installation path\n' "$prx"
  fi

  echo ''
  echo 'If you later want to remove the retained PRX data, stop separately started'
  echo 'servers and CLI commands first, then run:'
  printf '  rm -rf %q\n' "$home/Library/Application Support/prx/"
  printf '  rm -rf %q\n' "$home/Library/Logs/prx/"
  echo 'Review PRX_DB, PRX_CONFIG, PRX_RUN_DIR, and command-specific paths separately.'
  echo 'The PATH entry for ~/.local/bin was not changed; remove it from your shell configuration only if unused.'
}

main "$@"
