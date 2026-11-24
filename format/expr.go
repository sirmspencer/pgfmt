package format

import (
	"fmt"
	"strconv"
	"strings"

	pg "github.com/pganalyze/pg_query_go/v6"
)

// operator precedence, lowest to highest, following gram.y
// learn: iota is a counter that starts at 0 in each const block and goes up by one per line. lines
// learn: that omit a value repeat the previous expression, so this reads precOr = 1, precAnd = 2,
// learn: and so on. it gives us an ordered list of precedence levels without numbering them by
// learn: hand.
const (
	precOr = iota + 1
	precAnd
	precNot
	precIs
	precCmp
	precLike
	precOther
	precAdd
	precMul
	precExp
	precAt
	precCollate
	precUnary
	precTypecast
	precAtom
)

// learn: a single case can list several values separated by commas.
func operatorPrec(op string) int {
	switch op {
	case "=", "<", ">", "<=", ">=", "<>":
		return precCmp
	case "+", "-":
		return precAdd
	case "*", "/", "%":
		return precMul
	case "^":
		return precExp
	}
	return precOther
}

// unsupported aborts printing of the current statement, which is then left as written
// learn: this type is only used as a panic value. go code normally returns errors, but a recursive
// learn: printer would need an error check after almost every call. instead fail() panics with this
// learn: type and tryStatement in format.go recovers it and keeps the statement as written.
// learn: encoding/json in the standard library uses the same trick internally. the panic never
// learn: escapes this package.
type unsupported struct{ what string }

// learn: `args ...any` is variadic: callers pass any number of arguments of any type, which arrive
// learn: as a slice. `args...` spreads the slice back out into fmt.Sprintf. `any` is shorthand for
// learn: interface{}, the empty interface that every type satisfies.
func fail(format string, args ...any) {
	panic(unsupported{fmt.Sprintf(format, args...)})
}

// expr prints an expression inline; subqueries and case expressions span lines
// learn: `text, _ :=` keeps the first return value and discards the second. go makes you discard
// learn: explicitly rather than silently ignoring a result.
func (p *printer) expr(node *pg.Node) string {
	text, _ := p.exprPrec(node)
	return text
}

// operand prints node, adding parentheses when it binds looser than minPrec
// learn: the syntax tree has no parentheses, so we add them only where precedence requires. this is
// learn: why `a * b + c` stays bare but `(a + b) * c` keeps its parentheses.
func (p *printer) operand(node *pg.Node, minPrec int) string {
	text, prec := p.exprPrec(node)
	if prec < minPrec {
		return "(" + text + ")"
	}
	return text
}

// learn: make([]string, n) allocates a slice with n empty strings that we then fill by index.
func (p *printer) exprList(nodes []*pg.Node) string {
	return strings.Join(p.exprs(nodes), ", ")
}

// exprs prints each node, for callers that lay the items out themselves
func (p *printer) exprs(nodes []*pg.Node) []string {
	parts := make([]string, len(nodes))
	for index, node := range nodes {
		parts[index] = p.expr(node)
	}
	return parts
}

