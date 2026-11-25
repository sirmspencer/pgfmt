package format

import (
	"strings"

	pg "github.com/pganalyze/pg_query_go/v6"
	"google.golang.org/protobuf/proto"
)

// function bodies are strings to the parser, so their sql is formatted separately

// splitDollarBody separates a dollar-quoted body like $fn$ ... $fn$ into its
// delimiter and the code between
// learn: returning several values, with ok last, lets the caller test success and use the parts in
// learn: one statement: `if tag, code, ok := splitDollarBody(body); ok {`.
func splitDollarBody(body string) (tag, code string, ok bool) {
	if !strings.HasPrefix(body, "$") {
		return "", "", false
	}
	end := strings.Index(body[1:], "$")
	if end < 0 {
		return "", "", false
	}
	tag = body[:end+2]
	if len(body) < 2*len(tag) || !strings.HasSuffix(body, tag) {
		return "", "", false
	}
	return tag, body[len(tag) : len(body)-len(tag)], true
}

// formatSQLBody formats a language sql body with the same rules as a file:
// a one-line result stays on the as line, anything longer is indented inside the quotes
func formatSQLBody(body string, opts Options) string {
	tag, code, ok := splitDollarBody(body)
	if !ok {
		return body
	}
	unit := strings.Repeat(" ", opts.Indent)
	if !opts.KeepComments {
		code = stripCommentLines(code)
	}
	result, err := formatSQL(code, opts)
	if err != nil || strings.TrimSpace(result.Output) == "" {
		return reindentBody(lowercaseBody(body), unit)
	}
	out := strings.TrimSuffix(result.Output, "\n")
	if !strings.Contains(out, "\n") {
		return tag + " " + strings.ReplaceAll(out, literalNewline, "\n") + " " + tag
	}
	return tag + "\n" + strings.ReplaceAll(indentBy(out, unit), literalNewline, "\n") + "\n" + tag
}

// formatPlpgsqlBody formats each standalone sql statement inside a plpgsql
// body, then lowercases and indents the whole body
func formatPlpgsqlBody(body string, opts Options) string {
	tag, code, ok := splitDollarBody(body)
	if !ok {
		return body
	}
	unit := strings.Repeat(" ", opts.Indent)
	if !opts.KeepComments {
		code = stripCommentLines(code)
	}
	var out strings.Builder
	last := 0
	for _, span := range plpgsqlStatements(code) {
		lineStart := strings.LastIndex(code[:span.start], "\n") + 1
		lineIndent := leadingSpace(code[lineStart:])
		before := code[last:span.start]
		margin := lineIndent
		// learn: max is a built-in function since go 1.21, like len and append.
		if strings.TrimSpace(code[max(lineStart, last):span.start]) != "" {
			// the statement follows plpgsql code such as `then` or `return query` on
			// its line, so it moves to its own line one level deeper
			before = strings.TrimRight(before, " \t") + "\n" + lineIndent + unit
			margin = lineIndent + unit
		}
		out.WriteString(before)
		out.WriteString(formatEmbedded(code[span.start:span.end], margin, opts))
		last = span.end
		// code after the statement on the same line, such as `end if;`, moves to its own line
		rest := code[last:]
		if lineEnd := strings.Index(rest, "\n"); lineEnd >= 0 {
			rest = rest[:lineEnd]
		}
		trailing := strings.TrimSpace(rest)
		if trailing != "" && !strings.HasPrefix(trailing, "--") && !strings.HasPrefix(trailing, "/*") {
			out.WriteString("\n" + lineIndent)
			last += len(rest) - len(strings.TrimLeft(rest, " \t"))
		}
	}
	out.WriteString(code[last:])
	return reindentBody(lowercaseBody(tag+out.String()+tag), unit)
}

// leadingSpace returns the spaces and tabs at the start of text
func leadingSpace(text string) string {
	return text[:len(text)-len(strings.TrimLeft(text, " \t"))]
}

// codeSpan is a byte range [start, end) within a body
type codeSpan struct {
	start, end int
}

// statementWords start a sql statement inside plpgsql
var statementWords = map[string]bool{"select": true, "insert": true, "update": true, "delete": true, "with": true}

// statementLeaders are the tokens a plpgsql statement can follow
var statementLeaders = map[string]bool{"": true, ";": true, "begin": true, "then": true, "else": true, "loop": true}

