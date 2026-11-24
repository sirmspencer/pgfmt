package format

import (
	"strconv"
	"strings"

	pg "github.com/pganalyze/pg_query_go/v6"
)

// learn: postgres stores table persistence as a one letter string: p permanent, t temporary, u
// learn: unlogged.
func persistence(rel *pg.RangeVar) string {
	switch rel.Relpersistence {
	case "t":
		return "temporary "
	case "u":
		return "unlogged "
	}
	return ""
}

func (p *printer) createStmt(stmt *pg.CreateStmt) string {
	if len(stmt.InhRelations) > 0 || stmt.Partbound != nil || stmt.Partspec != nil || stmt.OfTypename != nil ||
		len(stmt.Constraints) > 0 || len(stmt.Options) > 0 || stmt.Tablespacename != "" || stmt.AccessMethod != "" ||
		stmt.Oncommit != pg.OnCommitAction_ONCOMMIT_NOOP {
		fail("create table option")
	}
	head := "create " + persistence(stmt.Relation) + "table "
	if stmt.IfNotExists {
		head += "if not exists "
	}
	head = mark(stmt.Relation.Location) + head + p.rangeVar(stmt.Relation)
	if len(stmt.TableElts) == 0 {
		return head + " ()"
	}
	elements := make([]string, len(stmt.TableElts))
	for index, node := range stmt.TableElts {
		switch {
		case node.GetColumnDef() != nil:
			elements[index] = p.columnDef(node.GetColumnDef())
		case node.GetConstraint() != nil:
			elements[index] = mark(node.GetConstraint().Location) + p.constraint(node.GetConstraint())
		default:
			fail("table element %T", node.Node)
		}
	}
	return head + " (\n" + indent(joinLeading(elements)) + "\n)"
}

func (p *printer) columnDef(col *pg.ColumnDef) string {
	if col.Compression != "" || col.Storage != "" || col.StorageName != "" || len(col.Fdwoptions) > 0 ||
		col.RawDefault != nil || col.Identity != "" || col.Generated != "" {
		fail("column option")
	}
	text := mark(col.Location) + ident(col.Colname)
	if col.TypeName != nil {
		text += " " + p.typeName(col.TypeName)
	}
	if col.CollClause != nil {
		text += " collate " + qualifiedName(col.CollClause.Collname, ident)
	}
	for _, node := range col.Constraints {
		text += " " + p.constraint(node.GetConstraint())
	}
	return text
}

var fkActions = map[string]string{
	"r": "restrict",
	"c": "cascade",
	"n": "set null",
	"d": "set default",
}

// learn: keys below is a closure assigned with := and called later as keys(). it reads con from the
// learn: surrounding function.
func (p *printer) constraint(con *pg.Constraint) string {
	if len(con.Including) > 0 || len(con.Options) > 0 || con.Indexname != "" || con.Indexspace != "" ||
		len(con.FkDelSetCols) > 0 || con.AccessMethod != "" || con.WhereClause != nil {
		fail("constraint option")
	}
	text := ""
	if con.Conname != "" {
		text = "constraint " + ident(con.Conname) + " "
	}
	keys := func() string {
		if len(con.Keys) == 0 {
			return ""
		}
		return " " + nameList(con.Keys)
	}
	switch con.Contype {
	case pg.ConstrType_CONSTR_NULL:
		text += "null"
	case pg.ConstrType_CONSTR_NOTNULL:
		text += "not null"
	case pg.ConstrType_CONSTR_DEFAULT:
		text += "default " + p.restricted(con.RawExpr)
	case pg.ConstrType_CONSTR_CHECK:
		text += "check (" + p.expr(con.RawExpr) + ")"
		if con.IsNoInherit {
			text += " no inherit"
		}
	case pg.ConstrType_CONSTR_PRIMARY:
		text += "primary key" + keys()
	case pg.ConstrType_CONSTR_UNIQUE:
		text += "unique"
		if con.NullsNotDistinct {
			text += " nulls not distinct"
		}
		text += keys()
	case pg.ConstrType_CONSTR_FOREIGN:
		if len(con.FkAttrs) > 0 {
			text += "foreign key " + nameList(con.FkAttrs) + " "
		}
		text += "references " + p.rangeVar(con.Pktable)
		if len(con.PkAttrs) > 0 {
			text += " " + nameList(con.PkAttrs)
		}
		switch con.FkMatchtype {
		case "f":
			text += " match full"
		case "p":
			text += " match partial"
		}
		if action := fkActions[con.FkUpdAction]; action != "" {
			text += " on update " + action
		}
		if action := fkActions[con.FkDelAction]; action != "" {
			text += " on delete " + action
		}
	case pg.ConstrType_CONSTR_IDENTITY:
		when := "by default"
		if con.GeneratedWhen == "a" {
			when = "always"
		}
		text += "generated " + when + " as identity"
	case pg.ConstrType_CONSTR_GENERATED:
		text += "generated always as (" + p.expr(con.RawExpr) + ") stored"
	case pg.ConstrType_CONSTR_ATTR_DEFERRABLE:
		text += "deferrable"
	case pg.ConstrType_CONSTR_ATTR_NOT_DEFERRABLE:
		text += "not deferrable"
	case pg.ConstrType_CONSTR_ATTR_DEFERRED:
		text += "initially deferred"
	case pg.ConstrType_CONSTR_ATTR_IMMEDIATE:
		text += "initially immediate"
	default:
		fail("constraint type %s", con.Contype)
	}
	if con.Deferrable {
		text += " deferrable"
	}
	if con.Initdeferred {
		text += " initially deferred"
	}
	if con.SkipValidation {
		text += " not valid"
	}
	return text
}

