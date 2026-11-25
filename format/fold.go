package format

import (
	"strings"

	pg "github.com/pganalyze/pg_query_go/v6"
)

// asciiLower lowercases only a-z, matching how postgres folds unquoted names
// learn: a string can be converted to []byte and back. ranging over the bytes and changing them in
// learn: place is safe here because ascii letters are always single bytes in utf-8; strings.ToLower
// learn: would also change non-ascii letters, which postgres leaves alone.
func asciiLower(text string) string {
	bytes := []byte(text)
	for index, char := range bytes {
		if char >= 'A' && char <= 'Z' {
			bytes[index] = char + ('a' - 'A')
		}
	}
	return string(bytes)
}

// foldable reports whether a token can be lowercased without changing its
// meaning: keywords and unquoted identifiers
func foldable(token *pg.ScanToken, text string) bool {
	if token.KeywordKind != pg.KeywordKind_NO_KEYWORD {
		return true
	}
	return token.Token == pg.Token_IDENT && !strings.HasPrefix(text, `"`) && !strings.HasPrefix(asciiLower(text), `u&"`)
}

// lowercaseCode lowercases the keywords and unquoted identifiers in sql or
// plpgsql code, keeping strings, quoted names, comments, and spacing as written
func lowercaseCode(code string) string {
	scan, err := pg.Scan(code)
	if err != nil {
		return code
	}
	var out strings.Builder
	last := int32(0)
	for _, token := range scan.Tokens {
		text := code[token.Start:token.End]
		if foldable(token, text) {
			text = asciiLower(text)
		}
		out.WriteString(code[last:token.Start])
		out.WriteString(text)
		last = token.End
	}
	out.WriteString(code[last:])
	return out.String()
}

// lowercaseBody lowercases the code inside a dollar-quoted function body such
// as $$ ... $$ or $fn$ ... $fn$; other quoting is returned unchanged
func lowercaseBody(body string) string {
	if !strings.HasPrefix(body, "$") {
		return body
	}
	end := strings.Index(body[1:], "$")
	if end < 0 {
		return body
	}
	tag := body[:end+2]
	if len(body) < 2*len(tag) || !strings.HasSuffix(body, tag) {
		return body
	}
	return tag + lowercaseCode(body[len(tag):len(body)-len(tag)]) + tag
}

// cleanText returns the original text between two offsets with keywords and
// unquoted identifiers lowercased, runs of spaces collapsed, trailing spaces
// removed, and line breaks kept. placeholders and literals are kept as written.
func (src *source) cleanText(from, to int32) string {
	var out strings.Builder
	last := from
	afterLineComment := false
	for _, token := range src.tokens {
		if token.Start < from || token.End > to {
			continue
		}
		gap := cleanGap(src.orig[last:token.Start])
		// no space before a comma on the same line, and a semicolon follows the
		// token before it, unless that token is a -- comment, which would hide it
		if (gap == " " && token.Token == pg.Token_ASCII_44) ||
			(token.Token == pg.Token_ASCII_59 && strings.TrimSpace(gap) == "" && !afterLineComment) {
			gap = ""
		}
		afterLineComment = token.Token == pg.Token_SQL_COMMENT
		out.WriteString(gap)
		text := src.orig[token.Start:token.End]
		// placeholders were swapped for $n tokens, so their original :name text is kept as is
		if token.Token != pg.Token_PARAM && !strings.HasPrefix(text, ":") && foldable(token, text) {
			text = asciiLower(text)
		}
		out.WriteString(text)
		last = token.End
	}
	return out.String()
}

// cleanGap reduces the whitespace between two tokens: a single space on one
// line, or the original line breaks (at most one blank line) and the next
// line's indentation
func cleanGap(gap string) string {
	if gap == "" {
		return ""
	}
	lines := strings.Split(gap, "\n")
	if len(lines) == 1 {
		return " "
	}
	breaks := min(len(lines)-1, 2)
	return strings.Repeat("\n", breaks) + lines[len(lines)-1]
}

