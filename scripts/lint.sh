#!/usr/bin/env bash
# 脚本与平台壳的检查：Shell 过 shellcheck，Swift 与 Kotlin 过函数圈复杂度上限（lizard），代码文件不超过行数上限。
# Dart 由 flutter analyze 检查，Go 由 scripts/check-go.sh 检查，不在这里。
# 用法：bash scripts/lint.sh（scripts/check.sh 调用它）。
# 工具经 uvx 取，版本固定在下方常量。每一步直接调用工具并原样透出它的输出，非零即退出。
# 兼容 macOS 自带的 bash 3.2（在本机 macOS 上经 scripts/check.sh 运行时可能用到它）。
set -euo pipefail
cd "$(dirname "$0")/.."

SHELLCHECK_VERSION=0.11.0
# PyPI 包 shellcheck-py 分发 shellcheck，该版本内置上面的 shellcheck。
SHELLCHECK_PY_VERSION=0.11.0.1
LIZARD_VERSION=1.26.0
# 单个函数的圈复杂度上限（Swift 与 Kotlin，由 lizard 量）与单个代码文件的行数上限。
MAX_COMPLEXITY=15
MAX_LINES=2000

files() {
  local f
  git ls-files -z --cached --others --exclude-standard -- "$@" | while IFS= read -r -d '' f; do
    [ -f "$f" ] || continue
    printf '%s\0' "$f"
  done
}

shellcheck() {
  # 本机已装同版本则直接用；否则经 uvx 取 shellcheck-py。
  local have
  have=$(command shellcheck --version 2>/dev/null | awk '$1 == "version:" { print $2 }' || true)
  if [ "$have" = "$SHELLCHECK_VERSION" ]; then
    command shellcheck "$@"
  else
    uvx -q --from "shellcheck-py==$SHELLCHECK_PY_VERSION" shellcheck "$@"
  fi
}

max_lines() {
  # 列出行数超过 MAX_LINES 的文件并非零退出。末尾加 /dev/null 保证 wc 总会多打一行合计，sed 把这一行去掉。
  wc -l -- "$@" /dev/null | sed '$d' | awk -v max="$MAX_LINES" '$1 > max { print; bad = 1 } END { exit bad }'
}

# 把匹配的文件读进数组 FILES。bash 3.2 没有 mapfile，空数组在 set -u 下也不能展开，
# 所以先写进临时文件（git ls-files 失败时随 set -e 退出），用到 FILES 的地方都先判断非空。
collect() {
  local list f
  list=$(mktemp)
  files "$@" >"$list"
  FILES=()
  while IFS= read -r -d '' f; do
    FILES+=("$f")
  done <"$list"
  rm -f "$list"
}

collect '*.sh' '*.bash'
if [ ${#FILES[@]} -gt 0 ]; then
  echo "== shellcheck"
  shellcheck -- "${FILES[@]}"
fi
collect '*.swift' '*.kt' '*.kts'
if [ ${#FILES[@]} -gt 0 ]; then
  echo "== Swift 与 Kotlin 函数圈复杂度不超过 $MAX_COMPLEXITY"
  uvx -q "lizard==$LIZARD_VERSION" -l swift -l kotlin -C "$MAX_COMPLEXITY" -w "${FILES[@]}"
  # 复杂度不认豁免：lizard 会放过带 #lizard forgives 注释的函数、注释里含 GENERATED CODE 的整个文件
  # 和仓根 whitelizard.txt 里列的函数，所以另查一遍。
  if grep -n -i -e 'lizard forgive' -e 'generated code' -- "${FILES[@]}" || [ -e whitelizard.txt ]; then
    echo "lint: 不接受 lizard 的豁免注释（#lizard forgives、GENERATED CODE）与 whitelizard.txt" >&2
    exit 1
  fi
fi
collect '*.go' '*.sh' '*.bash' '*.swift' '*.kt' '*.kts' '*.dart'
echo "== 代码文件不超过 $MAX_LINES 行"
max_lines "${FILES[@]}"
echo "lint: ok"
