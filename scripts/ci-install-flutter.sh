#!/usr/bin/env bash
# CI 安装 Flutter：按 mise.lock 里本平台的官方发布包地址与 SHA-256 下载、校验、解压，
# 再把 flutter/bin 写进 $GITHUB_PATH，后续步骤直接调用 PATH 上的 flutter 与 dart。
# 版本、地址与校验值只在 mise.lock（mise lock 生成），本机经 mise 用的是同一份。
# 用法：bash scripts/ci-install-flutter.sh [安装目录]，安装目录默认 $RUNNER_TOOL_CACHE 或 $HOME 下的 flutter-sdk。
# 兼容 macOS 自带的 bash 3.2。
set -euo pipefail
cd "$(dirname "$0")/.."

case "$(uname -s)" in
  Linux) os=linux ;;
  Darwin) os=macos ;;
  *)
    echo "不支持的系统：$(uname -s)" >&2
    exit 1
    ;;
esac
case "$(uname -m)" in
  x86_64 | amd64) arch=x64 ;;
  arm64 | aarch64) arch=arm64 ;;
  *)
    echo "不支持的架构：$(uname -m)" >&2
    exit 1
    ;;
esac
platform="$os-$arch"

# 取 [tools.flutter."platforms.<platform>"] 一节里的某个字段的值。
lock_field() {
  awk -v section="[tools.flutter.\"platforms.$platform\"]" -v key="$1" '
    /^\[/ { in_section = ($0 == section) }
    in_section && $1 == key { gsub(/^[^"]*"|"$/, ""); print; exit }
  ' mise.lock
}
url=$(lock_field url)
checksum=$(lock_field checksum)
checksum=${checksum#sha256:}
if [ -z "$url" ] || [ -z "$checksum" ]; then
  echo "mise.lock 里没有 $platform 的 Flutter 地址或校验值" >&2
  exit 1
fi

dest=${1:-${RUNNER_TOOL_CACHE:-$HOME}/flutter-sdk}
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
archive="$work/${url##*/}"

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

rm -rf "$dest"
mkdir -p "$dest"
case "$archive" in
  *.tar.xz) tar -xJf "$archive" -C "$dest" ;;
  *.zip) unzip -q "$archive" -d "$dest" ;;
  *)
    echo "不认识的包格式：$archive" >&2
    exit 1
    ;;
esac

bin="$dest/flutter/bin"
if [ -n "${GITHUB_PATH:-}" ]; then
  echo "$bin" >>"$GITHUB_PATH"
fi
"$bin/flutter" --version
echo "Flutter 已装到 $bin"
