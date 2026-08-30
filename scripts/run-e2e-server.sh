#!/bin/sh
set -eu

cd "$(dirname "$0")/.."
mkdir -p bin
go build -trimpath -o bin/prx ./cmd/prx

exec bin/prx serve --demo --addr 127.0.0.1:0
