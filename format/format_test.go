// learn: files ending in _test.go only compile under `go test`. being in package format, not
// learn: format_test, lets the tests call unexported functions like isComment.
package format

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	pg "github.com/pganalyze/pg_query_go/v6"
)

// learn: t.Helper() marks this as a helper, so a failure reports the caller's line instead of this
// learn: one. t.Fatal stops the current test immediately.
func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func caseDirs(t *testing.T) []string {
	t.Helper()
	dirs, err := filepath.Glob("../testdata/cases/*")
	if err != nil || len(dirs) == 0 {
		t.Fatalf("no test cases found: %v", err)
	}
	return dirs
}

func formatOrFail(t *testing.T, src string) *Result {
	t.Helper()
	result, err := FormatResult(src)
	if err != nil {
		t.Fatalf("format failed: %v", err)
	}
	return result
}

func assertOutput(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("output mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

// each case's input must format to its expected file
// learn: go test runs every function named TestXxx(t *testing.T). t.Run starts a named subtest, so
// learn: `go test ./format -run TestCases/03-joins` runs just one case. t.Errorf records a failure
// learn: and keeps going; t.Fatalf records it and stops that test.
func TestCases(t *testing.T) {
	for _, dir := range caseDirs(t) {
		t.Run(filepath.Base(dir), func(t *testing.T) {
			result := formatOrFail(t, readFile(t, filepath.Join(dir, "input.sql")))
			for _, skipped := range result.Skipped {
				t.Errorf("line %d left unformatted: %s", skipped.Line, skipped.Reason)
			}
			assertOutput(t, result.Output, readFile(t, filepath.Join(dir, "expected.sql")))
		})
	}
}

// formatting already formatted sql must not change it
func TestCasesAreStable(t *testing.T) {
	for _, dir := range caseDirs(t) {
		t.Run(filepath.Base(dir), func(t *testing.T) {
			expected := readFile(t, filepath.Join(dir, "expected.sql"))
			assertOutput(t, formatOrFail(t, expected).Output, expected)
		})
	}
}

func countComments(t *testing.T, src string) int {
	t.Helper()
	scan, err := pg.Scan(src)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, token := range scan.Tokens {
		if isComment(token) {
			count++
		}
	}
	return count
}

// no comment survives formatting
func TestNoCommentsInOutput(t *testing.T) {
	inputs := []string{
		"-- only a comment\n",
		"select 1; -- one\n-- two\nselect 2 /* three */;\n-- four\n",
		"select a, -- first\n  b -- second\nfrom t -- third\n-- fourth\n;\n",
		"select 1 /* multi\n   line */ from t;\n",
	}
	for _, dir := range caseDirs(t) {
		inputs = append(inputs, readFile(t, filepath.Join(dir, "input.sql")))
	}
	for _, input := range inputs {
		output := formatOrFail(t, input).Output
		if got := countComments(t, output); got != 0 {
			t.Errorf("output has %d comments\ninput:\n%s\noutput:\n%s", got, input, output)
		}
	}
}

// learn: a table-driven test: a slice of anonymous structs, one per case, then one loop that checks
// learn: them all. this is the usual style for go tests.
func TestPlaceholders(t *testing.T) {
	tests := []struct{ name, input, want string }{
		{"cast after placeholder", "select :id::int", "select :id::integer\n"},
		{"kebab case", "select * from t where a = :user-id", "select *\nfrom t\nwhere a = :user-id\n"},
		{"snake case", "select :example_key, :example_key::int, :example_key_2 from t", "select :example_key, :example_key::integer, :example_key_2\nfrom t\n"},
		{"snake case in a list", "select * from t where id in (:example_keys)", "select *\nfrom t\nwhere id in (:example_keys)\n"},
		{"kebab case with a number", "select :address-2", "select :address-2\n"},
		{"minus needs spaces", "select :limit - 1", "select :limit - 1\n"},
		{"inside a string", "select ':not_a_param'", "select ':not_a_param'\n"},
		{"inside a comment", "-- uses :id\nselect :id", "select :id\n"},
		{"array slice", "select tags[1:2] from t", "select tags[1:2]\nfrom t\n"},
		{"repeated name", "select :a, :a", "select :a, :a\n"},
		{"mixed with numbered params", "select $1, :name", "select $1, :name\n"},
		{"keyword as name", "select * from t limit :limit offset :offset", "select *\nfrom t\nlimit :limit\noffset :offset\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertOutput(t, formatOrFail(t, test.input).Output, test.want)
		})
	}
}

func TestStatementsWithoutSemicolons(t *testing.T) {
	assertOutput(t, formatOrFail(t, "SELECT id FROM users").Output, "select id\nfrom users\n")
}

func TestCommentLinesLeaveNoBlankLines(t *testing.T) {
	input := "-- file header\n\n-- query comment\nselect 1;\n\n\n-- footer\nselect 2;\n-- end\n"
	want := "select 1;\n\nselect 2;\n"
	assertOutput(t, formatOrFail(t, input).Output, want)
}

func TestUnsupportedStatementIsKept(t *testing.T) {
	input := "SELECT a FROM t;\nVACUUM   ANALYZE t;\nselect b from t\n"
	result := formatOrFail(t, input)
	assertOutput(t, result.Output, "select a\nfrom t;\n\nvacuum analyze t;\n\nselect b\nfrom t\n")
	if len(result.Skipped) != 1 || result.Skipped[0].Line != 2 {
		t.Errorf("skipped = %+v, want one statement on line 2", result.Skipped)
	}
}

func TestUnsupportedExpressionIsKept(t *testing.T) {
	input := "select x from t where name similar to 'a%' -- removed\n;"
	result := formatOrFail(t, input)
	assertOutput(t, result.Output, "select x from t where name similar to 'a%';\n")
	if len(result.Skipped) != 1 {
		t.Errorf("skipped = %+v, want one", result.Skipped)
	}
}

func TestSyntaxError(t *testing.T) {
	_, err := Format("select 1;\nselect from where;\n")
	var syntax *SyntaxError
	if !errors.As(err, &syntax) {
		t.Fatalf("error = %v, want a syntax error", err)
	}
	if syntax.Line != 2 {
		t.Errorf("line = %d, want 2", syntax.Line)
	}
}

func TestPrecedenceIsKept(t *testing.T) {
	tests := []struct{ input, want string }{
		{"select a - (b - c)", "select a - (b - c)\n"},
		{"select (a + b) * c", "select (a + b) * c\n"},
		{"select a * b + c", "select a * b + c\n"},
		{"select (-1)::int", "select (-1)::integer\n"},
		{"select - (-x)", "select -(-x)\n"},
		{"select not (a = b)", "select not (a = b)\n"},
		{"select (a or b) and c", "select (a or b) and c\n"},
		{"select a or b and c", "select a or (b and c)\n"},
		{"select (a = b) is true", "select a = b is true\n"},
		{"select a = (b is true)", "select a = (b is true)\n"},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			assertOutput(t, formatOrFail(t, test.input).Output, test.want)
		})
	}
}

