#!/bin/sh
set -eu

cd "$(dirname "$0")/.."
mkdir -p bin
go build -trimpath -o bin/prmap ./cmd/prmap

prx_e2e_dir="$(mktemp -d)"
prx_e2e_db="$prx_e2e_dir/prmap-e2e.db"

bin/prmap --db "$prx_e2e_db" --github-fixture demo seed --slug graph-8 --tasks 8 >/dev/null
bin/prmap --db "$prx_e2e_db" --github-fixture demo seed --slug graph-50 --tasks 50 >/dev/null
bin/prmap --db "$prx_e2e_db" --github-fixture demo seed --slug graph-100 --tasks 100 >/dev/null
exec bin/prmap --db "$prx_e2e_db" --github-fixture demo serve --addr 127.0.0.1:7331