func (p *printer) exprPrec(node *pg.Node) (string, int) {
	switch {
	// learn: a switch with no value after it is a tidier if / else if chain: the first true case
	// learn: runs. pg.Node is a protobuf oneof wrapper: exactly one of its GetX() methods returns
	// learn: non-nil and the rest return nil, so this switch finds which kind of node we have.
	// learn: generated getters are nil safe, which is why calling GetColumnRef() on any node is
	// learn: allowed.
	case node.GetColumnRef() != nil:
		return p.columnRef(node.GetColumnRef().Fields), precAtom
	case node.GetAConst() != nil:
		return p.constant(node.GetAConst())
	case node.GetParamRef() != nil:
		number := node.GetParamRef().Number
		if name, ok := p.params.names[number]; ok {
			return ":" + name, precAtom
		}
		return "$" + strconv.Itoa(int(number)), precAtom
	case node.GetAExpr() != nil:
		return p.aExpr(node.GetAExpr())
	case node.GetBoolExpr() != nil:
		return p.boolExpr(node.GetBoolExpr())
	case node.GetNullTest() != nil:
		test := node.GetNullTest()
		if test.Argisrow {
			fail("row null test")
		}
		word := " is null"
		if test.Nulltesttype == pg.NullTestType_IS_NOT_NULL {
			word = " is not null"
		}
		return p.operand(test.Arg, precIs+1) + word, precIs
	case node.GetBooleanTest() != nil:
		test := node.GetBooleanTest()
		words := map[pg.BoolTestType]string{
			pg.BoolTestType_IS_TRUE:        " is true",
			pg.BoolTestType_IS_NOT_TRUE:    " is not true",
			pg.BoolTestType_IS_FALSE:       " is false",
			pg.BoolTestType_IS_NOT_FALSE:   " is not false",
			pg.BoolTestType_IS_UNKNOWN:     " is unknown",
			pg.BoolTestType_IS_NOT_UNKNOWN: " is not unknown",
		}
		return p.operand(test.Arg, precIs+1) + words[test.Booltesttype], precIs
	case node.GetSubLink() != nil:
		return p.subLink(node.GetSubLink())
	case node.GetFuncCall() != nil:
		return p.funcCall(node.GetFuncCall())
	case node.GetTypeCast() != nil:
		return p.typeCast(node.GetTypeCast())
	case node.GetCaseExpr() != nil:
		return p.caseExpr(node.GetCaseExpr()), precAtom
	case node.GetCoalesceExpr() != nil:
		return list("coalesce(", p.exprs(node.GetCoalesceExpr().Args), ")"), precAtom
	case node.GetMinMaxExpr() != nil:
		minMax := node.GetMinMaxExpr()
		name := "greatest"
		if minMax.Op == pg.MinMaxOp_IS_LEAST {
			name = "least"
		}
		return list(name+"(", p.exprs(minMax.Args), ")"), precAtom
	case node.GetSqlvalueFunction() != nil:
		return sqlValueFunction(node.GetSqlvalueFunction()), precAtom
	case node.GetAArrayExpr() != nil:
		return list("array[", p.exprs(node.GetAArrayExpr().Elements), "]"), precAtom
	case node.GetRowExpr() != nil:
		row := node.GetRowExpr()
		if row.RowFormat == pg.CoercionForm_COERCE_IMPLICIT_CAST {
			return list("(", p.exprs(row.Args), ")"), precAtom
		}
		return list("row(", p.exprs(row.Args), ")"), precAtom
	case node.GetAIndirection() != nil:
		return p.indirection(node.GetAIndirection()), precAtom
	case node.GetCollateClause() != nil:
		collate := node.GetCollateClause()
		return p.operand(collate.Arg, precCollate) + " collate " + qualifiedName(collate.Collname, ident), precCollate
	case node.GetNamedArgExpr() != nil:
		named := node.GetNamedArgExpr()
		return labelIdent(named.Name) + " => " + p.expr(named.Arg), precAtom
	case node.GetSetToDefault() != nil:
		return "default", precAtom
	case node.GetAStar() != nil:
		return "*", precAtom
	}
	fail("expression %T", node.Node)
	return "", 0
}

func (p *printer) columnRef(fields []*pg.Node) string {
	parts := make([]string, len(fields))
	for index, field := range fields {
		switch {
		case field.GetString_() != nil:
			parts[index] = ident(field.GetString_().Sval)
		case field.GetAStar() != nil:
			parts[index] = "*"
		default:
			fail("column reference field %T", field.Node)
		}
	}
	return strings.Join(parts, ".")
}