func TestQuotedNames(t *testing.T) {
	tests := []struct{ input, want string }{
		{`select "user".id from "user"`, "select \"user\".id\nfrom \"user\"\n"},
		{`select "left"("x"), "left".id from "left"`, "select left(x), \"left\".id\nfrom \"left\"\n"},
		{`select 1 as "order", 2 as "Two"`, "select 1 as order, 2 as \"Two\"\n"},
		{`select "Mixed" from t`, "select \"Mixed\"\nfrom t\n"},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			assertOutput(t, formatOrFail(t, test.input).Output, test.want)
		})
	}
}

func TestStringLiteralsAreUntouched(t *testing.T) {
	input := "select E'a\\nb', $tag$it's\n  raw$tag$, 'multi\n  line' from t"
	output := formatOrFail(t, input).Output
	for _, literal := range []string{`E'a\nb'`, "$tag$it's\n  raw$tag$", "'multi\n  line'"} {
		if !strings.Contains(output, literal) {
			t.Errorf("output lost literal %q:\n%s", literal, output)
		}
	}
}

func TestRestrictedSyntax(t *testing.T) {
	tests := []struct{ input, want string }{
		{"select (comp).a, (t.comp).b from t", "select (comp).a, (t.comp).b\nfrom t\n"},
		{`select "coalesce"(1), "extract"('a', b)`, "select \"coalesce\"(1), \"extract\"('a', b)\n"},
		{"create table t (a timestamp default (now() at time zone 'utc'), b int default -1)",
			"create table t (\n  a timestamp default (now() at time zone 'utc'),\n  b integer   default -1\n)\n"},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := formatOrFail(t, test.input)
			if len(result.Skipped) > 0 {
				t.Errorf("skipped: %+v", result.Skipped)
			}
			assertOutput(t, result.Output, test.want)
		})
	}
}

