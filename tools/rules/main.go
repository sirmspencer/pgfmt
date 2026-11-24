// Command rules rebuilds rules.md from docs/rules.template.md. Run it from the
// repository root with `go run ./tools/rules`.
package main

import (
	"fmt"
	"os"

	"github.com/sirmspencer/pgfmt/internal/rulesdoc"
)

func main() {
	template, err := os.ReadFile(rulesdoc.TemplatePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "rules: run from the repository root:", err)
		os.Exit(1)
	}
	rendered, err := rulesdoc.Render(string(template))
	if err != nil {
		fmt.Fprintln(os.Stderr, "rules:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(rulesdoc.OutputPath, []byte(rendered), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "rules:", err)
		os.Exit(1)
	}
	fmt.Println("wrote", rulesdoc.OutputPath)
}
