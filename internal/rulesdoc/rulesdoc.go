// Package rulesdoc builds rules.md from docs/rules.template.md by running
// every sql example through the formatter and writing its output below it.
// learn: a package under internal/ can only be imported from inside this module, so it is a safe
// learn: place for project tooling that should not become part of the public api.
package rulesdoc

import (
	"fmt"
	"regexp"

	"github.com/sirmspencer/pgfmt/format"
)

// TemplatePath and OutputPath are relative to the repository root
const (
	TemplatePath = "docs/rules.template.md"
	OutputPath   = "rules.md"
)

// learn: (?s) lets . match newlines, and .*? is a lazy match that stops at the first closing fence.
var example = regexp.MustCompile("(?s)```sql\n(.*?)```")

// Render returns the rules document for template, with "↓" and the formatted
// output after every sql example
func Render(template string) (string, error) {
	var failure error
	rendered := example.ReplaceAllStringFunc(template, func(block string) string {
		input := example.FindStringSubmatch(block)[1]
		output, err := format.Format(input)
		if err != nil && failure == nil {
			failure = fmt.Errorf("example does not format: %w\n%s", err, input)
		}
		return "```sql\n" + input + "```\n\n↓\n\n```sql\n" + output + "```"
	})
	return rendered, failure
}