func TestFunctionBodyCase(t *testing.T) {
	tests := []struct{ name, input, want string }{
		{"plpgsql is lowercased and indented",
			"create function f() returns trigger language plpgsql as $fn$\nBEGIN\n  RAISE NOTICE 'Keep %', NEW.Id;\n  RETURN \"Quoted\";\nEND;\n$fn$",
			"create function f()\n  returns trigger\n  language plpgsql\nas $fn$\n  begin\n    raise notice 'Keep %', new.id;\n    return \"Quoted\";\n  end;\n$fn$\n"},
		{"shared indentation is replaced, nesting and strings are kept",
			"create function f() returns int as $$\n      BEGIN\n          RAISE NOTICE 'two\n   lines';\n\n          RETURN 1;\n      END;\n$$ language plpgsql",
			"create function f()\n  returns integer\n  language plpgsql\nas $$\n  begin\n      raise notice 'two\n   lines';\n\n      return 1;\n  end;\n$$\n"},
		{"single line sql is lowercased",
			"create function f() returns int language sql as $$ SELECT 1 $$",
			"create function f()\n  returns integer\n  language sql\nas $$ select 1 $$\n"},
		{"other languages are kept",
			"create function f() returns int language plpython3u as $$\nReturn 1\n$$",
			"create function f()\n  returns integer\n  language plpython3u\nas $$\nReturn 1\n$$\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := formatOrFail(t, test.input)
			if len(result.Skipped) > 0 {
				t.Errorf("skipped: %+v", result.Skipped)
			}
			assertOutput(t, result.Output, test.want)
		})
	}
}

func TestUnsupportedStatementCleanup(t *testing.T) {
	input := "COPY   Users (Id , Name)  FROM STDIN   WITH (FORMAT csv)\n   WHERE note = 'Keep  This' AND id = :User-Id ;"
	want := "copy users (id, name) from stdin with (format csv)\n   where note = 'Keep  This' and id = :User-Id;\n"
	result := formatOrFail(t, input)
	if len(result.Skipped) != 1 {
		t.Errorf("skipped = %+v, want one", result.Skipped)
	}
	assertOutput(t, result.Output, want)
}

func TestStrayCommas(t *testing.T) {
	tests := []struct {
		name, input, want string
		fixes             int
	}{
		{"trailing in select", "select a, b, from t", "select a, b\nfrom t\n", 1},
		{"trailing with leading commas", "select\n  a\n  , b\n  ,\nfrom t", "select a, b\nfrom t\n", 1},
		{"trailing in in list", "select a from t where id in (1, 2,)", "select a\nfrom t\nwhere id in (1, 2)\n", 1},
		{"trailing in insert", "insert into t (a, b,) values (1, 2,)", "insert into t (a, b)\nvalues (1, 2)\n", 2},
		{"trailing in function call", "select f(a, b,) from t", "select f(a, b)\nfrom t\n", 1},
		{"trailing in create table", "create table t (a int,)", "create table t (\n  a integer\n)\n", 1},
		{"trailing before order by", "select a from t group by a, order by a", "select a\nfrom t\ngroup by a\norder by a\n", 1},
		{"trailing before semicolon", "select a, b,;", "select a, b;\n", 1},
		{"doubled", "select a,, b from t", "select a, b\nfrom t\n", 1},
		{"leading", "select , a from t", "select a\nfrom t\n", 1},
		{"comma in a string is untouched", "select 'a,', b, from t", "select 'a,', b\nfrom t\n", 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := formatOrFail(t, test.input)
			assertOutput(t, result.Output, test.want)
			if len(result.Fixed) != test.fixes {
				t.Errorf("fixed %d commas, want %d: %+v", len(result.Fixed), test.fixes, result.Fixed)
			}
		})
	}
}

