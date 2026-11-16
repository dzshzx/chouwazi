#!/usr/bin/env bash
# 按仓里最高的纯 vX.Y.Z tag 和升级档位算出发版的版本号与构建号；release.yml 调用本脚本。
# 用法：bash scripts/release-version.sh <patch|minor|major> [--release]
# 基线是形如 vX.Y.Z（纯数字三段，无前后缀）的 tag 中最高的一个，没有时按 0.0.0。
# 构建号（iOS CFBundleVersion、Android versionCode）= 主版本 × 1000000 + 次版本 × 1000 + 修订号，
# 次版本与修订号须小于 1000，这样构建号随版本单调递增。
# --release 用于正式发版：HEAD 已带纯 vX.Y.Z tag 时失败，说明这个提交已发过版。
# 输出 RELEASE_VERSION=、RELEASE_BUILD= 两行；在 GitHub Actions 里同时写进 $GITHUB_ENV，后续步骤直接读。
# 兼容 macOS 自带的 bash 3.2。
set -euo pipefail
cd "$(dirname "$0")/.."

usage() {
  echo "用法：bash scripts/release-version.sh <patch|minor|major> [--release]" >&2
  exit 2
}
bump=${1:-}
case "$bump" in
  patch | minor | major) ;;
  *) usage ;;
esac
case "${2:-}" in
  '') release= ;;
  --release) release=1 ;;
  *) usage ;;
esac

# 只留纯 vX.Y.Z 的 tag。
pure_tags() { awk '/^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$/'; }

if [ -n "$release" ]; then
  released=$(git tag --points-at HEAD | pure_tags)
  if [ -n "$released" ]; then
    echo "这个提交已发过版：$released。先合入新的改动再发版。" >&2
    exit 1
  fi
fi

previous=$(git tag -l 'v*' | pure_tags | cut -c2- | sort -t. -k1,1n -k2,2n -k3,3n | tail -n 1)
IFS=. read -r major minor patch <<EOF
${previous:-0.0.0}
EOF

case "$bump" in
  major) major=$((major + 1)) minor=0 patch=0 ;;
  minor) minor=$((minor + 1)) patch=0 ;;
  patch) patch=$((patch + 1)) ;;
esac

if [ "$minor" -ge 1000 ] || [ "$patch" -ge 1000 ]; then
  echo "次版本与修订号须小于 1000，构建号公式才能保证单调递增：$major.$minor.$patch" >&2
  exit 1
fi

result="RELEASE_VERSION=$major.$minor.$patch
RELEASE_BUILD=$((major * 1000000 + minor * 1000 + patch))"
echo "$result"
if [ -n "${GITHUB_ENV:-}" ]; then
  echo "$result" >>"$GITHUB_ENV"
fi
