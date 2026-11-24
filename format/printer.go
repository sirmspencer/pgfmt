package format

import (
	"strconv"
	"strings"

	pg "github.com/pganalyze/pg_query_go/v6"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// learn: untyped constants. \x01 to \x03 are control characters that never appear in sql, so they
// learn: are safe as invisible markers inside the text we build. literalNewline stands in for
// learn: newlines inside string literals so indent() cannot change a literal's contents.
const (
	indentUnit     = "  "
	markStart      = "\x01"
	markEnd        = "\x02"
	literalNewline = "\x03"
)

// learn: the printer carries shared state. methods take p *printer so every helper can reach the
// learn: source text and the placeholder names.
type printer struct {
	src    *source
	params *placeholders
}

// mark tags the start of a printed item with its source offset so
// comments can be placed next to the code they were written beside
// learn: go never converts between numeric types implicitly, so int(loc) is required before calling
// learn: strconv.Itoa, which takes an int.
func mark(loc int32) string {
	if loc < 0 {
		return ""
	}
	return markStart + strconv.Itoa(int(loc)) + markEnd
}

func markNode(node proto.Message) string {
	return mark(minLocation(node))
}

// minLocation returns the leftmost source offset found anywhere in node
// learn: protoreflect walks any protobuf message generically, without knowing its concrete type.
// learn: walk is a closure, a function value defined inside another function that can read and
// learn: change best. it is declared with var before it is assigned so that it can call itself
// learn: recursively.
func minLocation(node proto.Message) int32 {
	best := int32(-1)
	var walk func(protoreflect.Message)
	walk = func(msg protoreflect.Message) {
		if field := msg.Descriptor().Fields().ByName("location"); field != nil && field.Kind() == protoreflect.Int32Kind {
			if loc := int32(msg.Get(field).Int()); loc >= 0 && (best < 0 || loc < best) {
				best = loc
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
	walk(node.ProtoReflect())
	return best
}

// indent shifts every non-empty line one level right
// learn: strings.Split returns a slice. assigning to lines[index] changes the slice in place, then
// learn: strings.Join puts it back together.
func indent(text string) string {
	lines := strings.Split(text, "\n")
	for index, line := range lines {
		if line != "" {
			lines[index] = indentUnit + line
		}
	}
	return strings.Join(lines, "\n")
}

// clause prints a keyword with its items: inline for one item, one per line otherwise
// learn: this one helper produces the house style: `select id` for one item, a keyword followed by
// learn: indented lines for several, with the comma leading each line after the first.
func (p *printer) clause(keyword string, items []string) string {
	if len(items) == 1 {
		return keyword + " " + items[0]
	}
	return p.keywordMark(items[0]) + keyword + "\n" + indent(joinLeading(items))
}

// keywordMark tags a keyword printed alone on its line with the position of
// the keyword in the source: the token just before the first item. comments
// written above the clause then stay above it, not inside it.
func (p *printer) keywordMark(first string) string {
	match := markPattern.FindStringSubmatch(first)
	if match == nil {
		return ""
	}
	loc, _ := strconv.Atoi(match[1])
	index, ok := p.src.tokenAt[int32(loc)]
	if !ok {
		return ""
	}
	for index--; index >= 0; index-- {
		if !isComment(p.src.tokens[index]) {
			return mark(p.src.tokens[index].Start)
		}
	}
	return ""
}

// maxInlineItems is the longest parenthesized list that stays on one line
const maxInlineItems = 3

// joinLeading puts each item on its own line, with a comma before every item after the first
func joinLeading(items []string) string {
	return strings.Join(items, "\n, ")
}

// list prints a parenthesized list on one line when it has at most
// maxInlineItems items, and one item per line otherwise
// learn: open and close are passed in so one helper serves "name(" ")", "in (" ")", and "array[" "]".
func list(open string, items []string, close string) string {
	if len(items) <= maxInlineItems {
		return open + strings.Join(items, ", ") + close
	}
	return open + "\n" + indent(joinLeading(items)) + "\n" + close
}

// conditionItems splits a top-level and/or into lines led by their connective
// learn: the parser flattens `a and b and c` into one node with three arguments, so a single loop
// learn: prints each condition on its own line, prefixed with and/or after the first.
func (p *printer) conditionItems(node *pg.Node) []string {
	expr := node.GetBoolExpr()
	if expr == nil || expr.Boolop == pg.BoolExprType_NOT_EXPR {
		return []string{markNode(node) + p.expr(node)}
	}
	word := "and "
	if expr.Boolop == pg.BoolExprType_OR_EXPR {
		word = "or "
	}
	items := make([]string, len(expr.Args))
	for index, arg := range expr.Args {
		text := ""
		if nested := arg.GetBoolExpr(); nested != nil && nested.Boolop != pg.BoolExprType_NOT_EXPR {
			text = "(\n" + indent(strings.Join(p.conditionItems(arg), "\n")) + "\n)"
		} else {
			text = p.operand(arg, precAnd+1)
		}
		if index > 0 {
			text = word + text
		}
		items[index] = markNode(arg) + text
	}
	return items
}

func (p *printer) conditionClause(keyword string, node *pg.Node) string {
	items := p.conditionItems(node)
	if len(items) == 1 {
		return keyword + " " + items[0]
	}
	return p.keywordMark(items[0]) + keyword + "\n" + indent(strings.Join(items, "\n"))
}

// statement prints any supported statement node
// learn: the entry point for printing any statement. each case hands off to a method for that
// learn: statement type; anything not listed calls fail and is left as written.
func (p *printer) statement(node *pg.Node) string {
	switch {
	case node.GetSelectStmt() != nil:
		return p.selectStmt(node.GetSelectStmt())
	case node.GetInsertStmt() != nil:
		return p.insertStmt(node.GetInsertStmt())
	case node.GetUpdateStmt() != nil:
		return p.updateStmt(node.GetUpdateStmt())
	case node.GetDeleteStmt() != nil:
		return p.deleteStmt(node.GetDeleteStmt())
	case node.GetCreateStmt() != nil:
		return p.createStmt(node.GetCreateStmt())
	case node.GetAlterTableStmt() != nil:
		return p.alterTableStmt(node.GetAlterTableStmt())
	case node.GetIndexStmt() != nil:
		return p.indexStmt(node.GetIndexStmt())
	case node.GetTransactionStmt() != nil:
		return p.transactionStmt(node.GetTransactionStmt())
	case node.GetCreateFunctionStmt() != nil:
		return p.createFunctionStmt(node.GetCreateFunctionStmt())
	case node.GetViewStmt() != nil:
		return p.viewStmt(node.GetViewStmt())
	case node.GetDropStmt() != nil:
		return p.dropStmt(node.GetDropStmt())
	case node.GetRenameStmt() != nil:
		return p.renameStmt(node.GetRenameStmt())
	case node.GetCreateExtensionStmt() != nil:
		return p.createExtensionStmt(node.GetCreateExtensionStmt())
	}
	fail("statement %T", node.Node)
	return ""
}

// learn: parts collects the clauses in order. append may need to grow the slice and return a new
// learn: one, which is why it is always written as parts = append(parts, x).
func (p *printer) selectStmt(stmt *pg.SelectStmt) string {
	into := ""
	if stmt.IntoClause != nil {
		// plpgsql uses select ... into variable; plain sql reads it as create table
		clause := stmt.IntoClause
		if clause.Rel == nil || len(clause.ColNames) > 0 || clause.AccessMethod != "" || len(clause.Options) > 0 ||
			clause.OnCommit != pg.OnCommitAction_ONCOMMIT_NOOP || clause.TableSpaceName != "" || clause.ViewQuery != nil ||
			clause.SkipData || stmt.Op != pg.SetOperation_SETOP_NONE {
			fail("select into option")
		}
		into = "into " + persistence(clause.Rel) + p.rangeVar(clause.Rel)
	}
	var parts []string
	if stmt.WithClause != nil {
		parts = append(parts, p.with(stmt.WithClause))
	}
	switch {
	case stmt.Op != pg.SetOperation_SETOP_NONE:
		word := map[pg.SetOperation]string{
			pg.SetOperation_SETOP_UNION:     "union",
			pg.SetOperation_SETOP_INTERSECT: "intersect",
			pg.SetOperation_SETOP_EXCEPT:    "except",
		}[stmt.Op]
		if stmt.All {
			word += " all"
		}
		// the empty strings become blank lines around the set operator
		parts = append(parts, p.setOperand(stmt, stmt.Larg, false), "", word, "", p.setOperand(stmt, stmt.Rarg, true))
	case len(stmt.ValuesLists) > 0:
		parts = append(parts, p.values(stmt.ValuesLists))
	default:
		parts = append(parts, p.selectList(stmt))
		if into != "" {
			parts = append(parts, into)
		}
		if len(stmt.FromClause) > 0 {
			parts = append(parts, p.clause("from", p.fromItems(stmt.FromClause)))
		}
		if stmt.WhereClause != nil {
			parts = append(parts, p.conditionClause("where", stmt.WhereClause))
		}
		if len(stmt.GroupClause) > 0 {
			if stmt.GroupDistinct {
				fail("group by distinct")
			}
			parts = append(parts, p.shortClause("group by", stmt.GroupClause, p.markedExprs(stmt.GroupClause)))
		}
		if stmt.HavingClause != nil {
			parts = append(parts, p.conditionClause("having", stmt.HavingClause))
		}
		if len(stmt.WindowClause) > 0 {
			fail("window clause")
		}
	}
	if len(stmt.SortClause) > 0 {
		items := make([]string, len(stmt.SortClause))
		for index, node := range stmt.SortClause {
			items[index] = markNode(node) + p.sortBy(node)
		}
		parts = append(parts, p.shortClause("order by", stmt.SortClause, items))
	}
	if stmt.LimitOption == pg.LimitOption_LIMIT_OPTION_WITH_TIES {
		fail("fetch with ties")
	}
	if stmt.LimitCount != nil {
		if value := stmt.LimitCount.GetAConst(); value != nil && value.Isnull {
			parts = append(parts, markNode(stmt.LimitCount)+"limit all")
		} else {
			parts = append(parts, markNode(stmt.LimitCount)+"limit "+p.expr(stmt.LimitCount))
		}
	}
	if stmt.LimitOffset != nil {
		parts = append(parts, markNode(stmt.LimitOffset)+"offset "+p.expr(stmt.LimitOffset))
	}
	for _, node := range stmt.LockingClause {
		parts = append(parts, p.locking(node.GetLockingClause()))
	}
	return strings.Join(parts, "\n")
}

// setOperand prints one side of a set operation, refusing shapes that need parentheses
func (p *printer) setOperand(parent, child *pg.SelectStmt, right bool) string {
	if child.WithClause != nil || len(child.SortClause) > 0 || child.LimitCount != nil ||
		child.LimitOffset != nil || len(child.LockingClause) > 0 {
		fail("parenthesized set operand")
	}
	if child.Op != pg.SetOperation_SETOP_NONE {
		if right || (parent.Op == pg.SetOperation_SETOP_INTERSECT && child.Op != pg.SetOperation_SETOP_INTERSECT) {
			fail("nested set operation")
		}
	}
	return p.selectStmt(child)
}

func (p *printer) selectList(stmt *pg.SelectStmt) string {
	keyword := "select"
	if len(stmt.DistinctClause) > 0 {
		if len(stmt.DistinctClause) == 1 && stmt.DistinctClause[0].Node == nil {
			keyword += " distinct"
		} else {
			keyword += " distinct on (" + p.exprList(stmt.DistinctClause) + ")"
		}
	}
	if len(stmt.TargetList) == 0 {
		return keyword
	}
	return p.shortClause(keyword, stmt.TargetList, p.targets(stmt.TargetList))
}

// shortClause prints select, group by, and order by: on the keyword's line
// when inlineItems allows, otherwise one item per line
func (p *printer) shortClause(keyword string, nodes []*pg.Node, items []string) string {
	if p.inlineItems(nodes, items) {
		return keyword + " " + strings.Join(items, ", ")
	}
	return p.clause(keyword, items)
}

// inlineItems reports whether a clause's items stay on the keyword's line: there
// are at most maxInlineItems, none spans several lines, and no comment sits
// between items, since on one line that comment would lose its place
func (p *printer) inlineItems(nodes []*pg.Node, items []string) bool {
	if len(items) < 2 || len(items) > maxInlineItems {
		return len(items) < 2
	}
	for _, item := range items {
		if strings.Contains(item, "\n") {
			return false
		}
	}
	first, last := minLocation(nodes[0]), minLocation(nodes[len(nodes)-1])
	for _, token := range p.src.tokens {
		if isComment(token) && token.Start > first && token.Start < last {
			return false
		}
	}
	return true
}

func (p *printer) targets(nodes []*pg.Node) []string {
	items := make([]string, len(nodes))
	for index, node := range nodes {
		target := node.GetResTarget()
		if len(target.Indirection) > 0 {
			fail("target indirection")
		}
		text := p.expr(target.Val)
		if target.Name != "" {
			text += " as " + labelIdent(target.Name)
		}
		items[index] = markNode(node) + text
	}
	return items
}

func (p *printer) markedExprs(nodes []*pg.Node) []string {
	items := make([]string, len(nodes))
	for index, node := range nodes {
		items[index] = markNode(node) + p.expr(node)
	}
	return items
}

func (p *printer) values(lists []*pg.Node) string {
	rows := make([]string, len(lists))
	for index, node := range lists {
		rows[index] = markNode(node) + list("(", p.exprs(node.GetList().Items), ")")
	}
	return p.clause("values", rows)
}

func (p *printer) with(with *pg.WithClause) string {
	head := "with "
	if with.Recursive {
		head += "recursive "
	}
	ctes := make([]string, len(with.Ctes))
	for index, node := range with.Ctes {
		cte := node.GetCommonTableExpr()
		if cte.SearchClause != nil || cte.CycleClause != nil {
			fail("cte search or cycle")
		}
		text := mark(cte.Location) + ident(cte.Ctename)
		if len(cte.Aliascolnames) > 0 {
			text += " (" + nameList(cte.Aliascolnames) + ")"
		}
		text += " as "
		switch cte.Ctematerialized {
		case pg.CTEMaterialize_CTEMaterializeAlways:
			text += "materialized "
		case pg.CTEMaterialize_CTEMaterializeNever:
			text += "not materialized "
		}
		ctes[index] = text + p.subquery(cte.Ctequery)
	}
	return head + strings.Join(ctes, "\n, ")
}

func (p *printer) locking(lock *pg.LockingClause) string {
	words := map[pg.LockClauseStrength]string{
		pg.LockClauseStrength_LCS_FORKEYSHARE:    "for key share",
		pg.LockClauseStrength_LCS_FORSHARE:       "for share",
		pg.LockClauseStrength_LCS_FORNOKEYUPDATE: "for no key update",
		pg.LockClauseStrength_LCS_FORUPDATE:      "for update",
	}
	text, ok := words[lock.Strength]
	if !ok {
		fail("lock strength %s", lock.Strength)
	}
	if len(lock.LockedRels) > 0 {
		rels := make([]string, len(lock.LockedRels))
		for index, node := range lock.LockedRels {
			rels[index] = p.rangeVar(node.GetRangeVar())
		}
		text += " of " + strings.Join(rels, ", ")
	}
	switch lock.WaitPolicy {
	case pg.LockWaitPolicy_LockWaitSkip:
		text += " skip locked"
	case pg.LockWaitPolicy_LockWaitError:
		text += " nowait"
	}
	return text
}

func (p *printer) fromItems(nodes []*pg.Node) []string {
	items := make([]string, len(nodes))
	for index, node := range nodes {
		items[index] = p.fromItem(node)
	}
	return items
}

func (p *printer) fromItem(node *pg.Node) string {
	switch {
	case node.GetRangeVar() != nil:
		return markNode(node) + p.rangeVar(node.GetRangeVar())
	case node.GetJoinExpr() != nil:
		return p.join(node.GetJoinExpr())
	case node.GetRangeSubselect() != nil:
		sub := node.GetRangeSubselect()
		text := markNode(node)
		if sub.Lateral {
			text += "lateral "
		}
		return text + p.subquery(sub.Subquery) + p.alias(sub.Alias)
	case node.GetRangeFunction() != nil:
		fn := node.GetRangeFunction()
		if fn.IsRowsfrom || len(fn.Coldeflist) > 0 || len(fn.Functions) != 1 {
			fail("function in from")
		}
		items := fn.Functions[0].GetList().Items
		if len(items) > 1 && items[1].Node != nil {
			fail("function column definitions")
		}
		text := markNode(node)
		if fn.Lateral {
			text += "lateral "
		}
		text += p.expr(items[0])
		if fn.Ordinality {
			text += " with ordinality"
		}
		return text + p.alias(fn.Alias)
	}
	fail("from item %T", node.Node)
	return ""
}

// learn: a map literal can be indexed right away: map[...]string{...}[key]. a missing key returns
// learn: "", which we check for. joins are nested left to right in the tree, so printing the left
// learn: side first recursively puts the joins in source order.
func (p *printer) join(join *pg.JoinExpr) string {
	if join.Alias != nil || join.JoinUsingAlias != nil || join.Rarg.GetJoinExpr() != nil {
		fail("parenthesized join")
	}
	word := map[pg.JoinType]string{
		pg.JoinType_JOIN_INNER: "inner join",
		pg.JoinType_JOIN_LEFT:  "left join",
		pg.JoinType_JOIN_RIGHT: "right join",
		pg.JoinType_JOIN_FULL:  "full join",
	}[join.Jointype]
	if word == "" {
		fail("join type %s", join.Jointype)
	}
	switch {
	case join.IsNatural:
		word = "natural " + word
	case join.Jointype == pg.JoinType_JOIN_INNER && join.Quals == nil && len(join.UsingClause) == 0:
		word = "cross join"
	}
	text := p.fromItem(join.Larg) + "\n" + markNode(join.Rarg) + word + " " + p.fromItem(join.Rarg)
	if len(join.UsingClause) > 0 {
		text += "\n" + indent("using ("+nameList(join.UsingClause)+")")
	}
	if join.Quals != nil {
		text += "\n" + indent(p.onClause(join.Quals))
	}
	return text
}

// onClause keeps the extra conditions aligned with on
func (p *printer) onClause(node *pg.Node) string {
	items := p.conditionItems(node)
	items[0] = "on " + items[0]
	return strings.Join(items, "\n")
}

// learn: RangeVar is a table reference. Inh is false when the query said `only`.
func (p *printer) rangeVar(rel *pg.RangeVar) string {
	text := ""
	if !rel.Inh {
		text = "only "
	}
	if rel.Catalogname != "" {
		text += ident(rel.Catalogname) + "."
	}
	if rel.Schemaname != "" {
		text += ident(rel.Schemaname) + "."
	}
	return text + ident(rel.Relname) + p.alias(rel.Alias)
}

func (p *printer) alias(alias *pg.Alias) string {
	if alias == nil {
		return ""
	}
	text := " as " + ident(alias.Aliasname)
	if len(alias.Colnames) > 0 {
		text += " (" + nameList(alias.Colnames) + ")"
	}
	return text
}

func (p *printer) returning(nodes []*pg.Node) string {
	return p.shortClause("returning", nodes, p.targets(nodes))
}

func (p *printer) setItems(nodes []*pg.Node) []string {
	items := make([]string, len(nodes))
	for index, node := range nodes {
		target := node.GetResTarget()
		if len(target.Indirection) > 0 || target.Val.GetMultiAssignRef() != nil {
			fail("set target form")
		}
		items[index] = markNode(node) + ident(target.Name) + " = " + p.expr(target.Val)
	}
	return items
}

// learn: the values list or select inside an insert is an ordinary SelectStmt node, so it reuses
// learn: the same printer as a standalone query.
func (p *printer) insertStmt(stmt *pg.InsertStmt) string {
	if stmt.Override != pg.OverridingKind_OVERRIDING_NOT_SET {
		fail("insert overriding")
	}
	var parts []string
	if stmt.WithClause != nil {
		parts = append(parts, p.with(stmt.WithClause))
	}
	head := "insert into " + p.rangeVar(stmt.Relation)
	if len(stmt.Cols) > 0 {
		cols := make([]string, len(stmt.Cols))
		for index, node := range stmt.Cols {
			target := node.GetResTarget()
			if len(target.Indirection) > 0 {
				fail("insert column indirection")
			}
			cols[index] = ident(target.Name)
		}
		head += " " + list("(", cols, ")")
	}
	parts = append(parts, mark(stmt.Relation.Location)+head)
	if stmt.SelectStmt == nil {
		parts = append(parts, "default values")
	} else {
		parts = append(parts, p.statement(stmt.SelectStmt))
	}
	if conflict := stmt.OnConflictClause; conflict != nil {
		text := "on conflict"
		if infer := conflict.Infer; infer != nil {
			if infer.Conname != "" {
				text += " on constraint " + ident(infer.Conname)
			} else {
				text += " (" + p.indexElems(infer.IndexElems) + ")"
				if infer.WhereClause != nil {
					text += " where " + p.expr(infer.WhereClause)
				}
			}
		}
		switch conflict.Action {
		case pg.OnConflictAction_ONCONFLICT_NOTHING:
			parts = append(parts, mark(conflict.Location)+text+" do nothing")
		case pg.OnConflictAction_ONCONFLICT_UPDATE:
			parts = append(parts, mark(conflict.Location)+text+" do update", p.clause("set", p.setItems(conflict.TargetList)))
			if conflict.WhereClause != nil {
				parts = append(parts, p.conditionClause("where", conflict.WhereClause))
			}
		default:
			fail("on conflict action")
		}
	}
	if len(stmt.ReturningList) > 0 {
		parts = append(parts, p.returning(stmt.ReturningList))
	}
	return strings.Join(parts, "\n")
}

func (p *printer) updateStmt(stmt *pg.UpdateStmt) string {
	var parts []string
	if stmt.WithClause != nil {
		parts = append(parts, p.with(stmt.WithClause))
	}
	parts = append(parts, mark(stmt.Relation.Location)+"update "+p.rangeVar(stmt.Relation))
	parts = append(parts, p.clause("set", p.setItems(stmt.TargetList)))
	if len(stmt.FromClause) > 0 {
		parts = append(parts, p.clause("from", p.fromItems(stmt.FromClause)))
	}
	if stmt.WhereClause != nil {
		parts = append(parts, p.conditionClause("where", stmt.WhereClause))
	}
	if len(stmt.ReturningList) > 0 {
		parts = append(parts, p.returning(stmt.ReturningList))
	}
	return strings.Join(parts, "\n")
}

func (p *printer) deleteStmt(stmt *pg.DeleteStmt) string {
	var parts []string
	if stmt.WithClause != nil {
		parts = append(parts, p.with(stmt.WithClause))
	}
	parts = append(parts, mark(stmt.Relation.Location)+"delete from "+p.rangeVar(stmt.Relation))
	if len(stmt.UsingClause) > 0 {
		parts = append(parts, p.clause("using", p.fromItems(stmt.UsingClause)))
	}
	if stmt.WhereClause != nil {
		parts = append(parts, p.conditionClause("where", stmt.WhereClause))
	}
	if len(stmt.ReturningList) > 0 {
		parts = append(parts, p.returning(stmt.ReturningList))
	}
	return strings.Join(parts, "\n")
}
