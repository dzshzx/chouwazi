#!/usr/bin/env bash
# Source this file before invoking the pinned Flutter OH and Hvigor tools.
CHOUWAZI_ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
export CHOUWAZI_TOOLCHAINS="${CHOUWAZI_TOOLCHAINS:-$CHOUWAZI_ROOT/.tooling}"
export FLUTTER_OH_ROOT="$CHOUWAZI_TOOLCHAINS/flutter-oh"
export DEVECO_SDK_HOME="$CHOUWAZI_TOOLCHAINS/harmony/command-line-tools/sdk"
export NODE_HOME="$CHOUWAZI_TOOLCHAINS/harmony/command-line-tools/tool/node"
export PATH="$FLUTTER_OH_ROOT/bin:$NODE_HOME/bin:$CHOUWAZI_TOOLCHAINS/harmony/command-line-tools/ohpm/bin:$CHOUWAZI_TOOLCHAINS/harmony/command-line-tools/hvigor/bin:$PATH"
export FLUTTER_STORAGE_BASE_URL="${FLUTTER_STORAGE_BASE_URL:-https://storage.flutter-io.cn}"
export PUB_HOSTED_URL="${PUB_HOSTED_URL:-https://pub.dev}"
