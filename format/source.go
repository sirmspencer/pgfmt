// learn: every go file starts with a package clause. all .go files in one directory belong to the
// learn: same package, here `format`. other code imports it by module path plus directory:
// learn: github.com/sirmspencer/pgfmt/format.
package format

// learn: imports are grouped by convention: standard library first, then third party. `pg` before
// learn: the path is an alias, so we write pg.Parse instead of pg_query.Parse. go refuses to
// learn: compile a file with an unused import or an unused local variable.
import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	pg "github.com/pganalyze/pg_query_go/v6"
)

// placeholderName matches a :name placeholder body, including kebab-case names
// learn: a package-level variable, initialized once when the program starts. MustCompile panics on
// learn: a bad pattern instead of returning an error, which is fine for patterns written in the
// learn: code. the backticks make a raw string: backslashes are not escape characters inside it.
var placeholderName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(-[A-Za-z0-9_]+)*`)

// placeholders maps :name placeholders to the $n parameters postgres can parse
// learn: a struct groups named fields, like a record or a class without inheritance. names starting
// learn: with a lowercase letter are unexported (private to this package); uppercase names are
// learn: exported. map[string]int32 is a hash map from string keys to int32 values.
type placeholders struct {
	numbers map[string]int32
	names   map[int32]string
	next    int32
}

// learn: go has no constructors. by convention a newX function builds a ready-to-use value.
// learn: &placeholders{...} creates the struct and returns a pointer to it, so every caller shares
// learn: the same instance instead of a copy. maps must be created (here with map literals) before
// learn: writing to them; writing to a nil map panics.
func newPlaceholders() *placeholders {
	return &placeholders{numbers: map[string]int32{}, names: map[int32]string{}, next: 1}
}

// substitute replaces each :name with a zero-padded $n of the same length,
// so every byte offset in the parse tree still points into the original text
// learn: this is a method. (ph *placeholders) is the receiver, similar to `this` in other
// learn: languages; a pointer receiver lets the method modify ph. go functions can return several
// learn: values, and the idiom for failure is returning an error last, where nil means success.
func (ph *placeholders) substitute(src string) (string, error) {
	// learn: := declares new variables and infers their types from the right side. the `if err !=
	// learn: nil` check right after is go's error handling: errors are ordinary return values, not
	// learn: exceptions.
	scan, err := pg.Scan(src)
	if err != nil {
		return "", err
	}
	// learn: range over a slice yields (index, element). `_` discards the index. this first pass
	// learn: finds the highest $n already in the file so our generated numbers cannot collide with
	// learn: real ones.
	for _, token := range scan.Tokens {
		// learn: src[a:b] slices a string to bytes a..b-1 without copying. strconv.Atoi parses an
		// learn: int and returns (value, error). the `if x, err := f(); cond {` form scopes x and
		// learn: err to the if block.
		if token.Token == pg.Token_PARAM {
			if number, err := strconv.Atoi(src[token.Start+1 : token.End]); err == nil && int32(number) >= ph.next {
				ph.next = int32(number) + 1
			}
		}
	}
	// learn: var declares a variable holding its zero value. every go type has one (0, "", false,
	// learn: nil, or a struct of zero values), and a zero strings.Builder is ready to use. the
	// learn: builder appends efficiently instead of allocating a new string for every +.
	var out strings.Builder
	last := 0
	depth := 0
	for _, token := range scan.Tokens {
		// learn: switch compares token.Token against each case. cases do not fall through, so no
		// learn: break is needed. depth counts [ ] nesting so array slices like tags[1:2] are never
		// learn: treated as placeholders. the Token_ASCII_NN names are generated constants for
		// learn: single characters: 91 is [, 93 is ], 58 is :.
		switch token.Token {
		case pg.Token_ASCII_91:
			depth++
		case pg.Token_ASCII_93:
			depth--
		case pg.Token_ASCII_58:
			if depth > 0 {
				continue
			}
			name := placeholderName.FindString(src[token.End:])
			if name == "" {
				continue
			}
			number := ph.numberFor(name)
			digits := strconv.Itoa(int(number))
			if len(digits) > len(name) {
				return "", fmt.Errorf("placeholder :%s is too short to substitute", name)
			}
			out.WriteString(src[last:token.Start])
			// learn: the replacement is zero padded to the exact length of the original, so
			// learn: :user_id (8 bytes) becomes $0000001 (8 bytes). postgres reads it as parameter
			// learn: 1, and because the lengths match, every byte offset the parser reports still
			// learn: points at the same spot in the original text.
			out.WriteString("$" + strings.Repeat("0", len(name)-len(digits)) + digits)
			last = int(token.End) + len(name)
		}
	}
	out.WriteString(src[last:])
	return out.String(), nil
}

