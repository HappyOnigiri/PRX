#!/bin/sh
# Usage: check-go-coverage.sh PROFILE MINIMUM PACKAGE...
#
# Restricts the Go coverage PROFILE to the files of PACKAGE... and fails when their
# total statement coverage is below MINIMUM percent. Restricting the profile lets one
# `go test -coverprofile ./...` run measure only the handwritten packages.
set -eu

if [ "$#" -lt 3 ]; then
	echo "usage: $0 PROFILE MINIMUM PACKAGE..." >&2
	exit 2
fi

profile=$1
minimum=$2
shift 2
GO=${GO:-go}

filtered="$(mktemp)"
trap 'rm -f "$filtered"' EXIT

# Match files directly inside each package directory: <import path>/<file>:<positions>.
pattern="$("$GO" list "$@" | sed 's|[.]|\\.|g' | paste -sd '|' -)"
{
	head -n 1 "$profile"
	grep -E "^($pattern)/[^/]+:" "$profile" || true
} >"$filtered"

coverage="$("$GO" tool cover -func="$filtered" | awk '/^total:/ { gsub(/%/, "", $3); print $3 }')"
printf 'Go coverage: %s%% (minimum: %s%%)\n' "$coverage" "$minimum"
awk -v actual="$coverage" -v minimum="$minimum" 'BEGIN {
	if (actual !~ /^[0-9]+([.][0-9]+)?$/ || minimum !~ /^[0-9]+([.][0-9]+)?$/) exit 2
	if (actual + 0 < minimum + 0) exit 1
}'