func behavior(value pg.DropBehavior) string {
	if value == pg.DropBehavior_DROP_CASCADE {
		return " cascade"
	}
	return ""
}

// learn: one action stays on the alter table line; several are indented one per line with commas,
// learn: the same rule clause() applies to select lists.
func (p *printer) alterTableStmt(stmt *pg.AlterTableStmt) string {
	if stmt.Objtype != pg.ObjectType_OBJECT_TABLE {
		fail("alter %s", stmt.Objtype)
	}
	head := "alter table "
	if stmt.MissingOk {
		head += "if exists "
	}
	head = mark(stmt.Relation.Location) + head + p.rangeVar(stmt.Relation)
	cmds := make([]string, len(stmt.Cmds))
	for index, node := range stmt.Cmds {
		cmds[index] = p.alterTableCmd(node.GetAlterTableCmd())
	}
	if len(cmds) == 1 {
		return head + " " + cmds[0]
	}
	return head + "\n" + indent(joinLeading(cmds))
}

func (p *printer) alterTableCmd(cmd *pg.AlterTableCmd) string {
	column := func() string { return "alter column " + ident(cmd.Name) }
	switch cmd.Subtype {
	case pg.AlterTableType_AT_AddColumn:
		text := "add column "
		if cmd.MissingOk {
			text += "if not exists "
		}
		return text + p.columnDef(cmd.Def.GetColumnDef())
	case pg.AlterTableType_AT_DropColumn:
		text := "drop column "
		if cmd.MissingOk {
			text += "if exists "
		}
		return text + ident(cmd.Name) + behavior(cmd.Behavior)
	case pg.AlterTableType_AT_AlterColumnType:
		def := cmd.Def.GetColumnDef()
		text := column() + " type " + p.typeName(def.TypeName)
		if def.CollClause != nil {
			text += " collate " + qualifiedName(def.CollClause.Collname, ident)
		}
		if def.RawDefault != nil {
			text += " using " + p.expr(def.RawDefault)
		}
		return text
	case pg.AlterTableType_AT_SetNotNull:
		return column() + " set not null"
	case pg.AlterTableType_AT_DropNotNull:
		return column() + " drop not null"
	case pg.AlterTableType_AT_ColumnDefault:
		if cmd.Def == nil {
			return column() + " drop default"
		}
		return column() + " set default " + p.restricted(cmd.Def)
	case pg.AlterTableType_AT_AddConstraint:
		return "add " + p.constraint(cmd.Def.GetConstraint())
	case pg.AlterTableType_AT_DropConstraint:
		text := "drop constraint "
		if cmd.MissingOk {
			text += "if exists "
		}
		return text + ident(cmd.Name) + behavior(cmd.Behavior)
	}
	fail("alter table command %s", cmd.Subtype)
	return ""
}