// a missing comma is a real error, and fixes found before it are still reported
func TestUnfixableErrorKeepsFixes(t *testing.T) {
	result, err := FormatResult("select a, from t;\nselect id email name from users;")
	var syntax *SyntaxError
	if !errors.As(err, &syntax) {
		t.Fatalf("error = %v, want a syntax error", err)
	}
	if syntax.Line != 2 || syntax.Column != 17 {
		t.Errorf("error at %d:%d, want 2:17", syntax.Line, syntax.Column)
	}
	if result == nil || len(result.Fixed) != 1 || result.Fixed[0].Line != 1 || result.Fixed[0].Column != 9 {
		t.Errorf("fixes = %+v, want one at 1:9", result)
	}
}

// columns count characters, not bytes
func TestSyntaxErrorColumnCountsCharacters(t *testing.T) {
	_, err := Format("select 'héllo' frm t")
	var syntax *SyntaxError
	if !errors.As(err, &syntax) {
		t.Fatalf("error = %v, want a syntax error", err)
	}
	if syntax.Column != 20 {
		t.Errorf("column = %d, want 20", syntax.Column)
	}
}

func TestPlaceholderReadings(t *testing.T) {
	tests := []struct{ name, input, want string }{
		{"value", "select * from t where id = :id", "select *\nfrom t\nwhere id = :id\n"},
		{"fragment on its own line", "select * from t\nwhere a = 1\n:filters\norder by a", "select *\nfrom t\nwhere a = 1\n:filters\norder by a\n"},
		{"fragment after a name", "select * from t where active :more", "select *\nfrom t\nwhere active :more\n"},
		{"fragment at the end", "select * from t :where-clause", "select *\nfrom t :where-clause\n"},
		{"name", "select * from :table-name", "select *\nfrom :table-name\n"},
		{"snake case fragment", "select * from t\nwhere a = :example_key\n:example_filter_clause", "select *\nfrom t\nwhere a = :example_key\n:example_filter_clause\n"},
		{"snake case name", "select * from :example_table", "select *\nfrom :example_table\n"},
		{"typed literal on a built-in type stays a value", "select date :day", "select :day::date\n"},
		{"clojure keyword characters", "select * from t where a = :ok? and b = :force!", "select *\nfrom t\nwhere\n  a = :ok?\n  and b = :force!\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := formatOrFail(t, test.input)
			if len(result.Skipped) > 0 {
				t.Errorf("skipped: %+v", result.Skipped)
			}
			assertOutput(t, result.Output, test.want)
		})
	}
}

// error messages name the placeholder, not its stand-in
func TestPlaceholderErrorMessage(t *testing.T) {
	_, err := Format("select * from t where a = 1 :ab")
	var syntax *SyntaxError
	if !errors.As(err, &syntax) {
		t.Fatalf("error = %v, want a syntax error", err)
	}
	if syntax.Message != `syntax error at or near ":ab"` {
		t.Errorf("message = %q", syntax.Message)
	}
}

func TestSelectList(t *testing.T) {
	tests := []struct{ name, input, want string }{
		{"three items stay inline", "select id, name, email from users", "select id, name, email\nfrom users\n"},
		{"four items break", "select id, name, email, phone from users", "select\n  id,\n  name,\n  email,\n  phone\nfrom users\n"},
		{"a multi-line item breaks", "select id, case when a then 1 end as b from t", "select\n  id,\n  case\n    when a then 1\n  end as b\nfrom t\n"},
		{"comments between items are removed", "select id, -- key\n name from t", "select id, name\nfrom t\n"},
		{"distinct", "select distinct a, b from t", "select distinct a, b\nfrom t\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertOutput(t, formatOrFail(t, test.input).Output, test.want)
		})
	}
}

