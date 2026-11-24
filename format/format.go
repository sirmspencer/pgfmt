// learn: a comment directly above the package clause is the package documentation that `go doc`
// learn: shows. exported names (Format, FormatResult, Result) get doc comments that start with the
// learn: name.
// Package format formats postgres sql files while preserving comments
// and :name placeholders.
package format

import (
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
}

// SyntaxError reports sql that postgres cannot parse
type SyntaxError struct {
	Line    int
	Column  int
	Message string
}

// learn: any type with an Error() string method satisfies the built-in error interface. interfaces
// learn: are satisfied implicitly; there is no implements keyword.
func (err *SyntaxError) Error() string {
	return fmt.Sprintf("%d:%d: %s", err.Line, err.Column, err.Message)
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
	result, err := formatSQL(src)
	if err != nil {
		return nil, err
	}
	result.Output = strings.ReplaceAll(result.Output, literalNewline, "\n")
	return result, nil
}

// formatSQL is FormatResult, except that newlines inside literals are still
// marked, so a caller nesting the output (a function body) can indent it safely
func formatSQL(src string) (*Result, error) {
	params := newPlaceholders()
	text, err := params.substitute(src)
	if err != nil {
		return nil, syntaxError(src, err)
	}
	tree, err := pg.Parse(text)
	if err != nil {
		return nil, syntaxError(src, err)
	}
	source, err := newSource(src, text)
	if err != nil {
		return nil, syntaxError(src, err)
	}
	spans, footer := splitStatements(source, tree)
	printer := &printer{src: source, params: params}

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
		line, col := lineCol(src, pgErr.Cursorpos-1)
		return &SyntaxError{Line: line, Column: col, Message: pgErr.Message}
	}
	return err
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
	printed := p.statement(span.raw.Stmt)
	if !p.sameTree(span.raw.Stmt, printed) {
		return "", "formatting would change the statement"
	}
	lines, tags := splitMarks(printed)
	if span.semicolon {
		lines[len(lines)-1] += ";"
	}
	for _, comment := range span.inner {
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
