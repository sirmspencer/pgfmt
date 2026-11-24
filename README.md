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

The style is defined by the input and expected pairs in `testdata/cases`. In summary:

- Keywords, functions, types, and unquoted identifiers are lowercase. Quoted identifiers are not changed.
- Indentation is four spaces.
- A clause with one item stays on one line (`select id`, `where id = :id`). A clause with several items places each item on its own indented line, with trailing commas.
- Joins align with `from`. The `on` condition is indented beneath the join, and additional `and` conditions align with `on`.
- Subqueries and grouped `and`/`or` conditions open a parenthesis at the end of the line and close it on a separate line.
- Statements are separated by one blank line.

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
pgfmt < query.sql          # format stdin to stdout
pgfmt query.sql            # print the formatted file
pgfmt -w migrations/*.sql  # rewrite files in place
pgfmt -check $(git ls-files '*.sql')  # list unformatted files, exit 1 if any
```

Exit status is 0 on success, 1 when `-check` finds unformatted files, and 2 on a syntax error. On a syntax error nothing is written to stdout, so an editor keeps the buffer as it was.

## Editor setup

Each editor runs pgfmt on the buffer through stdin and replaces the buffer with stdout.

### VS Code

Install the Custom Local Formatters extension (`jkillian.custom-local-formatters`), then add to `settings.json`:

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
- Arguments: `-w $FilePath$`
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