func TestOtherListsFollowTheListsRule(t *testing.T) {
	tests := []struct{ name, input, want string }{
		{"index columns", "create index i on t (a, b, c, d)", "create index i on t (\n  a,\n  b,\n  c,\n  d\n)\n"},
		{"short index columns", "create index i on t (a, b, c)", "create index i on t (a, b, c)\n"},
		{"join using", "select 1 from a inner join b using (w, x, y, z)", "select 1\nfrom a\ninner join b\n  using (\n    w,\n    x,\n    y,\n    z\n  )\n"},
		{"conflict target", "insert into t (a) values (1) on conflict (a, b, c, d) do nothing", "insert into t (a)\nvalues (1)\non conflict (\n  a,\n  b,\n  c,\n  d\n) do nothing\n"},
		{"drop", "drop table a, b, c, d", "drop table\n  a,\n  b,\n  c,\n  d\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertOutput(t, formatOrFail(t, test.input).Output, test.want)
		})
	}
}

func TestCommentsAreRemoved(t *testing.T) {
	tests := []struct{ name, input, want string }{
		{"line and block comments", "-- header\nselect id, -- key\n  email /* contact */ from users; -- trailing\n-- footer\n",
			"select id, email\nfrom users;\n"},
		{"placeholders are kept", "select * from t\nwhere a = :a\n:filters -- note\n", "select *\nfrom t\nwhere a = :a\n:filters\n"},
		{"strings that look like comments are kept", "select '-- not a comment', '/* nor */'", "select '-- not a comment', '/* nor */'\n"},
		{"function bodies", "create function f() returns int language plpgsql as $$\nbegin\n  -- gone\n  return 1; /* gone */\nend;\n$$",
			"create function f()\n  returns integer\n  language plpgsql\nas $$\n  begin\n    return 1;\n  end;\n$$\n"},
		{"unsupported statements", "VACUUM users -- why\n;", "vacuum users;\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertOutput(t, formatOrFail(t, test.input).Output, test.want)
		})
	}
}

// nest builds depth levels of a pattern, where %s marks where the next level goes
func nest(pattern, innermost string, depth int) string {
	text := innermost
	for level := 0; level < depth; level++ {
		text = strings.ReplaceAll(pattern, "%d", strconv.Itoa(level))
		text = strings.Replace(text, "%s", innermost, 1)
		innermost = text
	}
	return text
}

// very deep structures format without being skipped, keep their meaning
// (checked by the formatter itself), and format the same way twice
func TestDeepNesting(t *testing.T) {
	tests := []struct {
		name string
		sql  string
	}{
		{"scalar subqueries", "select " + nest("(select %s from t%d)", "1", 30)},
		{"subqueries in from", "select * from " + nest("(select * from %s) as s%d", "base", 30)},
		{"in subqueries", "select id from t where id in " + nest("(select id from t%d where id in %s)", "(1, 2)", 25)},
		{"boolean groups", "select 1 from t where " + nest("a%d = %d and (b%d = %d or (%s))", "c = 0", 30)},
		{"function calls", "select " + nest("f%d(%s)", "x", 40)},
		{"arithmetic", "select " + nest("((%s + a%d) * 2)", "1", 40)},
		{"case expressions", "select " + nest("case when a%d then %s else %d end", "0", 20)},
		{"ctes in subqueries", "select * from " + nest("(with c%d as (select * from %s) select * from c%d) as s%d", "base", 12)},
		{"long lists inside lists", "select " + nest("coalesce(a%d, b%d, c%d, %s)", "d", 15)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// work that repeats at every level grows exponentially, so a slow run means a regression
			started := time.Now()
			result := formatOrFail(t, test.sql)
			if elapsed := time.Since(started); elapsed > 2*time.Second {
				t.Errorf("formatting took %s", elapsed)
			}
			for _, skipped := range result.Skipped {
				t.Fatalf("left unformatted: %s", skipped.Reason)
			}
			again := formatOrFail(t, result.Output)
			if again.Output != result.Output {
				t.Errorf("formatting twice changed the output")
			}
		})
	}
}

func TestSQLSyntaxFunctions(t *testing.T) {
	input := "select trim(a), trim(both 'x' from a), trim(leading from a), substring(a from 2 for 3), position('b' in a) from t"
	want := "select\n  trim(a),\n  trim(both 'x' from a),\n  trim(leading from a),\n  substring(a from 2 for 3),\n  position('b' in a)\nfrom t\n"
	result := formatOrFail(t, input)
	if len(result.Skipped) > 0 {
		t.Errorf("skipped: %+v", result.Skipped)
	}
	assertOutput(t, result.Output, want)
}
