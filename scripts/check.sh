#!/usr/bin/env bash
# 共享源码检查汇总入口：与 .github/workflows/build.yml「Check shared source」一步同源。
# 用法：bash scripts/check.sh
# Flutter 经 mise 按 mise.toml 固定版本解析；跑完后已跟踪文件须与开始时一致。
# 脚本与平台壳的检查在 scripts/lint.sh，这里与 build.yml 都调用它。
# APK/iOS 真实构建、产物校验与真机体验不在此列，经手动 build.yml 与真机验证。
set -euo pipefail
cd "$(dirname "$0")/.."

tracked_state() { git status --porcelain --untracked-files=no && git diff HEAD | git hash-object --stdin; }
before=$(tracked_state)

mise exec -- flutter pub get --enforce-lockfile
mise exec -- dart format --output=none --set-exit-if-changed lib test
mise exec -- flutter analyze --no-pub
mise exec -- flutter test --no-pub
bash scripts/lint.sh

if [[ "$(tracked_state)" != "$before" ]]; then
  echo "检查改动了已跟踪文件：" >&2
  git status --porcelain --untracked-files=no >&2
  exit 1
fi
