#!/usr/bin/env bash
# tool/ 的 Go 模块门禁：版本断言、gofmt、go vet、staticcheck、函数认知复杂度上限与 go test。
# 用法：bash scripts/check-go.sh（scripts/check.sh 调用它）。兼容 macOS 自带的 bash 3.2。
# go 直接用 PATH 上的（本机的 go，CI 由 setup-go 按 tool/go.mod 安装），不设 GOTOOLCHAIN。
set -euo pipefail
cd "$(dirname "$0")/../tool"

STATICCHECK=honnef.co/go/tools/cmd/staticcheck@v0.8.2
GOCOGNIT=github.com/uudashr/gocognit/cmd/gocognit@v1.3.0
# 单个 Go 函数的认知复杂度上限，测试文件里的函数也算。
MAX_COGNITIVE=30

# staticcheck 与 gocognit 经 go run 包@版本 取用：先查本机模块缓存，缓存里没有才走代理下载。
go_tool() {
  GOPROXY="file://$(go env GOMODCACHE)/cache/download,$(go env GOPROXY)" go run "$@"
}

# 工具链版本只写在 go.mod 的 go 行：本机的 Go 比它旧时 go 命令自己下载并换用这一版；比它新时直接用本机那一版，
# 下面的断言报出来。toolchain 行会让 go 命令和 CI 的 setup-go 改按它选版本，版本就有了第二个来源。
want="go$(sed -n 's/^go \([0-9][0-9.]*\)$/\1/p' go.mod)"
have=$(go env GOVERSION)
if [ "$want" != "$have" ]; then
  echo "check-go: Go 工具链是 $have，与 tool/go.mod 的 $want 不符：把 go.mod 的 go 行改成本机这一版，或设 GOTOOLCHAIN=$want 运行" >&2
  exit 1
fi
if grep -q '^toolchain' go.mod; then
  echo "check-go: tool/go.mod 不应有 toolchain 行，版本只由 go 行固定" >&2
  exit 1
fi
# gofmt 的输出随 Go 版本变：用 go 命令按 go.mod 选出的那份工具链自带的 gofmt，不用 PATH 里的。
unformatted=$("$(go env GOROOT)/bin/gofmt" -l .)
if [ -n "$unformatted" ]; then
  printf 'check-go: gofmt 需要重排：\n%s\n' "$unformatted" >&2
  exit 1
fi
# 复杂度不认代码里的豁免标记；gocognit 没有关掉它的开关，这里直接查标记。
if git grep -n 'gocognit:ignore' -- '*.go'; then
  echo "check-go: 不接受上面的 gocognit:ignore 标记，要放宽复杂度上限只能改 MAX_COGNITIVE" >&2
  exit 1
fi
go vet ./...
# staticcheck 用默认规则集。gocognit 只读源码，带构建标签的文件也查；有函数超线时列出并非零退出。
go_tool "$STATICCHECK" ./...
go_tool "$GOCOGNIT" -over "$MAX_COGNITIVE" .
go test -count=1 ./...