// plpgsqlStatements finds the standalone sql statements in plpgsql code: a
// statement word right after a statement boundary or `return query`, up to
// the semicolon that ends it. sql inside expressions, such as
// `if exists (select ...)` or `for r in select ... loop`, is not included.
func plpgsqlStatements(code string) []codeSpan {
	scan, err := pg.Scan(code)
	if err != nil {
		return nil
	}
	tokens := scan.Tokens
	var spans []codeSpan
	var words []string
	for index := 0; index < len(tokens); index++ {
		token := tokens[index]
		if isComment(token) {
			continue
		}
		word := asciiLower(code[token.Start:token.End])
		prev, before := "", ""
		if count := len(words); count > 0 {
			prev = words[count-1]
			if count > 1 {
				before = words[count-2]
			}
		}
		if statementWords[word] && (statementLeaders[prev] || (prev == "query" && before == "return")) {
			if end, next := statementEnd(tokens, index); next >= 0 {
				spans = append(spans, codeSpan{start: int(token.Start), end: int(end)})
				words = append(words, ";")
				index = next
				continue
			}
		}
		words = append(words, word)
	}
	return spans
}

// statementEnd returns the end offset and token index of the semicolon that
// ends the statement starting at tokens[from], or -1 when there is none
func statementEnd(tokens []*pg.ScanToken, from int) (int32, int) {
	depth := 0
	for index := from; index < len(tokens); index++ {
		switch tokens[index].Token {
		case pg.Token_ASCII_40:
			depth++
		case pg.Token_ASCII_41:
			depth--
		case pg.Token_ASCII_59:
			if depth == 0 {
				return tokens[index].End, index
			}
		}
	}
	return -1, -1
}

// formatEmbedded formats one sql statement from plpgsql code, starting each
// continuation line with margin. a statement that does not parse as sql, such
// as plpgsql's `into strict`, is kept as written.
func formatEmbedded(text, margin string, opts Options) string {
	result, err := formatSQL(text, opts)
	if err != nil {
		return text
	}
	formatted := strings.TrimSuffix(result.Output, "\n")
	lines := strings.Split(formatted, "\n")
	for index := 1; index < len(lines); index++ {
		if lines[index] != "" {
			lines[index] = margin + lines[index]
		}
	}
	return strings.ReplaceAll(strings.Join(lines, "\n"), literalNewline, "\n")
}

// canonicalizeBody replaces a function body with a form that two bodies with
// the same meaning share, so verification compares meaning, not layout
func canonicalizeBody(fn *pg.CreateFunctionStmt) {
	language := ""
	var body *pg.String
	for _, node := range fn.Options {
		opt := node.GetDefElem()
		switch {
		case opt.Defname == "language" && opt.Arg.GetString_() != nil:
			language = opt.Arg.GetString_().Sval
		case opt.Defname == "as" && opt.Arg.GetList() != nil && len(opt.Arg.GetList().Items) == 1:
			body = opt.Arg.GetList().Items[0].GetString_()
		}
	}
	if body == nil {
		return
	}
	switch language {
	case "sql":
		body.Sval = canonicalSQL(body.Sval)
	case "plpgsql":
		body.Sval = canonicalPlpgsql(body.Sval)
	}
}

// canonicalSQL is the normalized syntax tree of sql code, or its tokens when it does not parse
// learn: proto.MarshalOptions{Deterministic: true} encodes a message to bytes the same way every
// learn: time, so two equal trees always give equal strings.
func canonicalSQL(code string) string {
	text, err := newPlaceholders(nil).substitute(code)
	if err != nil {
		return canonicalCode(code)
	}
	tree, err := pg.Parse(text)
	if err != nil {
		return canonicalCode(code)
	}
	var out []byte
	for _, raw := range tree.Stmts {
		encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(normalize(raw.Stmt))
		if err != nil {
			return canonicalCode(code)
		}
		out = append(out, encoded...)
		out = append(out, 0)
	}
	return string(out)
}

// canonicalPlpgsql compares the embedded sql statements by syntax tree and the
// plpgsql code around them by tokens
func canonicalPlpgsql(code string) string {
	var parts []string
	last := 0
	for _, span := range plpgsqlStatements(code) {
		parts = append(parts, canonicalCode(code[last:span.start]), canonicalSQL(code[span.start:span.end]))
		last = span.end
	}
	parts = append(parts, canonicalCode(code[last:]))
	return strings.Join(parts, "\x00")
}
