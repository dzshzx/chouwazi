#!/usr/bin/env bash
set +x
set -euo pipefail
umask 077

# Dependency-management exemption: cleanup must run even if uv setup/build
# failed. This helper uses only the standard library of the runner's Python.
python3 scripts/testflight-assets.py cleanup
