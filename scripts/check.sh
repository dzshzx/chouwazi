#!/usr/bin/env bash
# 共享源码检查汇总入口，也是合入前门禁；CI 只在 Linux 上调用本脚本：build.yml 的 Android 作业与 release.yml 的 gate 作业。
# 用法：bash scripts/check.sh
# 本机与 CI 都经 fvm 按仓根 .fvmrc 取 Flutter：PATH 上须有 fvm（CI 由 scripts/ci-setup-fvm.sh 安装），
# flutter/dart 一律经 `fvm flutter …`、`fvm dart …` 调用。跑完后已跟踪文件须与开始时一致。
# 脚本与平台壳的检查在 scripts/lint.sh，这里调用它。
# tool/ 的 Go 模块：版本断言、gofmt、go vet、staticcheck、函数认知复杂度上限与 go test，go 直接用 PATH 上的。
# 仓内的 Python 文件只能是 scripts/python-allowlist.txt 列出的（一行一个路径或目录前缀，现为空）。
# APK/iOS 真实构建、产物校验与真机体验不在此列，经手动 build.yml 与真机验证。
set -euo pipefail
cd "$(dirname "$0")/.."

tracked_state() { git status --porcelain --untracked-files=no && git diff HEAD | git hash-object --stdin; }
before=$(tracked_state)

bash scripts/flutter-install.sh
fvm flutter pub get --enforce-lockfile
fvm dart format --output=none --set-exit-if-changed lib test
fvm flutter analyze --no-pub
fvm flutter test --no-pub
bash scripts/lint.sh
bash scripts/check-go.sh

# 允许清单之外的 Python 文件一律拒绝。
# case 模式前的左括号不能省：macOS 自带的 bash 3.2 会把 $( ) 里模式的右括号当成命令替换的结尾。
unexpected=$(git ls-files '*.py' | while IFS= read -r file; do
  allowed=
  while IFS= read -r entry || [ -n "$entry" ]; do
    case "$file" in ("$entry" | "$entry"/*) [ -n "$entry" ] && allowed=1 ;; esac
  done <scripts/python-allowlist.txt
  [ -n "$allowed" ] || echo "$file"
done)
if [ -n "$unexpected" ]; then
  echo "这些 Python 文件不在 scripts/python-allowlist.txt 里：" >&2
  echo "$unexpected" >&2
  exit 1
fi

if [[ "$(tracked_state)" != "$before" ]]; then
  echo "检查改动了已跟踪文件：" >&2
  git status --porcelain --untracked-files=no >&2
  exit 1
fi