// learn: `value, ok := m[key]` is the comma-ok idiom: ok reports whether the key exists. a plain
// learn: m[key] on a missing key returns the zero value, which would be ambiguous here.
func (ph *placeholders) numberFor(name string) int32 {
	if number, ok := ph.numbers[name]; ok {
		return number
	}
	number := ph.next
	ph.next++
	ph.numbers[name] = number
	ph.names[number] = name
	return number
}

// source holds the substituted text and its tokens, indexed by start offset
// learn: []*pg.ScanToken is a slice (a growable array view) of pointers to token structs. pg_query
// learn: generates these structs from libpg_query's protobuf schema. tokenAt is an index we build
// learn: so we can jump from a byte offset to its token without scanning the slice.
type source struct {
	orig    string
	text    string
	tokens  []*pg.ScanToken
	tokenAt map[int32]int
}

func newSource(orig, text string) (*source, error) {
	scan, err := pg.Scan(text)
	if err != nil {
		return nil, err
	}
	src := &source{orig: orig, text: text, tokens: scan.Tokens, tokenAt: map[int32]int{}}
	for index, token := range scan.Tokens {
		src.tokenAt[token.Start] = index
	}
	return src, nil
}

func isComment(token *pg.ScanToken) bool {
	return token.Token == pg.Token_SQL_COMMENT || token.Token == pg.Token_C_COMMENT
}

// tokenText returns the original text of the token starting at loc
// learn: returning (string, bool) is the same comma-ok idiom as map lookups. the parser reports a
// learn: negative number's location on the minus sign, so the minus is glued to the token after it.
func (src *source) tokenText(loc int32) (string, bool) {
	index, ok := src.tokenAt[loc]
	if !ok {
		return "", false
	}
	token := src.tokens[index]
	text := src.orig[token.Start:token.End]
	if token.Token == pg.Token_ASCII_45 && index+1 < len(src.tokens) {
		next := src.tokens[index+1]
		return "-" + src.orig[next.Start:next.End], true
	}
	return text, true
}

// stringAfter returns the first string constant token after loc
func (src *source) stringAfter(loc int32) (string, bool) {
	for _, token := range src.tokens {
		if token.Start > loc && token.Token == pg.Token_SCONST {
			return src.orig[token.Start:token.End], true
		}
	}
	return "", false
}

// ownLine reports whether only whitespace precedes pos on its line
// learn: indexing a string returns a byte, not a character. that is fine here because we only
// learn: compare against ascii bytes. single quotes are character literals ('\n'); double quotes
// learn: are strings.
func (src *source) ownLine(pos int32) bool {
	for index := int(pos) - 1; index >= 0; index-- {
		switch src.orig[index] {
		case '\n':
			return true
		case ' ', '\t', '\r':
			continue
		default:
			return false
		}
	}
	return true
}

// blankLineBetween reports whether the text between two offsets contains an empty line
func (src *source) blankLineBetween(from, to int32) bool {
	if from < 0 || to <= from {
		return false
	}
	return strings.Count(src.orig[from:to], "\n") >= 2
}

// lineCol converts a byte offset to a 1-based line and column
// learn: most string helpers live in the strings package as functions (strings.Count,
// learn: strings.LastIndex) rather than as methods on the string type.
func lineCol(text string, pos int) (int, int) {
	if pos > len(text) {
		pos = len(text)
	}
	before := text[:pos]
	line := strings.Count(before, "\n") + 1
	col := pos - strings.LastIndex(before, "\n")
	return line, col
}
