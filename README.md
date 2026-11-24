*README generated with LLM assistance*

# pgfmt

pgfmt formats PostgreSQL `.sql` files with a single fixed style, so that every editor and every developer produces the same output. It parses each statement with the PostgreSQL parser (libpg_query, through `pg_query_go`) and prints it again from the syntax tree.

## Guarantees

- **Meaning is preserved.** After formatting a statement, pgfmt parses the result and compares its syntax tree with the original. If the trees differ, the statement is left exactly as written and a warning is printed.
- **Unsupported syntax is left alone.** Statements that use syntax the printer does not handle yet are copied through unchanged and reported on stderr.
- **Comments are kept.** Comments stay next to the item they were written beside.
- **Placeholders are kept.** `:name` placeholders, including kebab-case names such as `:user-id`, are preserved. A colon inside array brackets (`tags[1:2]`), a cast (`::`), a string, or a comment is never treated as a placeholder.
- **Literals are untouched.** String and number literals, including dollar-quoted function bodies, are printed exactly as written.

## Style

The style is defined by the input and expected pairs in `testdata/cases` and documented rule by rule in [rules.md](rules.md). In summary:

- Keywords, functions, types, and unquoted identifiers are lowercase. Quoted identifiers are not changed.
- Indentation is two spaces.
- `select`, `group by`, `order by`, and `returning` stay on one line up to three items. Other clauses stay on one line with one item (`where id = :id`). Longer lists place each item on its own indented line, with a leading comma on each line after the first.
- Parenthesized lists, including function arguments, stay on one line up to three items and are split one item per line from four items.
- Joins align with `from`. The `on` condition is indented beneath the join, and additional `and` conditions align with `on`.
- Subqueries and grouped `and`/`or` conditions open a parenthesis at the end of the line and close it on a separate line.
- Set operators such as `union` have a blank line before and after.
- Statements are separated by one blank line. There is no line length limit.

## Installation

### Homebrew

```sh
brew tap sirmspencer/tap
brew trust sirmspencer/tap
brew install pgfmt
```

The formula builds from source, so the first install takes a minute or two while the PostgreSQL parser compiles.

### Go

pgfmt uses cgo, so a C compiler is required to build it (on macOS, the Xcode command line tools).

```sh
go install github.com/sirmspencer/pgfmt/cmd/pgfmt@latest
```

Run `pgfmt -version` to confirm which build is on the path.

## Usage

```sh
pgfmt < query.sql                     # format stdin to stdout
pgfmt query.sql                       # print the formatted file
pgfmt -check $(git ls-files '*.sql')  # report problems, change nothing
pgfmt -fix migrations/*.sql           # format and repair files in place
```

`-check` reports every problem it finds, one per line, without changing any file:

```
queries.sql: not formatted
queries.sql:3:14: stray comma (fixable)
queries.sql:7:17: syntax error at or near "name"
```

`-fix` formats each file, removes stray commas, and rewrites the file. A file with a syntax error that cannot be fixed is left unchanged, and the error is reported. Reading from stdin, as editors do, formats and repairs the same way; on an error that cannot be fixed nothing is written to stdout, so the editor keeps the buffer as it was.

A stray comma is one that can only be extra: a trailing comma before a closing parenthesis, a semicolon, the end of a statement, or a keyword such as `from`; a doubled comma; or a comma directly after an opening parenthesis or a keyword. A missing comma cannot be repaired and is reported as a syntax error.

| Exit status | `-check` | `-fix` and stdin |
|---|---|---|
| 0 | Nothing to change | Success |
| 1 | Files need formatting or have fixable problems | Not used |
| 2 | A syntax error that cannot be fixed, or a file could not be read | A syntax error that cannot be fixed, or a file could not be read or written |

Statements that use syntax pgfmt does not lay out are reported on stderr in every mode. They do not affect the exit status.

## Editor setup

Each editor runs pgfmt on the buffer through stdin and replaces the buffer with stdout.

### VS Code

A dedicated pgfmt extension is planned under `editors/vscode`. Until it is published, pgfmt runs through [Custom Local Formatters](https://marketplace.visualstudio.com/items?itemName=jkillian.custom-local-formatters) (`jkillian.custom-local-formatters`), a third-party extension that registers any command-line program as a VS Code formatter. It sends the document to the command's stdin and replaces the document with the command's stdout, which is how pgfmt already works.

Install the extension, then add to `settings.json`:

```json
{
  "customLocalFormatters.formatters": [
    { "command": "pgfmt", "languages": ["sql"] }
  ],
  "[sql]": {
    "editor.defaultFormatter": "jkillian.custom-local-formatters",
    "editor.formatOnSave": true
  }
}
```

When the pgfmt extension is published, it replaces this setup.

### Neovim

With conform.nvim:

```lua
require("conform").setup({
  formatters = {
    pgfmt = { command = "pgfmt", stdin = true },
  },
  formatters_by_ft = {
    sql = { "pgfmt" },
  },
  format_on_save = { timeout_ms = 2000 },
})
```

### IntelliJ

Add a File Watcher under Settings, Tools, File Watchers, with these values:

- File type: SQL
- Program: `pgfmt`
- Arguments: `-fix $FilePath$`
- Output paths to refresh: `$FilePath$`

Disable the built-in SQL code style reformatting on save, so that the two formatters do not conflict.

### Emacs

With apheleia:

```elisp
(with-eval-after-load 'apheleia
  (setf (alist-get 'pgfmt apheleia-formatters) '("pgfmt"))
  (setf (alist-get 'sql-mode apheleia-mode-alist) 'pgfmt))
```

## Enforcement

A pre-commit hook can reject unformatted files:

```sh
#!/bin/sh
files=$(git diff --cached --name-only --diff-filter=ACM -- '*.sql')
[ -z "$files" ] || pgfmt -check $files
```

In CI, run `pgfmt -check $(git ls-files '*.sql')`.

## Development

```sh
go test ./...
./bin/link     # build this checkout and put it ahead of the homebrew version
./bin/unlink   # remove the dev build and restore the homebrew version
```

Editor plugins run whichever `pgfmt` is on the path, so `./bin/link` is how a local change is tested in an editor.

Each directory in `testdata/cases` contains an `input.sql` and the `expected.sql` it must format to. The tests also check that formatting `expected.sql` leaves it unchanged and that no comments are lost.

`rules.md` is generated. Edit `docs/rules.template.md`, which contains each rule and its input examples, then run `go run ./tools/rules` to write `rules.md` with the formatter's output below every example. `go test ./...` fails if `rules.md` is out of date.
