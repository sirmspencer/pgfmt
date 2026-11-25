// learn: a comment directly above the package clause is the package documentation that `go doc`
// learn: shows. exported names (Format, FormatResult, Result) get doc comments that start with the
// learn: name.
// Package format formats postgres sql files while preserving comments
// and :name placeholders.
package format

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	pg "github.com/pganalyze/pg_query_go/v6"
	"github.com/pganalyze/pg_query_go/v6/parser"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Skipped describes a statement that was left exactly as written
// learn: exported types and fields (uppercase) can be used by other packages, here by cmd/pgfmt.
type Skipped struct {
	Line   int
	Reason string
}

// Result is the formatted text plus any statements that could not be formatted
type Result struct {
	Output  string
	Skipped []Skipped
	Fixed   []Fixed
}

// Fixed describes a syntax error pgfmt repaired, such as a stray comma
type Fixed struct {
	Line    int
	Column  int
	Message string
}

// SyntaxError reports sql that postgres cannot parse
type SyntaxError struct {
	Line    int
	Column  int
	Message string
	// Offset is the byte offset of the error, or -1 when postgres gave no position
	Offset int
}

// learn: any type with an Error() string method satisfies the built-in error interface. interfaces
// learn: are satisfied implicitly; there is no implements keyword.
func (err *SyntaxError) Error() string {
	return fmt.Sprintf("%d:%d: %s", err.Line, err.Column, err.Message)
}

// Options are the formatting settings. DefaultOptions is the house style;
// a .pgfmt.json file changes individual settings.
type Options struct {
	// Indent is the number of spaces per indentation level
	Indent int `json:"indent"`
	// ClauseListLimit is the most items select, group by, order by, and
	// returning keep on the keyword's line; 0 always splits, even one item
	ClauseListLimit int `json:"clauseListLimit"`
	// ParenListLimit is the most items a parenthesized list, such as function
	// arguments or an in list, keeps on one line; 0 always splits a non-empty list
	ParenListLimit int `json:"parenListLimit"`
	// KeepComments keeps comments in place instead of removing them
	KeepComments bool `json:"keepComments"`
	// Align lines up create table columns, set assignments, and select aliases
	Align bool `json:"align"`
	// BlankLinesAroundSetOperators puts a blank line before and after union,
	// intersect, and except
	BlankLinesAroundSetOperators bool `json:"blankLinesAroundSetOperators"`
}

// DefaultOptions returns the house style
func DefaultOptions() Options {
	return Options{
		Indent:                       2,
		ClauseListLimit:              3,
		ParenListLimit:               3,
		KeepComments:                 false,
		Align:                        true,
		BlankLinesAroundSetOperators: true,
	}
}

// ParseOptions reads settings from json, starting from DefaultOptions, so a
// file only needs the settings it changes. unknown keys are an error, which
// catches misspelled settings.
// learn: json.Decoder fills the struct fields named by the json tags. starting from the defaults
// learn: means any key missing from the file keeps its default value.
func ParseOptions(data []byte) (Options, error) {
	opts := DefaultOptions()
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&opts); err != nil {
		return opts, err
	}
	return opts, opts.validate()
}

func (opts Options) validate() error {
	switch {
	case opts.Indent < 1 || opts.Indent > 8:
		return fmt.Errorf("indent must be between 1 and 8, got %d", opts.Indent)
	case opts.ClauseListLimit < 0:
		return fmt.Errorf("clauseListLimit must be 0 or more, got %d", opts.ClauseListLimit)
	case opts.ParenListLimit < 0:
		return fmt.Errorf("parenListLimit must be 0 or more, got %d", opts.ParenListLimit)
	}
	return nil
}

// Format returns the formatted text of a sql file
func Format(src string) (string, error) {
	result, err := FormatResult(src)
	if err != nil {
		return "", err
	}
	return result.Output, nil
}

// statementSpan locates one statement and its comments in the source
// learn: the comments after each field document it. int32 matches the offsets pg_query reports.
type statementSpan struct {
	raw       *pg.RawStmt
	start     int32 // first token of the statement
	end       int32 // end of the last token before the semicolon
	semicolon bool
	through   int32 // end of the statement including its semicolon
	finish    int32 // end of the statement including trailing comments
	leading   []*pg.ScanToken
	inner     []*pg.ScanToken
	trailing  []*pg.ScanToken
}

// FormatResult formats src and reports statements it left unchanged
// learn: the whole pipeline: swap placeholders for $n, parse, scan tokens, split the file into
// learn: statements with their comments, print each statement, then join them with blank lines.
func FormatResult(src string) (*Result, error) {
	return FormatResultWith(src, DefaultOptions())
}