func (p *printer) indexElems(nodes []*pg.Node) []string {
	parts := make([]string, len(nodes))
	for index, node := range nodes {
		elem := node.GetIndexElem()
		if len(elem.Opclassopts) > 0 || elem.Indexcolname != "" {
			fail("index element option")
		}
		text := ""
		switch {
		case elem.Name != "":
			text = ident(elem.Name)
		case elem.Expr.GetFuncCall() != nil:
			text = p.expr(elem.Expr)
		default:
			text = "(" + p.expr(elem.Expr) + ")"
		}
		if len(elem.Collation) > 0 {
			text += " collate " + qualifiedName(elem.Collation, ident)
		}
		if len(elem.Opclass) > 0 {
			text += " " + qualifiedName(elem.Opclass, ident)
		}
		switch elem.Ordering {
		case pg.SortByDir_SORTBY_ASC:
			text += " asc"
		case pg.SortByDir_SORTBY_DESC:
			text += " desc"
		}
		switch elem.NullsOrdering {
		case pg.SortByNulls_SORTBY_NULLS_FIRST:
			text += " nulls first"
		case pg.SortByNulls_SORTBY_NULLS_LAST:
			text += " nulls last"
		}
		parts[index] = text
	}
	return parts
}

func (p *printer) indexStmt(stmt *pg.IndexStmt) string {
	if len(stmt.Options) > 0 || stmt.TableSpace != "" || len(stmt.ExcludeOpNames) > 0 || stmt.Primary || stmt.Isconstraint {
		fail("index option")
	}
	head := "create "
	if stmt.Unique {
		head += "unique "
	}
	head += "index "
	if stmt.Concurrent {
		head += "concurrently "
	}
	if stmt.IfNotExists {
		head += "if not exists "
	}
	if stmt.Idxname != "" {
		head += ident(stmt.Idxname)
	}
	head = mark(stmt.Relation.Location) + strings.TrimSuffix(head, " ")
	target := "on " + p.rangeVar(stmt.Relation)
	if stmt.AccessMethod != "btree" {
		target += " using " + ident(stmt.AccessMethod)
	}
	target += " " + list("(", p.indexElems(stmt.IndexParams), ")")
	if len(stmt.IndexIncludingParams) > 0 {
		target += " include " + list("(", p.indexElems(stmt.IndexIncludingParams), ")")
	}
	if stmt.NullsNotDistinct {
		target += " nulls not distinct"
	}
	if stmt.WhereClause == nil {
		return head + " " + target
	}
	return head + "\n" + indent(target+"\n"+markNode(stmt.WhereClause)+"where "+p.expr(stmt.WhereClause))
}

func (p *printer) transactionStmt(stmt *pg.TransactionStmt) string {
	if len(stmt.Options) > 0 || stmt.Chain || stmt.Gid != "" {
		fail("transaction option")
	}
	switch stmt.Kind {
	case pg.TransactionStmtKind_TRANS_STMT_BEGIN:
		return "begin"
	case pg.TransactionStmtKind_TRANS_STMT_START:
		return "start transaction"
	case pg.TransactionStmtKind_TRANS_STMT_COMMIT:
		return "commit"
	case pg.TransactionStmtKind_TRANS_STMT_ROLLBACK:
		return "rollback"
	case pg.TransactionStmtKind_TRANS_STMT_SAVEPOINT:
		return "savepoint " + ident(stmt.SavepointName)
	case pg.TransactionStmtKind_TRANS_STMT_RELEASE:
		return "release savepoint " + ident(stmt.SavepointName)
	case pg.TransactionStmtKind_TRANS_STMT_ROLLBACK_TO:
		return "rollback to savepoint " + ident(stmt.SavepointName)
	}
	fail("transaction %s", stmt.Kind)
	return ""
}

