#!/usr/bin/env bash
# 按仓里最高的纯 vX.Y.Z tag 和升级档位算出发版的版本号与构建号；release.yml 调用本脚本。
# 用法：bash scripts/release-version.sh <patch|minor|major>
# 基线是形如 vX.Y.Z（纯数字三段，无前后缀）的 tag 中最高的一个，没有时按 0.0.0。
# 构建号（iOS CFBundleVersion、Android versionCode）= 主版本 × 1000000 + 次版本 × 1000 + 修订号，
# 次版本与修订号须小于 1000，这样构建号随版本单调递增。
# 输出 previous=、version=、build=、tag= 四行；在 GitHub Actions 里同时写进 $GITHUB_OUTPUT。
# 兼容 macOS 自带的 bash 3.2。
set -euo pipefail
cd "$(dirname "$0")/.."

bump=${1:-}
case "$bump" in
  patch | minor | major) ;;
  *)
    echo "用法：bash scripts/release-version.sh <patch|minor|major>" >&2
    exit 2
    ;;
esac

previous=$(git tag -l 'v*' |
  awk '/^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$/ { print substr($0, 2) }' |
  sort -t. -k1,1n -k2,2n -k3,3n | tail -n 1)
previous=${previous:-0.0.0}
IFS=. read -r major minor patch <<EOF
$previous
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

version="$major.$minor.$patch"
build=$((major * 1000000 + minor * 1000 + patch))
result="previous=$previous
version=$version
build=$build
tag=v$version"
echo "$result"
if [ -n "${GITHUB_OUTPUT:-}" ]; then
  echo "$result" >>"$GITHUB_OUTPUT"
fi