// FormatResultWith is FormatResult with settings other than the defaults
func FormatResultWith(src string, opts Options) (*Result, error) {
	if err := opts.validate(); err != nil {
		return nil, err
	}
	result, err := formatSQL(src, opts)
	if err != nil {
		return result, err
	}
	result.Output = strings.ReplaceAll(result.Output, literalNewline, "\n")
	return result, nil
}

// formatSQL is FormatResult, except that newlines inside literals are still
// marked, so a caller nesting the output (a function body) can indent it safely
func formatSQL(src string, opts Options) (*Result, error) {
	// comments are removed unless kept; blanking them keeps every offset and line number
	if !opts.KeepComments {
		src = stripComments(src)
	}
	var fixes []Fixed
	// modes remembers how each placeholder must be read, learned from failed parses
	modes := map[string]placeholderMode{}
	lastFragment := ""
	for {
		params := newPlaceholders(modes)
		result, err := formatOnce(src, params, opts)
		if err == nil {
			result.Fixed = fixes
			return result, nil
		}
		// the parser stopped at a placeholder: read it as a fragment, then as a name
		if name, ok := placeholderAt(src, err); ok && params.escalate(name) {
			lastFragment = ""
			if modes[name] == asFragment {
				lastFragment = name
			}
			continue
		}
		if fixed, fix, ok := removeStrayComma(src, err); ok {
			fixes = append(fixes, fix)
			src = fixed
			continue
		}
		// hiding a fragment left a gap the statement needs, so it must be a name
		if lastFragment != "" && params.escalate(lastFragment) {
			lastFragment = ""
			continue
		}
		// the fixes found so far are still reported next to the error
		return &Result{Fixed: fixes}, err
	}
}

// placeholderAt returns the :name placeholder where a syntax error happened
func placeholderAt(src string, err error) (string, bool) {
	var syntax *SyntaxError
	if !errors.As(err, &syntax) || syntax.Offset < 0 || syntax.Offset >= len(src) || src[syntax.Offset] != ':' {
		return "", false
	}
	name := placeholderName.FindString(src[syntax.Offset+1:])
	return name, name != ""
}

// formatOnce formats src without repairing syntax errors
func formatOnce(src string, params *placeholders, opts Options) (*Result, error) {
	text, err := params.substitute(src)
	if err != nil {
		return nil, syntaxError(src, err)
	}
	tree, err := pg.Parse(text)
	if err != nil {
		return nil, restoreMessage(params, syntaxError(src, err))
	}
	if offset := misreadPlaceholder(tree, params); offset >= 0 {
		line, col := lineCol(src, offset)
		return nil, &SyntaxError{Line: line, Column: col, Offset: offset, Message: "placeholder read as a typed literal"}
	}
	source, err := newSource(src, text)
	if err != nil {
		return nil, syntaxError(src, err)
	}
	spans, footer := splitStatements(source, tree)
	printer := &printer{src: source, params: params, opts: opts}

	result := &Result{}
	var groups []string
	for _, span := range spans {
		body, reason := printer.formatStatement(span)
		if reason != "" {
			line, _ := lineCol(src, int(span.start))
			result.Skipped = append(result.Skipped, Skipped{Line: line, Reason: reason})
		}
		groups = append(groups, leadingComments(source, span.leading, span.start)+body)
	}
	if len(footer) > 0 {
		groups = append(groups, strings.TrimSuffix(leadingComments(source, footer, -1), "\n"))
	}
	output := strings.Join(groups, "\n\n")
	if output != "" {
		output += "\n"
	}
	result.Output = output
	return result, nil
}

// learn: errors.As looks through err (and anything it wraps) for a *parser.Error and, if found,
// learn: stores it in pgErr. it is the typed way to inspect an error; errors.Is compares against
// learn: sentinel values.
func syntaxError(src string, err error) error {
	var pgErr *parser.Error
	if errors.As(err, &pgErr) {
		offset := -1
		if pgErr.Cursorpos > 0 {
			offset = byteOffset(src, pgErr.Cursorpos-1)
		}
		line, col := lineCol(src, max(offset, 0))
		return &SyntaxError{Line: line, Column: col, Message: pgErr.Message, Offset: offset}
	}
	return err
}

// restoreMessage shows :name instead of its stand-in in a syntax error message
func restoreMessage(params *placeholders, err error) error {
	var syntax *SyntaxError
	if errors.As(err, &syntax) {
		syntax.Message = params.restore(syntax.Message)
	}
	return err
}

// byteOffset converts postgres's character position to a byte offset
// learn: ranging over a string yields (byte index, rune) pairs, stepping over multi-byte characters,
// learn: so counting iterations counts characters while index tracks bytes.
func byteOffset(text string, chars int) int {
	count := 0
	for index := range text {
		if count == chars {
			return index
		}
		count++
	}
	return len(text)
}