// learn: constants are printed from the original source tokens so numbers and strings come out
// learn: exactly as written, including e'...' strings and dollar quoting.
func (p *printer) constant(value *pg.A_Const) (string, int) {
	if value.Isnull {
		return "null", precAtom
	}
	// learn: value.Val is an interface holding one of several concrete types. x.(T) is a type
	// learn: assertion; the two-value form sets ok instead of panicking when x does not hold a T.
	if boolean, ok := value.Val.(*pg.A_Const_Boolval); ok {
		if boolean.Boolval.Boolval {
			return "true", precAtom
		}
		return "false", precAtom
	}
	text, ok := "", false
	if value.Location >= 0 {
		text, ok = p.src.tokenText(value.Location)
	}
	if !ok {
		// learn: a type switch checks the concrete type stored in an interface. inside each case,
		// learn: typed has that case's type, so typed.Ival is accessible in the first case only.
		switch typed := value.Val.(type) {
		case *pg.A_Const_Ival:
			text = strconv.Itoa(int(typed.Ival.Ival))
		case *pg.A_Const_Fval:
			text = typed.Fval.Fval
		case *pg.A_Const_Sval:
			text = quoteString(typed.Sval.Sval)
		default:
			fail("constant %T", value.Val)
		}
	}
	// newlines inside literals are protected from indentation until the end
	text = strings.ReplaceAll(text, "\n", literalNewline)
	if strings.HasPrefix(text, "-") {
		return text, precUnary
	}
	return text, precAtom
}

// restricted prints an expression where postgres only accepts its restricted
// b_expr form, such as a column default
func (p *printer) restricted(node *pg.Node) string {
	return p.operand(node, precUnary)
}

