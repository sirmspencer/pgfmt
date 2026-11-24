// learn: files ending in _test.go only compile under `go test`. being in package format, not
// learn: format_test, lets the tests call unexported functions like isComment.
package format

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

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

func TestCommentsAreKept(t *testing.T) {
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
		if got, want := countComments(t, output), countComments(t, input); got != want {
			t.Errorf("comment count %d, want %d\ninput:\n%s\noutput:\n%s", got, want, input, output)
		}
	}
}

// learn: a table-driven test: a slice of anonymous structs, one per case, then one loop that checks
// learn: them all. this is the usual style for go tests.
func TestPlaceholders(t *testing.T) {
	tests := []struct{ name, input, want string }{
		{"cast after placeholder", "select :id::int", "select :id::integer\n"},
		{"kebab case", "select * from t where a = :user-id", "select *\nfrom t\nwhere a = :user-id\n"},
		{"kebab case with a number", "select :address-2", "select :address-2\n"},
		{"minus needs spaces", "select :limit - 1", "select :limit - 1\n"},
		{"inside a string", "select ':not_a_param'", "select ':not_a_param'\n"},
		{"inside a comment", "-- uses :id\nselect :id", "-- uses :id\nselect :id\n"},
		{"array slice", "select tags[1:2] from t", "select tags[1:2]\nfrom t\n"},
		{"repeated name", "select :a, :a", "select\n  :a\n  , :a\n"},
		{"mixed with numbered params", "select $1, :name", "select\n  $1\n  , :name\n"},
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

func TestBlankLinesAroundComments(t *testing.T) {
	input := "-- file header\n\n-- query comment\nselect 1;\n\n\n-- footer\n"
	want := "-- file header\n\n-- query comment\nselect 1;\n\n-- footer\n"
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
	input := "select x from t where name similar to 'a%' -- keep\n;"
	result := formatOrFail(t, input)
	assertOutput(t, result.Output, input+"\n")
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
		{`select "left"("x"), "left".id from "left"`, "select\n  left(x)\n  , \"left\".id\nfrom \"left\"\n"},
		{`select 1 as "order", 2 as "Two"`, "select\n  1 as order\n  , 2 as \"Two\"\n"},
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
		{"select (comp).a, (t.comp).b from t", "select\n  (comp).a\n  , (t.comp).b\nfrom t\n"},
		{`select "coalesce"(1), "extract"('a', b)`, "select\n  \"coalesce\"(1)\n  , \"extract\"('a', b)\n"},
		{"create table t (a timestamp default (now() at time zone 'utc'), b int default -1)",
			"create table t (\n  a timestamp default (now() at time zone 'utc')\n  , b integer default -1\n)\n"},
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
