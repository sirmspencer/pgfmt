package rulesdoc

import (
	"os"
	"path/filepath"
	"testing"
)

// rules.md must match what the template renders to, so every example in it is
// the formatter's real output and nobody edits rules.md by hand
func TestRulesDocumentIsCurrent(t *testing.T) {
	root := filepath.Join("..", "..")
	template, err := os.ReadFile(filepath.Join(root, TemplatePath))
	if err != nil {
		t.Fatal(err)
	}
	current, err := os.ReadFile(filepath.Join(root, OutputPath))
	if err != nil {
		t.Fatal(err)
	}
	rendered, err := Render(string(template))
	if err != nil {
		t.Fatal(err)
	}
	if rendered != string(current) {
		t.Errorf("%s is out of date with %s or the formatter; run `go run ./tools/rules`", OutputPath, TemplatePath)
	}
}