func (p *printer) createFunctionStmt(stmt *pg.CreateFunctionStmt) string {
	if stmt.SqlBody != nil {
		fail("sql function body")
	}
	head := "create "
	if stmt.Replace {
		head += "or replace "
	}
	if stmt.IsProcedure {
		head += "procedure "
	} else {
		head += "function "
	}
	params := make([]string, len(stmt.Parameters))
	for index, node := range stmt.Parameters {
		param := node.GetFunctionParameter()
		text := map[pg.FunctionParameterMode]string{
			pg.FunctionParameterMode_FUNC_PARAM_IN:       "in ",
			pg.FunctionParameterMode_FUNC_PARAM_OUT:      "out ",
			pg.FunctionParameterMode_FUNC_PARAM_INOUT:    "inout ",
			pg.FunctionParameterMode_FUNC_PARAM_VARIADIC: "variadic ",
			pg.FunctionParameterMode_FUNC_PARAM_DEFAULT:  "",
		}[param.Mode]
		if param.Mode == pg.FunctionParameterMode_FUNC_PARAM_TABLE {
			fail("returns table")
		}
		if param.Name != "" {
			text += ident(param.Name) + " "
		}
		text += p.typeName(param.ArgType)
		if param.Defexpr != nil {
			text += " default " + p.expr(param.Defexpr)
		}
		params[index] = text
	}
	lines := []string{head + list(qualifiedName(stmt.Funcname, funcIdent)+"(", params, ")")}
	if stmt.ReturnType != nil {
		lines = append(lines, indentUnit+"returns "+p.typeName(stmt.ReturnType))
	}
	language := ""
	for _, node := range stmt.Options {
		if opt := node.GetDefElem(); opt.Defname == "language" && opt.Arg.GetString_() != nil {
			language = opt.Arg.GetString_().Sval
		}
	}
	// options are indented under the signature; the body comes last at column 0
	// so its closing delimiter lines up with as
	body := ""
	for _, node := range stmt.Options {
		opt := node.GetDefElem()
		if opt.Defname == "as" {
			body = p.functionOption(opt, language)
			continue
		}
		lines = append(lines, indentUnit+p.functionOption(opt, language))
	}
	if body != "" {
		lines = append(lines, body)
	}
	return strings.Join(lines, "\n")
}

// learn: the flag, word, and number closures validate an option's argument. fail() never returns,
// learn: but go cannot know that, so a return statement is still required after it.
func (p *printer) functionOption(opt *pg.DefElem, language string) string {
	flag := func() bool {
		if opt.Arg == nil || opt.Arg.GetBoolean() == nil {
			fail("function option %s", opt.Defname)
		}
		return opt.Arg.GetBoolean().Boolval
	}
	word := func() string {
		if opt.Arg == nil || opt.Arg.GetString_() == nil {
			fail("function option %s", opt.Defname)
		}
		return opt.Arg.GetString_().Sval
	}
	number := func() string {
		switch {
		case opt.Arg.GetInteger() != nil:
			return strconv.Itoa(int(opt.Arg.GetInteger().Ival))
		case opt.Arg.GetFloat() != nil:
			return opt.Arg.GetFloat().Fval
		}
		fail("function option %s", opt.Defname)
		return ""
	}
	switch opt.Defname {
	case "as":
		list := opt.Arg.GetList()
		if list == nil || len(list.Items) != 1 {
			fail("function body form")
		}
		body, ok := p.src.stringAfter(opt.Location)
		if !ok {
			fail("function body")
		}
		// sql and plpgsql fold case like sql does; other languages such as plpython must stay as written
		switch language {
		case "sql":
			body = formatSQLBody(body)
		case "plpgsql":
			body = formatPlpgsqlBody(body)
		}
		return "as " + strings.ReplaceAll(body, "\n", literalNewline)
	case "language":
		return "language " + ident(word())
	case "volatility":
		return word()
	case "strict":
		if flag() {
			return "strict"
		}
		return "called on null input"
	case "security":
		if flag() {
			return "security definer"
		}
		return "security invoker"
	case "leakproof":
		if flag() {
			return "leakproof"
		}
		return "not leakproof"
	case "window":
		return "window"
	case "parallel":
		return "parallel " + word()
	case "cost":
		return "cost " + number()
	case "rows":
		return "rows " + number()
	}
	fail("function option %s", opt.Defname)
	return ""
}