// canonicalCode reduces code to its tokens, lowercased where folding is safe
// and joined by single spaces. two bodies with the same canonical code differ
// only in case and whitespace between tokens, which sql and plpgsql ignore.
func canonicalCode(code string) string {
	scan, err := pg.Scan(code)
	if err != nil {
		return code
	}
	// comments never change meaning, so a body with comments removed compares equal
	var parts []string
	for _, token := range scan.Tokens {
		if isComment(token) {
			continue
		}
		text := code[token.Start:token.End]
		if foldable(token, text) {
			text = asciiLower(text)
		}
		parts = append(parts, text)
	}
	return strings.Join(parts, " ")
}

// reindentBody indents the lines of a multi-line dollar-quoted body one level,
// after removing the indentation they share. lines that start inside a
// multi-line string or comment are left alone so their contents never change.
// learn: removing the shared indentation first makes the result the same no matter how the body
// learn: was indented before, so formatting twice gives the same output.
func reindentBody(body, unit string) string {
	end := strings.Index(body[1:], "$")
	if !strings.HasPrefix(body, "$") || end < 0 {
		return body
	}
	tag := body[:end+2]
	code := body[len(tag) : len(body)-len(tag)]
	if !strings.HasPrefix(code, "\n") {
		return body
	}
	scan, err := pg.Scan(code)
	if err != nil {
		return body
	}
	lines := strings.Split(code, "\n")
	protected := make([]bool, len(lines))
	offset := int32(0)
	for index, line := range lines {
		for _, token := range scan.Tokens {
			if token.Start < offset && offset < token.End {
				protected[index] = true
				break
			}
		}
		offset += int32(len(line)) + 1
	}
	shared := -1
	for index, line := range lines {
		if protected[index] || strings.TrimSpace(line) == "" {
			continue
		}
		width := len(line) - len(strings.TrimLeft(line, " \t"))
		if shared < 0 || width < shared {
			shared = width
		}
	}
	if shared < 0 {
		return body
	}
	for index, line := range lines {
		switch {
		case protected[index]:
		case strings.TrimSpace(line) == "":
			lines[index] = ""
		default:
			lines[index] = unit + line[shared:]
		}
	}
	return tag + strings.Join(lines, "\n") + tag
}

// stripComments blanks out every comment in src, keeping its line breaks, so
// offsets and line numbers are unchanged and the parser never sees a comment
// learn: a []byte copy of the string can be edited in place; converting back makes a new string.
func stripComments(src string) string {
	scan, err := pg.Scan(src)
	if err != nil {
		return src
	}
	out := []byte(src)
	for _, token := range scan.Tokens {
		if !isComment(token) || isSeparator(src[token.Start:token.End]) {
			continue
		}
		for index := token.Start; index < token.End; index++ {
			if out[index] != '\n' {
				out[index] = ' '
			}
		}
	}
	return string(out)
}

// stripCommentLines removes comments from function body code: a line that
// held only a comment is dropped, and spaces left behind a removed comment are trimmed
func stripCommentLines(code string) string {
	blanked := stripComments(code)
	if blanked == code {
		return code
	}
	original := strings.Split(code, "\n")
	lines := strings.Split(blanked, "\n")
	var kept []string
	for index, line := range lines {
		if line == original[index] {
			kept = append(kept, line)
			continue
		}
		if trimmed := strings.TrimRight(line, " \t"); strings.TrimSpace(trimmed) != "" {
			kept = append(kept, trimmed)
		}
	}
	return strings.Join(kept, "\n")
}

// isSeparator reports whether a comment is the --;; line that migration tools
// such as migratus use to separate statements. it is part of the file's
// structure, so it is kept even when comments are removed.
func isSeparator(comment string) bool {
	return strings.TrimSpace(comment) == "--;;"
}
