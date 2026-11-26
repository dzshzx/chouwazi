#!/usr/bin/env bash
# 经 fvm 按仓根 .fvmrc 安装 Flutter，并在第一次运行 Flutter 之前核对它的提交号；
# scripts/check.sh 与 scripts/ci-setup-fvm.sh 都调用本脚本。
# fvm 按 .fvmrc 的版本号（git tag）克隆 Flutter，tag 可以被改指，所以这里固定该 tag 的提交号。
# 升级 Flutter 时同时改 .fvmrc 与下方 FLUTTER_REVISION，提交号取自
# `git ls-remote https://github.com/flutter/flutter.git refs/tags/<版本>`。
# 用法：bash scripts/flutter-install.sh（PATH 上须有 fvm）
# 兼容 macOS 自带的 bash 3.2。
set -euo pipefail
cd "$(dirname "$0")/.."

FLUTTER_REVISION=c197ca5199e9ee5661e865596c9c07294dfab51c

# --fvm-skip-input：不交互提问；--skip-pub-get：依赖由调用方按锁文件解析。已装则跳过。
fvm install --fvm-skip-input --skip-pub-get
version=$(sed -n 's/.*"flutter": *"\([^"]*\)".*/\1/p' .fvmrc)
versions_dir=$(fvm api context | sed -n 's/.*"versionsCachePath": *"\([^"]*\)".*/\1/p')
revision=$(git -C "$versions_dir/$version" rev-parse HEAD)
if [ "$revision" != "$FLUTTER_REVISION" ]; then
  echo "Flutter $version 的提交号是 $revision，与固定的 $FLUTTER_REVISION 不符。" >&2
  echo "升级 Flutter 时把 FLUTTER_REVISION 改成 git ls-remote https://github.com/flutter/flutter.git refs/tags/$version 的结果。" >&2
  exit 1
fi
echo "Flutter $version 提交号核对一致：$revision"
