// Command chouwazi-tool holds the repository's build helpers. Run it through
// scripts/tool.sh, which builds it and runs it from the repository root:
//
//	bash scripts/tool.sh store-site [--preview] [--output DIR]
//	bash scripts/tool.sh testflight-assets <prepare|install|verify|cleanup|field NAME>
//	bash scripts/tool.sh ios-screenshots
package main

import (
	"fmt"
	"os"

	"chouwazi/tool/internal/screenshots"
	"chouwazi/tool/internal/storesite"
	"chouwazi/tool/internal/testflight"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: chouwazi-tool <store-site|testflight-assets|ios-screenshots> [arguments]")
		return 2
	}
	switch args[0] {
	case "store-site":
		return storesite.Main(args[1:], os.Stdout, os.Stderr)
	case "testflight-assets":
		return testflight.Main(args[1:], os.Stdout, os.Stderr)
	case "ios-screenshots":
		return screenshots.Main(args[1:], os.Stdout, os.Stderr)
	}
	fmt.Fprintf(os.Stderr, "unknown command %q\n", args[0])
	return 2
}
