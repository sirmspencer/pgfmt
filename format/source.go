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
	"unicode/utf8"

	pg "github.com/pganalyze/pg_query_go/v6"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// placeholderName matches a :name placeholder body. names follow clojure
// keywords as inquery uses them: kebab-case segments, and ? or ! allowed.
var placeholderName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_?!]*(-[A-Za-z0-9_?!]+)*`)

// placeholderMode is how a placeholder is shown to the parser. inquery
// replaces :name with raw text, so a placeholder can stand for a value, a sql
// fragment such as `and status = 'open'`, or a table or column name.
type placeholderMode int

// learn: a defined type over int plus iota gives an enum; the zero value, asValue, is the default
// learn: for any name missing from the modes map.
const (
	asValue      placeholderMode = iota // a parameter: $0001
	asFragment                          // a comment the parser skips: /*__*/
	asName                              // an identifier: _0001
	asList                              // a parenthesized list, as after values or in: ($001)
	asAssignment                        // a set assignment: _1=$0001
)

// placeholders maps :name placeholders to stand-ins postgres can parse
type placeholders struct {
	numbers  map[string]int32
	names    map[int32]string
	modes    map[string]placeholderMode
	restores map[string]string // stand-in text -> :name
	next     int32
}

// newPlaceholders starts with every placeholder treated as a value, except the
// modes already learned from earlier parse attempts
func newPlaceholders(modes map[string]placeholderMode) *placeholders {
	if modes == nil {
		modes = map[string]placeholderMode{}
	}
	return &placeholders{
		numbers:  map[string]int32{},
		names:    map[int32]string{},
		modes:    modes,
		restores: map[string]string{},
		next:     1,
	}
}

// substitute replaces each :name with a stand-in of the same length, so every
// byte offset in the parse tree still points into the original text
// learn: this is a method. (ph *placeholders) is the receiver, similar to `this` in other
// learn: languages; a pointer receiver lets the method modify ph.
func (ph *placeholders) substitute(src string) (string, error) {
	scan, err := pg.Scan(src)
	if err != nil {
		return "", err
	}
	for _, token := range scan.Tokens {
		if token.Token == pg.Token_PARAM {
			if number, err := strconv.Atoi(src[token.Start+1 : token.End]); err == nil && int32(number) >= ph.next {
				ph.next = int32(number) + 1
			}
		}
	}
	var out strings.Builder
	last := 0
	// one entry per open bracket: true for an array[...] constructor, whose
	// elements can be placeholders, false for a subscript such as a[1:n]
	var brackets []bool
	for index, token := range scan.Tokens {
		var previous *pg.ScanToken
		if index > 0 {
			previous = scan.Tokens[index-1]
		}
		switch token.Token {
		case pg.Token_ASCII_91:
			brackets = append(brackets, previous != nil && previous.Token == pg.Token_ARRAY)
		case pg.Token_ASCII_93:
			if len(brackets) > 0 {
				brackets = brackets[:len(brackets)-1]
			}
		case pg.Token_ASCII_58:
			if len(brackets) > 0 {
				// in a subscript a colon is a slice; in an array constructor
				// only an element can start with one
				inArray := brackets[len(brackets)-1] && previous != nil &&
					(previous.Token == pg.Token_ASCII_91 || previous.Token == pg.Token_ASCII_44)
				if !inArray {
					continue
				}
			}
			name := placeholderName.FindString(src[token.End:])
			if name == "" {
				continue
			}
			// after a dot, as in t.:column, the placeholder can only be a
			// name. read as a fragment it would vanish and join its neighbors.
			if _, known := ph.modes[name]; !known && previous != nil &&
				previous.Token == pg.Token_ASCII_46 && previous.End == token.Start {
				ph.modes[name] = asName
			}
			standIn, err := ph.standIn(name)
			if err != nil {
				return "", err
			}
			out.WriteString(src[last:token.Start])
			out.WriteString(standIn)
			last = int(token.End) + len(name)
		}
	}
	out.WriteString(src[last:])
	return out.String(), nil
}

// standIn returns the text that replaces :name, exactly as long as :name
func (ph *placeholders) standIn(name string) (string, error) {
	width := len(name) + 1
	number := ph.numberFor(name)
	digits := strconv.Itoa(int(number))
	text := ""
	switch ph.modes[name] {
	case asFragment:
		text = "/*" + strings.Repeat("_", width-4) + "*/"
	case asName:
		text = "_" + strings.Repeat("0", max(width-1-len(digits), 0)) + digits
	case asList:
		text = "($" + strings.Repeat("0", max(width-3-len(digits), 0)) + digits + ")"
	case asAssignment:
		// the column part stays short so it barely affects the alignment of the others
		text = "_" + digits + "=$" + strings.Repeat("0", max(width-3-2*len(digits), 0)) + digits
	default:
		text = "$" + strings.Repeat("0", max(width-1-len(digits), 0)) + digits
	}
	if len(text) != width {
		return "", fmt.Errorf("placeholder :%s is too short to substitute", name)
	}
	ph.restores[text] = ":" + name
	return text, nil
}

// next mode to try for a placeholder the parser rejected, or false when none is left
func (ph *placeholders) escalate(name string) bool {
	switch ph.modes[name] {
	case asValue:
		ph.modes[name] = asFragment
		// a fragment needs room for /* and */
		if len(name)+1 < 4 {
			ph.modes[name] = asName
		}
		return true
	case asFragment:
		ph.modes[name] = asName
		return true
	case asName:
		// ($1) needs four characters, _1=$1 five
		if len(name)+1 >= 4 {
			ph.modes[name] = asList
			return true
		}
		fallthrough
	case asList:
		if len(name)+1 >= 5 {
			ph.modes[name] = asAssignment
			return true
		}
	}
	return false
}

// assignment returns the placeholder a set target stands in for, when the
// target is an assignment stand-in such as _1=$0001
func (ph *placeholders) assignment(target *pg.ResTarget) (string, bool) {
	param := target.Val.GetParamRef()
	if param == nil {
		return "", false
	}
	name, ok := ph.names[param.Number]
	if !ok || ph.modes[name] != asAssignment || target.Name != "_"+strconv.Itoa(int(param.Number)) {
		return "", false
	}
	return name, true
}

// unwrap puts back placeholders whose stand-ins printed as more than one
// token: a list's parentheses and an assignment's column are dropped, leaving
// :name where the stand-in was
func (ph *placeholders) unwrap(text string) string {
	for name, mode := range ph.modes {
		placeholder := regexp.QuoteMeta(":" + name)
		var pattern *regexp.Regexp
		switch mode {
		case asList:
			pattern = regexp.MustCompile(`\(((?:` + markStart + `\d+` + markEnd + `)?)` + placeholder + `\)`)
		case asAssignment:
			number := strconv.Itoa(int(ph.numberFor(name)))
			pattern = regexp.MustCompile(`((?:` + markStart + `\d+` + markEnd + `)?)_` + number + ` *= *` + placeholder + `([^A-Za-z0-9_?!-]|$)`)
		default:
			continue
		}
		text = pattern.ReplaceAllString(text, "${1}:"+name+"${2}")
	}
	return text
}

// learn: \w is not enough here because stand-ins start with $ or _, so the class lists them.
var standInWord = regexp.MustCompile(`[$A-Za-z0-9_]+`)

// restore puts :name back wherever a stand-in appears as a whole word, such as
// in a printed identifier or a parser error message
func (ph *placeholders) restore(text string) string {
	if len(ph.restores) == 0 {
		return text
	}
	return standInWord.ReplaceAllStringFunc(text, func(word string) string {
		if name, ok := ph.restores[word]; ok {
			return name
		}
		return word
	})
}

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
	col := utf8.RuneCountInString(before[strings.LastIndex(before, "\n")+1:]) + 1
	return line, col
}

// typedLiteralTypes are built-in types a value placeholder may follow, as in
// `date :day` or `interval :span`. keyword types such as integer and
// timestamp parse as pg_catalog names and are always allowed.
var typedLiteralTypes = map[string]bool{
	"date": true, "text": true, "json": true, "jsonb": true, "uuid": true, "bytea": true,
	"inet": true, "cidr": true, "macaddr": true, "money": true, "xml": true, "timestamptz": true,
	"timetz": true, "int2": true, "int4": true, "int8": true, "float4": true, "float8": true, "bool": true,
}

// misreadPlaceholder finds a value placeholder the parser read as a typed
// literal after an ordinary name, as in `where active :filters`, which parses
// as a cast to a type called active. it returns the placeholder's offset, or -1.
func misreadPlaceholder(tree *pg.ParseResult, params *placeholders) int {
	found := -1
	var walk func(protoreflect.Message)
	walk = func(msg protoreflect.Message) {
		if cast, ok := msg.Interface().(*pg.TypeCast); ok && found < 0 {
			param := cast.Arg.GetParamRef()
			typ := cast.TypeName
			if param != nil && params.names[param.Number] != "" && typ.Location >= 0 && typ.Location < param.Location &&
				len(typ.Names) == 1 && !typedLiteralTypes[typ.Names[0].GetString_().Sval] {
				found = int(param.Location)
			}
		}
		msg.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
			switch {
			case field.IsList() && field.Kind() == protoreflect.MessageKind:
				list := value.List()
				for index := 0; index < list.Len(); index++ {
					walk(list.Get(index).Message())
				}
			case !field.IsList() && !field.IsMap() && field.Kind() == protoreflect.MessageKind:
				walk(value.Message())
			}
			return true
		})
	}
	walk(tree.ProtoReflect())
	return found
}