// splitStatements assigns every comment to a statement or to the file footer
// learn: the loops below use `for ; condition; post {` with no init statement, so tokenIndex lives
// learn: outside them and each statement continues where the previous one stopped.
func splitStatements(src *source, tree *pg.ParseResult) ([]*statementSpan, []*pg.ScanToken) {
	var spans []*statementSpan
	tokenIndex := 0
	tokens := src.tokens
	for _, raw := range tree.Stmts {
		limit := int32(len(src.text))
		if raw.StmtLen > 0 {
			limit = raw.StmtLocation + raw.StmtLen
		}
		span := &statementSpan{raw: raw, start: -1}
		for ; tokenIndex < len(tokens) && tokens[tokenIndex].Start < limit; tokenIndex++ {
			token := tokens[tokenIndex]
			switch {
			case isComment(token) && span.start < 0:
				span.leading = append(span.leading, token)
			case isComment(token):
				span.inner = append(span.inner, token)
			default:
				if span.start < 0 {
					span.start = token.Start
				}
				span.end = token.End
			}
		}
		span.through = span.end
		// comments between the statement and its semicolon stay inside it
		for index := tokenIndex; index < len(tokens); index++ {
			if tokens[index].Token == pg.Token_ASCII_59 {
				span.inner = append(span.inner, tokens[tokenIndex:index]...)
				span.semicolon = true
				span.through = tokens[index].End
				tokenIndex = index + 1
				break
			}
			if !isComment(tokens[index]) {
				break
			}
		}
		span.finish = span.through
		// comments on the same line after the semicolon trail the statement
		for ; tokenIndex < len(tokens) && isComment(tokens[tokenIndex]); tokenIndex++ {
			if strings.Contains(src.orig[span.finish:tokens[tokenIndex].Start], "\n") {
				break
			}
			span.trailing = append(span.trailing, tokens[tokenIndex])
			span.finish = tokens[tokenIndex].End
		}
		spans = append(spans, span)
	}
	var footer []*pg.ScanToken
	for ; tokenIndex < len(tokens); tokenIndex++ {
		if isComment(tokens[tokenIndex]) {
			footer = append(footer, tokens[tokenIndex])
		}
	}
	return spans, footer
}

// leadingComments prints comments above a statement, keeping single blank lines
func leadingComments(src *source, comments []*pg.ScanToken, next int32) string {
	var out strings.Builder
	for index, comment := range comments {
		out.WriteString(commentText(src, comment))
		out.WriteString("\n")
		following := next
		if index+1 < len(comments) {
			following = comments[index+1].Start
		}
		if src.blankLineBetween(comment.End, following) {
			out.WriteString("\n")
		}
	}
	return out.String()
}

func commentText(src *source, comment *pg.ScanToken) string {
	return strings.TrimRight(src.orig[comment.Start:comment.End], " \t\r\n")
}

// formatStatement prints one statement, falling back to the original text
// when it uses unsupported syntax or the result would parse differently
func (p *printer) formatStatement(span *statementSpan) (string, string) {
	body, reason := p.tryStatement(span)
	if reason == "" {
		return body, ""
	}
	text := p.src.cleanText(span.start, span.through)
	if !p.sameTree(span.raw.Stmt, text) {
		text = p.src.orig[span.start:span.through]
	}
	for _, comment := range span.trailing {
		text += " " + commentText(p.src, comment)
	}
	return strings.ReplaceAll(text, "\n", literalNewline), reason
}

// learn: named results (body, reason) are variables the function returns, and the deferred function
// learn: can assign them. defer schedules a call for when the function exits, even if it panics.
// learn: recover() inside a deferred function stops a panic and returns its value. we only swallow
// learn: our own unsupported type and re-panic anything else, so real bugs still crash loudly.
func (p *printer) tryStatement(span *statementSpan) (body string, reason string) {
	defer func() {
		if recovered := recover(); recovered != nil {
			skip, ok := recovered.(unsupported)
			if !ok {
				panic(recovered)
			}
			body, reason = "", "unsupported "+skip.what
		}
	}()
	// placeholders read as names were printed as their stand-ins
	printed := p.params.restore(p.statement(span.raw.Stmt))
	if !p.sameTree(span.raw.Stmt, printed) {
		return "", "formatting would change the statement"
	}
	lines, tags := splitMarks(printed)
	// placeholder fragments after the last token belong to the statement, so
	// they stay in front of the semicolon, in their original order
	var inner []*pg.ScanToken
	for _, comment := range span.inner {
		if comment.Start > span.end && strings.HasPrefix(p.src.orig[comment.Start:comment.End], ":") {
			if p.src.ownLine(comment.Start) {
				lines = append(lines, commentText(p.src, comment))
				tags = append(tags, -1)
			} else {
				lines[len(lines)-1] += " " + commentText(p.src, comment)
			}
			continue
		}
		inner = append(inner, comment)
	}
	if span.semicolon {
		lines[len(lines)-1] += ";"
	}
	for _, comment := range inner {
		lines, tags = p.placeComment(lines, tags, comment)
	}
	for _, comment := range span.trailing {
		lines[len(lines)-1] += " " + commentText(p.src, comment)
	}
	return strings.Join(lines, "\n"), ""
}

