#!/usr/bin/env bash
# 脚本与平台壳的检查：Python 过 ruff check 与函数认知复杂度上限（complexipy），Shell 过 shellcheck，
# Swift 与 Kotlin 过函数圈复杂度上限（lizard），代码文件不超过行数上限。Dart 由 flutter analyze 检查，不在这里。
# 用法：bash scripts/lint.sh（scripts/check.sh 与 build.yml「Check shared source」都调用它）。
# 工具经 uvx 取，版本固定在下方常量。每一步直接调用工具并原样透出它的输出，非零即退出。
set -euo pipefail
cd "$(dirname "$0")/.."

RUFF_VERSION=0.17.3
SHELLCHECK_VERSION=0.11.0
# PyPI 包 shellcheck-py 分发 shellcheck，该版本内置上面的 shellcheck。
SHELLCHECK_PY_VERSION=0.11.0.1
LIZARD_VERSION=1.26.0
COMPLEXIPY_VERSION=8.3.0
# 单个函数的复杂度上限与单个代码文件的行数上限。Python 量认知复杂度（complexipy），
# Swift 与 Kotlin 量圈复杂度（lizard），两者共用这一个上限。
MAX_COMPLEXITY=15
MAX_LINES=2000

files() {
  local f
  git ls-files -z --cached --others --exclude-standard -- "$@" | while IFS= read -r -d '' f; do
    [ -f "$f" ] || continue
    printf '%s\0' "$f"
  done
}

ruff() {
  uvx -q "ruff==$RUFF_VERSION" "$@"
}

# 缓存放仓外：complexipy 默认在当前目录建 .complexipy_cache。
complexipy() {
  uvx -q "complexipy==$COMPLEXIPY_VERSION" --cache-dir "${XDG_CACHE_HOME:-$HOME/.cache}/complexipy" "$@"
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

# 进程替换的退出码不会自动传出：每个 < <(...) 之后 wait "$!" 取回，git ls-files 失败时随 set -e 退出。
mapfile -d '' PY < <(files '*.py' '*.pyi')
wait "$!"
mapfile -d '' SH < <(files '*.sh' '*.bash')
wait "$!"
mapfile -d '' SHELLS < <(files '*.swift' '*.kt' '*.kts')
wait "$!"
mapfile -d '' CODE < <(files '*.py' '*.pyi' '*.sh' '*.bash' '*.swift' '*.kt' '*.kts' '*.dart')
wait "$!"

if [ ${#PY[@]} -gt 0 ]; then
  echo "== ruff check"
  ruff check --output-format concise -- "${PY[@]}"
  # --no-ignore：不认行内的 noqa: complexipy；--snapshot-ignore：不认基线快照 complexipy-snapshot.json；
  # --isolated：不读 pyproject.toml、complexipy.toml 里的配置。要放宽只能改 MAX_COMPLEXITY。
  echo "== Python 函数认知复杂度不超过 $MAX_COMPLEXITY"
  complexipy --max-complexity-allowed "$MAX_COMPLEXITY" --no-ignore --snapshot-ignore --isolated --failed --plain -- "${PY[@]}"
fi
if [ ${#SH[@]} -gt 0 ]; then
  echo "== shellcheck"
  shellcheck -- "${SH[@]}"
fi
if [ ${#SHELLS[@]} -gt 0 ]; then
  echo "== Swift 与 Kotlin 函数圈复杂度不超过 $MAX_COMPLEXITY"
  uvx -q "lizard==$LIZARD_VERSION" -l swift -l kotlin -C "$MAX_COMPLEXITY" -w "${SHELLS[@]}"
  # 复杂度不认豁免：lizard 会放过带 #lizard forgives 注释的函数、注释里含 GENERATED CODE 的整个文件
  # 和仓根 whitelizard.txt 里列的函数，所以另查一遍。
  if grep -n -i -e 'lizard forgive' -e 'generated code' -- "${SHELLS[@]}" || [ -e whitelizard.txt ]; then
    echo "lint: 不接受 lizard 的豁免注释（#lizard forgives、GENERATED CODE）与 whitelizard.txt" >&2
    exit 1
  fi
fi
echo "== 代码文件不超过 $MAX_LINES 行"
max_lines "${CODE[@]}"
echo "lint: ok"
