#!/usr/bin/env bash
set +x
set -euo pipefail
umask 077

# Cleanup must run even if an earlier step failed. Signing state only exists after
# `testflight.sh sign` built the helper into RUNNER_TEMP, so reuse that binary and
# build only when it is missing.
state_dir="${RUNNER_TEMP}/chouwazi-testflight-${GITHUB_RUN_ID}-${GITHUB_RUN_ATTEMPT}"
if [[ ! -e "$state_dir" && ! -L "$state_dir" ]]; then
  exit 0
fi
tool="$RUNNER_TEMP/chouwazi-tool"
if [[ -x "$tool" ]]; then
  exec "$tool" testflight-assets cleanup
fi
CHOUWAZI_TOOL="$tool" exec bash scripts/tool.sh testflight-assets cleanup
