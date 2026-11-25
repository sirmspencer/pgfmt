// Package rulesdoc builds rules.md from docs/rules.template.md by running
// every sql example through the formatter and writing its output below it.
// learn: a package under internal/ can only be imported from inside this module, so it is a safe
// learn: place for project tooling that should not become part of the public api.
package rulesdoc

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

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
			parsed, err := exampleOptions(settings)
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
			heading = exampleHeading(settings[1:]) + "\n\n```sql\n"
		}
		return heading + input + "```\n\n↓\n\n```sql\n" + output + "```"
	})
	return rendered, failure
}

// exampleOptions reads an example's settings. a "path" entry is not a setting:
// it names the file the example stands for, which decides whether it is a migration.
func exampleOptions(settings string) (format.Options, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(settings), &fields); err != nil {
		return format.Options{}, err
	}
	path := ""
	if raw, ok := fields["path"]; ok {
		if err := json.Unmarshal(raw, &path); err != nil {
			return format.Options{}, err
		}
		delete(fields, "path")
	}
	rest, err := json.Marshal(fields)
	if err != nil {
		return format.Options{}, err
	}
	opts, err := format.ParseOptions(rest)
	if err != nil {
		return opts, err
	}
	return opts.ForPath(path), nil
}

// exampleHeading describes an example's settings and, when given, the file it stands for
func exampleHeading(settings string) string {
	var fields map[string]json.RawMessage
	if json.Unmarshal([]byte(settings), &fields) != nil {
		return "With `" + settings + "`:"
	}
	path := ""
	if raw, ok := fields["path"]; ok {
		_ = json.Unmarshal(raw, &path)
		delete(fields, "path")
	}
	parts := []string{}
	if len(fields) > 0 {
		rest, _ := json.Marshal(fields)
		parts = append(parts, "with `"+string(rest)+"`")
	}
	if path != "" {
		parts = append(parts, "as `"+path+"`")
	}
	text := strings.Join(parts, ", ")
	return strings.ToUpper(text[:1]) + text[1:] + ":"
}