func quoteString(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

// learn: A_Expr covers most operators. postgres stores like as the operator ~~ and not like as !~~,
// learn: which is why the maps below translate internal operator names back to sql words.
func (p *printer) aExpr(expr *pg.A_Expr) (string, int) {
	op := ""
	if len(expr.Name) == 1 && expr.Name[0].GetString_() != nil {
		op = expr.Name[0].GetString_().Sval
	} else {
		fail("qualified operator")
	}
	switch expr.Kind {
	case pg.A_Expr_Kind_AEXPR_OP:
		prec := operatorPrec(op)
		if expr.Lexpr == nil {
			operand := p.operand(expr.Rexpr, precUnary)
			if strings.HasPrefix(operand, "-") || strings.HasPrefix(operand, "+") {
				operand = "(" + operand + ")"
			}
			return op + operand, precUnary
		}
		left := precCmp + 1
		if prec != precCmp {
			left = prec
		}
		return p.operand(expr.Lexpr, left) + " " + op + " " + p.operand(expr.Rexpr, prec+1), prec
	case pg.A_Expr_Kind_AEXPR_OP_ANY, pg.A_Expr_Kind_AEXPR_OP_ALL:
		word := "any"
		if expr.Kind == pg.A_Expr_Kind_AEXPR_OP_ALL {
			word = "all"
		}
		prec := operatorPrec(op)
		return p.operand(expr.Lexpr, prec+1) + " " + op + " " + word + "(" + p.expr(expr.Rexpr) + ")", prec
	case pg.A_Expr_Kind_AEXPR_DISTINCT:
		return p.operand(expr.Lexpr, precIs+1) + " is distinct from " + p.operand(expr.Rexpr, precIs+1), precIs
	case pg.A_Expr_Kind_AEXPR_NOT_DISTINCT:
		return p.operand(expr.Lexpr, precIs+1) + " is not distinct from " + p.operand(expr.Rexpr, precIs+1), precIs
	case pg.A_Expr_Kind_AEXPR_NULLIF:
		return "nullif(" + p.expr(expr.Lexpr) + ", " + p.expr(expr.Rexpr) + ")", precAtom
	case pg.A_Expr_Kind_AEXPR_IN:
		word := " in ("
		if op == "<>" {
			word = " not in ("
		}
		values := expr.Rexpr.GetList()
		if values == nil {
			fail("in without list")
		}
		return p.operand(expr.Lexpr, precLike+1) + list(word, p.exprs(values.Items), ")"), precLike
	case pg.A_Expr_Kind_AEXPR_LIKE, pg.A_Expr_Kind_AEXPR_ILIKE:
		words := map[string]string{"~~": " like ", "!~~": " not like ", "~~*": " ilike ", "!~~*": " not ilike "}
		word, ok := words[op]
		if !ok {
			fail("like operator %s", op)
		}
		if call := expr.Rexpr.GetFuncCall(); call != nil {
			fail("like with escape")
		}
		return p.operand(expr.Lexpr, precLike+1) + word + p.operand(expr.Rexpr, precLike+1), precLike
	case pg.A_Expr_Kind_AEXPR_BETWEEN, pg.A_Expr_Kind_AEXPR_NOT_BETWEEN,
		pg.A_Expr_Kind_AEXPR_BETWEEN_SYM, pg.A_Expr_Kind_AEXPR_NOT_BETWEEN_SYM:
		words := map[pg.A_Expr_Kind]string{
			pg.A_Expr_Kind_AEXPR_BETWEEN:         " between ",
			pg.A_Expr_Kind_AEXPR_NOT_BETWEEN:     " not between ",
			pg.A_Expr_Kind_AEXPR_BETWEEN_SYM:     " between symmetric ",
			pg.A_Expr_Kind_AEXPR_NOT_BETWEEN_SYM: " not between symmetric ",
		}
		bounds := expr.Rexpr.GetList()
		if bounds == nil || len(bounds.Items) != 2 {
			fail("between bounds")
		}
		return p.operand(expr.Lexpr, precLike+1) + words[expr.Kind] +
			p.operand(bounds.Items[0], precLike+1) + " and " + p.operand(bounds.Items[1], precLike+1), precLike
	}
	fail("operator kind %s", expr.Kind)
	return "", 0
}

// learn: the parser turns `x not in (select ...)` into not(x in (select ...)). we detect that shape
// learn: and print the natural form; the self check in format.go confirms both parse identically.
func (p *printer) boolExpr(expr *pg.BoolExpr) (string, int) {
	switch expr.Boolop {
	case pg.BoolExprType_NOT_EXPR:
		arg := expr.Args[0]
		if link := arg.GetSubLink(); link != nil && link.SubLinkType == pg.SubLinkType_ANY_SUBLINK && len(link.OperName) == 0 {
			return p.operand(link.Testexpr, precLike+1) + " not in " + p.subquery(link.Subselect), precLike
		}
		return "not " + p.operand(arg, precUnary), precNot
	case pg.BoolExprType_AND_EXPR, pg.BoolExprType_OR_EXPR:
		word, prec := " and ", precAnd
		if expr.Boolop == pg.BoolExprType_OR_EXPR {
			word, prec = " or ", precOr
		}
		parts := make([]string, len(expr.Args))
		for index, arg := range expr.Args {
			if arg.GetBoolExpr() != nil && arg.GetBoolExpr().Boolop != pg.BoolExprType_NOT_EXPR {
				parts[index] = "(" + p.expr(arg) + ")"
			} else {
				parts[index] = p.operand(arg, prec+1)
			}
		}
		return strings.Join(parts, word), prec
	}
	fail("boolean operator %s", expr.Boolop)
	return "", 0
}

// subquery prints a parenthesized select with its body indented
// learn: the printer builds text bottom up. a subquery prints itself starting at column 0 and the
// learn: caller shifts every line right with indent(). nothing ever needs to know its absolute
// learn: column.
func (p *printer) subquery(node *pg.Node) string {
	return "(\n" + indent(p.statement(node)) + "\n)"
}

func (p *printer) subLink(link *pg.SubLink) (string, int) {
	switch link.SubLinkType {
	case pg.SubLinkType_EXISTS_SUBLINK:
		return "exists " + p.subquery(link.Subselect), precAtom
	case pg.SubLinkType_EXPR_SUBLINK:
		return p.subquery(link.Subselect), precAtom
	case pg.SubLinkType_ARRAY_SUBLINK:
		return "array" + p.subquery(link.Subselect), precAtom
	case pg.SubLinkType_ANY_SUBLINK, pg.SubLinkType_ALL_SUBLINK:
		if len(link.OperName) == 0 {
			return p.operand(link.Testexpr, precLike+1) + " in " + p.subquery(link.Subselect), precLike
		}
		if len(link.OperName) != 1 {
			fail("qualified operator")
		}
		op := link.OperName[0].GetString_().Sval
		word := " any "
		if link.SubLinkType == pg.SubLinkType_ALL_SUBLINK {
			word = " all "
		}
		prec := operatorPrec(op)
		return p.operand(link.Testexpr, prec+1) + " " + op + word + p.subquery(link.Subselect), prec
	}
	fail("sublink %s", link.SubLinkType)
	return "", 0
}

// learn: string concatenation with + is fine for small pieces like these. for long loops a
// learn: strings.Builder would avoid repeated allocation.
func (p *printer) funcCall(call *pg.FuncCall) (string, int) {
	if call.Funcformat == pg.CoercionForm_COERCE_SQL_SYNTAX {
		return p.sqlSyntaxCall(call)
	}
	if call.AggWithinGroup || call.FuncVariadic {
		fail("function call form")
	}
	name := qualifiedName(call.Funcname, funcIdent)
	text := ""
	switch {
	case call.AggStar:
		text = name + "(*)"
	case call.AggDistinct || len(call.AggOrder) > 0:
		// aggregate modifiers keep the arguments on one line
		args := p.exprList(call.Args)
		if call.AggDistinct {
			args = "distinct " + args
		}
		if len(call.AggOrder) > 0 {
			args += " order by " + p.sortList(call.AggOrder)
		}
		text = name + "(" + args + ")"
	default:
		text = list(name+"(", p.exprs(call.Args), ")")
	}
	if call.AggFilter != nil {
		text += " filter (where " + p.expr(call.AggFilter) + ")"
	}
	if call.Over != nil {
		text += " over " + p.window(call.Over)
	}
	return text, precAtom
}

// sqlSyntaxCall prints functions written with special sql syntax
func (p *printer) sqlSyntaxCall(call *pg.FuncCall) (string, int) {
	name := ""
	if len(call.Funcname) == 2 {
		name = call.Funcname[1].GetString_().Sval
	}
	switch {
	case name == "timezone" && len(call.Args) == 2:
		return p.operand(call.Args[1], precAt) + " at time zone " + p.operand(call.Args[0], precAt+1), precAt
	case name == "extract" && len(call.Args) == 2 && call.Args[0].GetAConst() != nil:
		field := call.Args[0].GetAConst().GetSval()
		if field == nil {
			fail("extract field")
		}
		return "extract(" + field.Sval + " from " + p.expr(call.Args[1]) + ")", precAtom
	}
	fail("sql syntax function %s", name)
	return "", 0
}

// learn: FrameOptions is a bit field. def.FrameOptions&1 tests the lowest bit, which postgres sets
// learn: when a query spells out a frame such as `rows between ...`; those are not supported yet.
func (p *printer) window(def *pg.WindowDef) string {
	if def.Name != "" {
		return ident(def.Name)
	}
	if def.FrameOptions&1 != 0 {
		fail("window frame")
	}
	var parts []string
	if def.Refname != "" {
		parts = append(parts, ident(def.Refname))
	}
	if len(def.PartitionClause) > 0 {
		parts = append(parts, "partition by "+p.exprList(def.PartitionClause))
	}
	if len(def.OrderClause) > 0 {
		parts = append(parts, "order by "+p.sortList(def.OrderClause))
	}
	return "(" + strings.Join(parts, " ") + ")"
}

func (p *printer) sortList(nodes []*pg.Node) string {
	parts := make([]string, len(nodes))
	for index, node := range nodes {
		parts[index] = p.sortBy(node)
	}
	return strings.Join(parts, ", ")
}

func (p *printer) sortBy(node *pg.Node) string {
	sort := node.GetSortBy()
	if sort == nil {
		fail("sort item %T", node.Node)
	}
	text := p.expr(sort.Node)
	switch sort.SortbyDir {
	case pg.SortByDir_SORTBY_ASC:
		text += " asc"
	case pg.SortByDir_SORTBY_DESC:
		text += " desc"
	case pg.SortByDir_SORTBY_USING:
		fail("order by using")
	}
	switch sort.SortbyNulls {
	case pg.SortByNulls_SORTBY_NULLS_FIRST:
		text += " nulls first"
	case pg.SortByNulls_SORTBY_NULLS_LAST:
		text += " nulls last"
	}
	return text
}

// learn: cast(x as t) and x::t produce the same tree, so we always print the shorter :: form.
// learn: interval literals are the exception: interval '1 day' reads better than '1 day'::interval.
func (p *printer) typeCast(cast *pg.TypeCast) (string, int) {
	typ := cast.TypeName
	if value := cast.Arg.GetAConst(); value != nil && value.GetSval() != nil && isPlainType(typ, "interval") {
		text, _ := p.constant(value)
		return "interval " + text, precAtom
	}
	return p.operand(cast.Arg, precTypecast) + "::" + p.typeName(typ), precTypecast
}

// learn: mark() embeds an invisible source position at the start of each when line. format.go later
// learn: uses these marks to put comments next to the line they were written beside.
func (p *printer) caseExpr(expr *pg.CaseExpr) string {
	head := "case"
	if expr.Arg != nil {
		head += " " + p.expr(expr.Arg)
	}
	var lines []string
	for _, node := range expr.Args {
		when := node.GetCaseWhen()
		lines = append(lines, mark(when.Location)+"when "+p.expr(when.Expr)+" then "+p.expr(when.Result))
	}
	if expr.Defresult != nil {
		lines = append(lines, mark(minLocation(expr.Defresult))+"else "+p.expr(expr.Defresult))
	}
	return head + "\n" + indent(strings.Join(lines, "\n")) + "\nend"
}

func (p *printer) indirection(expr *pg.A_Indirection) string {
	// a field selected from a column needs parentheses or it reads as a qualified name
	selectsField := false
	for _, item := range expr.Indirection {
		if item.GetString_() != nil || item.GetAStar() != nil {
			selectsField = true
		}
	}
	text := p.expr(expr.Arg)
	if expr.Arg.GetParamRef() == nil && (expr.Arg.GetColumnRef() == nil || selectsField) {
		text = "(" + text + ")"
	}
	for _, item := range expr.Indirection {
		switch {
		case item.GetAIndices() != nil:
			indices := item.GetAIndices()
			lower, upper := "", ""
			if indices.Lidx != nil {
				lower = p.expr(indices.Lidx)
			}
			if indices.Uidx != nil {
				upper = p.expr(indices.Uidx)
			}
			if indices.IsSlice {
				text += "[" + lower + ":" + upper + "]"
			} else {
				text += "[" + upper + "]"
			}
		case item.GetString_() != nil:
			text += "." + ident(item.GetString_().Sval)
		case item.GetAStar() != nil:
			text += ".*"
		default:
			fail("indirection %T", item.Node)
		}
	}
	return text
}

// learn: fn.Op.String() works because generated protobuf enums implement the fmt.Stringer
// learn: interface, which is any type with a String() string method.
func sqlValueFunction(fn *pg.SQLValueFunction) string {
	names := map[pg.SQLValueFunctionOp]string{
		pg.SQLValueFunctionOp_SVFOP_CURRENT_DATE:        "current_date",
		pg.SQLValueFunctionOp_SVFOP_CURRENT_TIME:        "current_time",
		pg.SQLValueFunctionOp_SVFOP_CURRENT_TIME_N:      "current_time",
		pg.SQLValueFunctionOp_SVFOP_CURRENT_TIMESTAMP:   "current_timestamp",
		pg.SQLValueFunctionOp_SVFOP_CURRENT_TIMESTAMP_N: "current_timestamp",
		pg.SQLValueFunctionOp_SVFOP_LOCALTIME:           "localtime",
		pg.SQLValueFunctionOp_SVFOP_LOCALTIME_N:         "localtime",
		pg.SQLValueFunctionOp_SVFOP_LOCALTIMESTAMP:      "localtimestamp",
		pg.SQLValueFunctionOp_SVFOP_LOCALTIMESTAMP_N:    "localtimestamp",
		pg.SQLValueFunctionOp_SVFOP_CURRENT_ROLE:        "current_role",
		pg.SQLValueFunctionOp_SVFOP_CURRENT_USER:        "current_user",
		pg.SQLValueFunctionOp_SVFOP_USER:                "user",
		pg.SQLValueFunctionOp_SVFOP_SESSION_USER:        "session_user",
		pg.SQLValueFunctionOp_SVFOP_CURRENT_CATALOG:     "current_catalog",
		pg.SQLValueFunctionOp_SVFOP_CURRENT_SCHEMA:      "current_schema",
	}
	name, ok := names[fn.Op]
	if !ok {
		fail("sql value function %s", fn.Op)
	}
	if strings.HasSuffix(fn.Op.String(), "_N") {
		name += "(" + strconv.Itoa(int(fn.Typmod)) + ")"
	}
	return name
}
