package format

import (
	"regexp"
	"strconv"
	"strings"

	pg "github.com/pganalyze/pg_query_go/v6"
)

// learn: an identifier matching this pattern never needs quotes, unless it is also a keyword.
var simpleIdent = regexp.MustCompile(`^[a-z_][a-z0-9_$]*$`)

// learn: inside a quoted sql identifier a double quote is escaped by doubling it.
func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// ident prints a column, table, or alias name, quoting it only when required
// learn: postgres keywords fall into categories (see keywords.go). looking up a missing key in a
// learn: map[string]bool returns false, so these maps work as sets.
func ident(name string) string {
	if simpleIdent.MatchString(name) && !reservedKeywords[name] && !typeFuncKeywords[name] {
		return name
	}
	return quoteIdent(name)
}

// funcIdent prints a function or type name, which may be a type/function keyword
func funcIdent(name string) string {
	if simpleIdent.MatchString(name) && !reservedKeywords[name] && !colNameKeywords[name] {
		return name
	}
	return quoteIdent(name)
}

// labelIdent prints a name after as, where any keyword is allowed
func labelIdent(name string) string {
	if simpleIdent.MatchString(name) {
		return name
	}
	return quoteIdent(name)
}

// learn: functions are values in go. the quote parameter has type func(string) string, so a caller
// learn: passes ident or funcIdent depending on where the name appears.
func qualifiedName(nodes []*pg.Node, quote func(string) string) string {
	parts := make([]string, len(nodes))
	for index, node := range nodes {
		if node.GetString_() == nil {
			fail("name part %T", node.Node)
		}
		parts[index] = quote(node.GetString_().Sval)
	}
	return strings.Join(parts, ".")
}

func nameList(nodes []*pg.Node) string {
	parts := make([]string, len(nodes))
	for index, node := range nodes {
		if node.GetString_() == nil {
			fail("name %T", node.Node)
		}
		parts[index] = ident(node.GetString_().Sval)
	}
	return strings.Join(parts, ", ")
}

// canonicalTypes maps internal type names to the spelling we print
// learn: a map literal. the parser stores `integer` as pg_catalog.int4 and `boolean` as
// learn: pg_catalog.bool; this maps the internal names back to the spellings we print.
var canonicalTypes = map[string]string{
	"int2":   "smallint",
	"int4":   "integer",
	"int8":   "bigint",
	"float4": "real",
	"float8": "double precision",
	"bool":   "boolean",
}

// catalogTypes are pg_catalog types whose keyword spelling differs from the internal name
var catalogTypes = map[string]string{
	"bpchar":      "char",
	"varchar":     "varchar",
	"numeric":     "numeric",
	"timestamp":   "timestamp",
	"timestamptz": "timestamptz",
	"time":        "time",
	"timetz":      "timetz",
	"interval":    "interval",
	"bit":         "bit",
	"varbit":      "varbit",
}

func isPlainType(typ *pg.TypeName, name string) bool {
	return len(typ.Names) == 2 && typ.Names[0].GetString_().Sval == "pg_catalog" &&
		typ.Names[1].GetString_().Sval == name && len(typ.Typmods) == 0 && len(typ.ArrayBounds) == 0
}

// learn: the switch picks the first matching rule: known pg_catalog types, then bare aliases like
// learn: int4, then anything else printed as written.
func (p *printer) typeName(typ *pg.TypeName) string {
	if typ.PctType {
		fail("%%type reference")
	}
	names := make([]string, len(typ.Names))
	for index, node := range typ.Names {
		names[index] = node.GetString_().Sval
	}
	text := ""
	switch {
	case len(names) == 2 && names[0] == "pg_catalog" && canonicalTypes[names[1]] != "":
		text = canonicalTypes[names[1]]
	case len(names) == 2 && names[0] == "pg_catalog" && catalogTypes[names[1]] != "":
		text = catalogTypes[names[1]]
		if names[1] == "interval" && len(typ.Typmods) > 0 {
			fail("interval fields")
		}
	case len(names) == 1 && canonicalTypes[names[0]] != "":
		text = canonicalTypes[names[0]]
	default:
		text = qualifiedName(typ.Names, funcIdent)
	}
	if len(typ.Typmods) > 0 {
		text += "(" + p.exprList(typ.Typmods) + ")"
	}
	for _, bound := range typ.ArrayBounds {
		size := bound.GetInteger().Ival
		if size < 0 {
			text += "[]"
		} else {
			text += "[" + strconv.Itoa(int(size)) + "]"
		}
	}
	if typ.Setof {
		text = "setof " + text
	}
	return text
}
