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

// an example is a sql fence, optionally followed by settings as json on the
// same line: ```sql {"indent": 4}
// learn: (?s) lets . match newlines, and .*? is a lazy match that stops at the first closing fence.
var example = regexp.MustCompile("(?s)```sql( \\{[^\n]*\\})?\n(.*?)```")

// Render returns the rules document for template, with "↓" and the formatted
// output after every sql example
func Render(template string) (string, error) {
	var failure error
	fail := func(err error, input string) {
		if failure == nil {
			failure = fmt.Errorf("example does not format: %w\n%s", err, input)
		}
	}
	rendered := example.ReplaceAllStringFunc(template, func(block string) string {
		match := example.FindStringSubmatch(block)
		settings, input := match[1], match[2]
		opts := format.DefaultOptions()
		if settings != "" {
			parsed, err := format.ParseOptions([]byte(settings))
			if err != nil {
				fail(err, input)
			}
			opts = parsed
		}
		result, err := format.FormatResultWith(input, opts)
		output := ""
		if err != nil {
			fail(err, input)
		} else {
			output = result.Output
		}
		heading := "```sql\n"
		if settings != "" {
			heading = "With `" + settings[1:] + "`:\n\n```sql\n"
		}
		return heading + input + "```\n\n↓\n\n```sql\n" + output + "```"
	})
	return rendered, failure
}
