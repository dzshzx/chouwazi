#!/usr/bin/env bash
# 共享源码检查汇总入口，也是合入前门禁；.github/workflows/build.yml「Check shared source」也调用本脚本。
# 用法：bash scripts/check.sh
# 取 flutter/dart 的方式：有 mise 时经 mise exec 按 mise.toml 固定版本解析（本机）；
# 没有 mise 时直接用 PATH 上的命令（CI 由 flutter-action 装好）。跑完后已跟踪文件须与开始时一致。
# 脚本与平台壳的检查在 scripts/lint.sh，这里调用它。
# APK/iOS 真实构建、产物校验与真机体验不在此列，经手动 build.yml 与真机验证。
set -euo pipefail
cd "$(dirname "$0")/.."

tracked_state() { git status --porcelain --untracked-files=no && git diff HEAD | git hash-object --stdin; }
before=$(tracked_state)

if command -v mise >/dev/null 2>&1; then
  tool() { mise exec -- "$@"; }
else
  tool() { "$@"; }
fi

tool flutter pub get --enforce-lockfile
tool dart format --output=none --set-exit-if-changed lib test
tool flutter analyze --no-pub
tool flutter test --no-pub
bash scripts/lint.sh

if [[ "$(tracked_state)" != "$before" ]]; then
  echo "检查改动了已跟踪文件：" >&2
  git status --porcelain --untracked-files=no >&2
  exit 1
fi
