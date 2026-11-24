package format

import (
	"errors"

	pg "github.com/pganalyze/pg_query_go/v6"
)

// removeStrayComma repairs a syntax error caused by a comma that can only be
// extra: a trailing comma before a closing bracket, a semicolon, the end of
// the statement, or a reserved keyword; a doubled comma; or a comma right
// after an opening parenthesis or a keyword. it returns the source with that
// comma removed, or ok false when the error is anything else.
// learn: every return path returns all three values; go has no implicit defaults for results
// learn: unless they are named, so the failure case spells out `src, Fixed{}, false`.
func removeStrayComma(src string, err error) (string, Fixed, bool) {
	var syntax *SyntaxError
	if !errors.As(err, &syntax) || syntax.Offset < 0 {
		return src, Fixed{}, false
	}
	scan, scanErr := pg.Scan(src)
	if scanErr != nil {
		return src, Fixed{}, false
	}
	var tokens []*pg.ScanToken
	for _, token := range scan.Tokens {
		if !isComment(token) {
			tokens = append(tokens, token)
		}
	}
	// at is the token postgres stopped at; at == len(tokens) means the end of input
	at := len(tokens)
	for index, token := range tokens {
		if int(token.Start) >= syntax.Offset {
			at = index
			break
		}
	}
	isComma := func(index int) bool {
		return index >= 0 && index < len(tokens) && tokens[index].Token == pg.Token_ASCII_44
	}
	closes := func(index int) bool {
		if index >= len(tokens) {
			return true
		}
		switch tokens[index].Token {
		case pg.Token_ASCII_41, pg.Token_ASCII_93, pg.Token_ASCII_59:
			return true
		}
		return tokens[index].KeywordKind == pg.KeywordKind_RESERVED_KEYWORD
	}
	opens := func(index int) bool {
		if index < 0 {
			return false
		}
		token := tokens[index]
		return token.Token == pg.Token_ASCII_40 || token.Token == pg.Token_ASCII_44 || token.KeywordKind != pg.KeywordKind_NO_KEYWORD
	}
	comma := -1
	switch {
	case isComma(at-1) && closes(at):
		comma = at - 1
	case isComma(at) && opens(at-1):
		comma = at
	default:
		return src, Fixed{}, false
	}
	token := tokens[comma]
	line, col := lineCol(src, int(token.Start))
	return src[:token.Start] + src[token.End:], Fixed{Line: line, Column: col, Message: "stray comma"}, true
}
