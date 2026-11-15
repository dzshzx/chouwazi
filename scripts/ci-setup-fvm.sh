#!/usr/bin/env bash
# CI 安装 fvm 与 Flutter：按 runner 平台下载下方固定版本的 fvm 发布包，核对 SHA-256 后解压，
# 把 fvm 加进 $GITHUB_PATH，再由 fvm 按仓根 .fvmrc 安装 Flutter。后续步骤经 `fvm flutter …`、`fvm dart …` 调用，
# 本机用的是同一份 .fvmrc。升级 Flutter 只改 .fvmrc；升级 fvm 改下方版本与各平台校验值
# （取自 GitHub API 中该 release 各资产的 digest 字段）。
# 校验范围：只有 fvm 发布包核对 SHA-256。fvm 安装 Flutter 时不校验 sha256：它按 .fvmrc 里的版本号（git tag）
# 从 Flutter 官方 Git 仓库 https://github.com/flutter/flutter.git 克隆，tag 不按提交哈希固定，
# 只有 git 传输本身的对象校验；Dart SDK 与引擎产物由 Flutter 首次运行时从 Google 官方存储下载。
# 用法：bash scripts/ci-setup-fvm.sh。fvm 解压到 $RUNNER_TEMP（没有时用临时目录）下，Flutter 装在 fvm 的缓存目录。
# 兼容 macOS 自带的 bash 3.2。
set -euo pipefail
cd "$(dirname "$0")/.."

FVM_VERSION=4.3.2
FVM_SHA256_LINUX_X64=17dfced05f5b0cb32398dadd0691acc2bc458e8c4b0045a781cd5aa6c85f6ecb
FVM_SHA256_MACOS_ARM64=c9b1e67e5ff4ddae4710669ff9aa312f750947f68fac5b15fc5d248fb4419342

case "$(uname -s)-$(uname -m)" in
  Linux-x86_64 | Linux-amd64)
    platform=linux-x64
    checksum=$FVM_SHA256_LINUX_X64
    ;;
  Darwin-arm64)
    platform=macos-arm64
    checksum=$FVM_SHA256_MACOS_ARM64
    ;;
  *)
    echo "不支持的平台：$(uname -s) $(uname -m)" >&2
    exit 1
    ;;
esac

name="fvm-$FVM_VERSION-$platform.tar.gz"
url="https://github.com/btwld/fvm/releases/download/$FVM_VERSION/$name"
base=${RUNNER_TEMP:-$(mktemp -d)}
dest="$base/fvm-$FVM_VERSION"
archive="$base/$name"

echo "下载 $url"
curl -fsSL --retry 3 -o "$archive" "$url"
if command -v sha256sum >/dev/null 2>&1; then
  actual=$(sha256sum "$archive" | awk '{ print $1 }')
else
  actual=$(shasum -a 256 "$archive" | awk '{ print $1 }')
fi
if [ "$actual" != "$checksum" ]; then
  echo "SHA-256 不符：期望 $checksum，实际 $actual" >&2
  exit 1
fi

# 发布包里是 fvm/fvm 可执行文件与 fvm/src/。
rm -rf "$dest"
mkdir -p "$dest"
tar -xzf "$archive" -C "$dest"
rm -f "$archive"
bin="$dest/fvm"
if [ -n "${GITHUB_PATH:-}" ]; then
  echo "$bin" >>"$GITHUB_PATH"
fi
"$bin/fvm" --version

# --fvm-skip-input：不交互提问；--skip-pub-get：依赖由 scripts/check.sh 或后续步骤按锁文件解析。
"$bin/fvm" install --fvm-skip-input --skip-pub-get
"$bin/fvm" flutter --version