func (p *printer) viewStmt(stmt *pg.ViewStmt) string {
	if len(stmt.Options) > 0 || stmt.WithCheckOption != pg.ViewCheckOption_NO_CHECK_OPTION {
		fail("view option")
	}
	head := "create "
	if stmt.Replace {
		head += "or replace "
	}
	head += persistence(stmt.View) + "view " + p.rangeVar(stmt.View)
	if len(stmt.Aliases) > 0 {
		head += " " + nameList(stmt.Aliases)
	}
	return mark(stmt.View.Location) + head + " as\n" + p.statement(stmt.Query)
}

// learn: map keys can be any comparable type, here a generated enum.
var dropTypes = map[pg.ObjectType]string{
	pg.ObjectType_OBJECT_TABLE:     "table",
	pg.ObjectType_OBJECT_INDEX:     "index",
	pg.ObjectType_OBJECT_VIEW:      "view",
	pg.ObjectType_OBJECT_SEQUENCE:  "sequence",
	pg.ObjectType_OBJECT_MATVIEW:   "materialized view",
	pg.ObjectType_OBJECT_SCHEMA:    "schema",
	pg.ObjectType_OBJECT_EXTENSION: "extension",
	pg.ObjectType_OBJECT_TYPE:      "type",
}

func (p *printer) dropStmt(stmt *pg.DropStmt) string {
	kind, ok := dropTypes[stmt.RemoveType]
	if !ok {
		fail("drop %s", stmt.RemoveType)
	}
	text := "drop " + kind
	if stmt.Concurrent {
		text += " concurrently"
	}
	if stmt.MissingOk {
		text += " if exists"
	}
	names := make([]string, len(stmt.Objects))
	for index, node := range stmt.Objects {
		switch {
		case node.GetList() != nil:
			names[index] = qualifiedName(node.GetList().Items, ident)
		case node.GetString_() != nil:
			names[index] = ident(node.GetString_().Sval)
		case node.GetTypeName() != nil:
			names[index] = p.typeName(node.GetTypeName())
		default:
			fail("drop object %T", node.Node)
		}
	}
	if len(names) > maxInlineItems {
		return text + "\n" + indent(joinLeading(names)) + behavior(stmt.Behavior)
	}
	return text + " " + strings.Join(names, ", ") + behavior(stmt.Behavior)
}

func (p *printer) renameStmt(stmt *pg.RenameStmt) string {
	exists := ""
	if stmt.MissingOk {
		exists = "if exists "
	}
	switch {
	case stmt.RenameType == pg.ObjectType_OBJECT_TABLE:
		return "alter table " + exists + p.rangeVar(stmt.Relation) + " rename to " + ident(stmt.Newname)
	case stmt.RenameType == pg.ObjectType_OBJECT_COLUMN && stmt.RelationType == pg.ObjectType_OBJECT_TABLE:
		return "alter table " + exists + p.rangeVar(stmt.Relation) + " rename column " + ident(stmt.Subname) + " to " + ident(stmt.Newname)
	case stmt.RenameType == pg.ObjectType_OBJECT_TABCONSTRAINT:
		return "alter table " + exists + p.rangeVar(stmt.Relation) + " rename constraint " + ident(stmt.Subname) + " to " + ident(stmt.Newname)
	case stmt.RenameType == pg.ObjectType_OBJECT_INDEX:
		return "alter index " + exists + p.rangeVar(stmt.Relation) + " rename to " + ident(stmt.Newname)
	}
	fail("rename %s", stmt.RenameType)
	return ""
}

func (p *printer) createExtensionStmt(stmt *pg.CreateExtensionStmt) string {
	text := "create extension "
	if stmt.IfNotExists {
		text += "if not exists "
	}
	text += ident(stmt.Extname)
	for _, node := range stmt.Options {
		opt := node.GetDefElem()
		switch {
		case opt.Defname == "schema" && opt.Arg.GetString_() != nil:
			text += " schema " + ident(opt.Arg.GetString_().Sval)
		case opt.Defname == "cascade":
			text += " cascade"
		default:
			fail("extension option %s", opt.Defname)
		}
	}
	return text
}