var markPattern = regexp.MustCompile(markStart + `(\d+)` + markEnd)

// splitMarks strips item marks, remembering the first source offset on each line
func splitMarks(text string) ([]string, []int32) {
	lines := strings.Split(text, "\n")
	tags := make([]int32, len(lines))
	for index, line := range lines {
		tags[index] = -1
		if match := markPattern.FindStringSubmatch(line); match != nil {
			loc, _ := strconv.Atoi(match[1])
			tags[index] = int32(loc)
		}
		lines[index] = markPattern.ReplaceAllString(line, "")
	}
	return lines, tags
}

// placeComment puts a comment beside the printed item it was written next to
// learn: append(a[:i], append([]T{x}, a[i:]...)...) inserts x at index i. the inner append copies
// learn: the tail into a new slice first, so the outer append cannot overwrite it.
func (p *printer) placeComment(lines []string, tags []int32, comment *pg.ScanToken) ([]string, []int32) {
	text := commentText(p.src, comment)
	next := -1
	for index, tag := range tags {
		if tag > comment.Start {
			next = index
			break
		}
	}
	if p.src.ownLine(comment.Start) || next == 0 {
		if next < 0 {
			return append(lines, text), append(tags, -1)
		}
		prefix := lines[next][:len(lines[next])-len(strings.TrimLeft(lines[next], " "))]
		lines = append(lines[:next], append([]string{prefix + text}, lines[next:]...)...)
		tags = append(tags[:next], append([]int32{-1}, tags[next:]...)...)
		return lines, tags
	}
	target := len(lines) - 1
	if next > 0 {
		target = next - 1
	}
	lines[target] += " " + text
	return lines, tags
}

// sameTree checks that the printed text parses to the same statement
// learn: the safety net: parse what we printed and compare it with the original tree. if formatting
// learn: ever changed the meaning of a statement, the caller keeps the original text instead.
func (p *printer) sameTree(original *pg.Node, printed string) bool {
	clean := strings.ReplaceAll(markPattern.ReplaceAllString(printed, ""), literalNewline, "\n")
	text, err := p.params.substitute(clean)
	if err != nil {
		return false
	}
	tree, err := pg.Parse(text)
	if err != nil || len(tree.Stmts) != 1 {
		return false
	}
	return proto.Equal(normalize(original), normalize(tree.Stmts[0].Stmt))
}

// normalize drops source offsets and pg_catalog type prefixes, which formatting may change
// learn: proto.Clone deep copies the tree so we do not damage the one we still print from. fields
// learn: are cleared after Range finishes because changing a message while ranging over it is not
// learn: allowed.
func normalize(node *pg.Node) proto.Message {
	clone := proto.Clone(node)
	var walk func(protoreflect.Message)
	walk = func(msg protoreflect.Message) {
		if typ, ok := msg.Interface().(*pg.TypeName); ok && len(typ.Names) == 2 &&
			typ.Names[0].GetString_() != nil && typ.Names[0].GetString_().Sval == "pg_catalog" {
			typ.Names = typ.Names[1:]
		}
		// option order does not matter to postgres, and the body is printed last
		if fn, ok := msg.Interface().(*pg.CreateFunctionStmt); ok {
			sort.SliceStable(fn.Options, func(i, j int) bool {
				return fn.Options[i].GetDefElem().Defname < fn.Options[j].GetDefElem().Defname
			})
			canonicalizeBody(fn)
		}
		var clear []protoreflect.FieldDescriptor
		msg.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
			switch {
			case field.IsList() && field.Kind() == protoreflect.MessageKind:
				list := value.List()
				for index := 0; index < list.Len(); index++ {
					walk(list.Get(index).Message())
				}
			case !field.IsList() && !field.IsMap() && field.Kind() == protoreflect.MessageKind:
				walk(value.Message())
			case strings.HasSuffix(string(field.Name()), "location"):
				clear = append(clear, field)
			}
			return true
		})
		for _, field := range clear {
			msg.Clear(field)
		}
	}
	walk(clone.ProtoReflect())
	return clone
}
