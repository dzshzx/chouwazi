#!/usr/bin/env bash
# Build the Go helper in tool/ and run it from the repository root.
# Usage: bash scripts/tool.sh <store-site|testflight-assets|ios-screenshots> [arguments]
# Relative paths in the arguments are relative to the repository root. CHOUWAZI_TOOL names the
# built binary (default build/tool/chouwazi-tool); testflight.sh sets it so cleanup can reuse it.
set -euo pipefail
root=$(cd "$(dirname "$0")/.." && pwd)
bin=${CHOUWAZI_TOOL:-$root/build/tool/chouwazi-tool}
go -C "$root/tool" build -o "$bin" .
cd "$root"
exec "$bin" "$@"
