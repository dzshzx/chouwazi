#!/usr/bin/env bash
# tool/ 的 Go 模块门禁：版本断言、gofmt、go vet、staticcheck、函数认知复杂度上限与 go test。
# 用法：bash scripts/check-go.sh（scripts/check.sh 调用它）。兼容 macOS 自带的 bash 3.2。
# go 直接用 PATH 上的（本机的 go，CI 由 setup-go 按 tool/go.mod 安装），不设 GOTOOLCHAIN。
set -euo pipefail
cd "$(dirname "$0")/../tool"

# 工具版本、复杂度上限与共用检查函数在 scripts/gatelib.sh。
# shellcheck source=scripts/gatelib.sh
source ../scripts/gatelib.sh

# 工具链版本只写在 tool/go.mod 的 go 行，不许有 toolchain 行。
gate_go_version .
# gofmt 的输出随 Go 版本变：用 go 命令按 go.mod 选出的那份工具链自带的 gofmt，不用 PATH 里的。
unformatted=$("$(go env GOROOT)/bin/gofmt" -l .)
if [ -n "$unformatted" ]; then
  printf 'check-go: gofmt 需要重排：\n%s\n' "$unformatted" >&2
  exit 1
fi
# 复杂度不认代码里的豁免标记；gocognit 没有关掉它的开关，这里直接查标记。
gate_no_marker 'gocognit:ignore' '*.go'
go vet ./...
# staticcheck 用默认规则集。gocognit 只读源码，带构建标签的文件也查；有函数超线时列出并非零退出。
gate_go_run "$STATICCHECK" ./...
gate_go_run "$GOCOGNIT" -over "$MAX_COGNITIVE" .
go test -count=1 ./...
