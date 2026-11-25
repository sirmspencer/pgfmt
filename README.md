*README generated with LLM assistance*

# pgfmt

pgfmt formats PostgreSQL `.sql` files with a consistent style, so that every editor and every developer produces the same output. It parses each statement with the PostgreSQL parser (libpg_query, through `pg_query_go`) and prints it again from the syntax tree.

## Behavior

- **Formatted statements are checked.** After formatting a statement, pgfmt parses the result and compares its syntax tree with the original. When the trees differ, the statement is kept as written and reported on stderr.
- **Unsupported syntax keeps its structure.** A statement that uses syntax the printer does not lay out yet keeps its line breaks, with keywords lowercased and extra spaces removed, and is reported on stderr.
- **Comments are removed.** `--` and `/* */` comments are removed, including those inside `sql` and `plpgsql` function bodies, unless the `keepComments` setting is on. Text inside strings is not treated as a comment. A `--;;` line, which migration tools such as migratus use to separate statements, is kept.
- **Placeholders are kept.** `:name` placeholders, including kebab-case and snake_case names such as `:user-id` and `:example_key`, are kept as written. A colon inside array brackets (`tags[1:2]`), a cast (`::`), a string, or a comment is not treated as a placeholder.
- **Stray commas and missing semicolons are repaired.** A comma that can only be extra is removed, and a semicolon is added between two complete statements that have none. Both are reported, and `-check` lists them as fixable.
- **Literals are printed as written.** String and number literals, including dollar-quoted strings, keep their original text.

## Style

The style is defined by the input and expected pairs in `testdata/cases` and documented rule by rule in [rules.md](rules.md). In summary:

- Keywords, functions, types, and unquoted identifiers are lowercase. Quoted identifiers are not changed.
- Indentation is two spaces.
- `select`, `group by`, `order by`, and `returning` stay on one line up to three items. Other clauses stay on one line with one item (`where id = :id`). Longer lists place each item on its own indented line, with a trailing comma after each item but the last.
- Parenthesized lists, including function arguments, stay on one line up to three items and are split one item per line from four items.
- Joins align with `from`. The `on` condition is indented beneath the join, and additional `and` conditions align with `on`.
- Subqueries and grouped `and`/`or` conditions open a parenthesis at the end of the line and close it on a separate line.
- Set operators such as `union` have a blank line before and after.
- Statements are separated by one blank line. There is no line length limit.

## Installation

The editor plugins bundle pgfmt, so an editor needs nothing else. The command line, CI, and other editors use one of these:

### Homebrew

```sh
brew tap sirmspencer/tap
brew trust sirmspencer/tap
brew install pgfmt
```

### npm

```sh
npm install --save-dev @sirmspencer/pgfmt
npx pgfmt -check $(git ls-files '*.sql')
```

The package installs the `pgfmt` command and only the binary for the current platform. Without installing it, `npx @sirmspencer/pgfmt` runs it once.

### Binaries

Every [release](https://github.com/sirmspencer/pgfmt/releases) has archives for macOS (Apple silicon and Intel), Linux (x64 and ARM), and Windows (x64), with a `checksums.txt`.

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
pgfmt -config team.json query.sql     # use a specific settings file
pgfmt -stdin-path db/migrations/001.sql < 001.sql   # name the file read from stdin
```

`-check` reports every problem it finds, one per line, without changing any file:

```
queries.sql: not formatted
queries.sql:3:14: stray comma (fixable)
queries.sql:7:17: syntax error at or near "name"
```

`-fix` formats each file, removes stray commas, and rewrites the file. A file with a syntax error that cannot be fixed is left unchanged, and the error is reported. Reading from stdin, as editors do, formats and repairs the same way; on an error that cannot be fixed nothing is written to stdout, so the editor keeps the buffer as it was.

A missing semicolon is one between two statements on separate lines, as in a scratch file of queries separated by blank lines; a semicolon is never added to a file with a single statement or at the end of a file. A stray comma is one that can only be extra: a trailing comma before a closing parenthesis, a semicolon, the end of a statement, or a keyword such as `from`; a doubled comma; or a comma directly after an opening parenthesis or a keyword. A missing comma cannot be repaired and is reported as a syntax error.

| Exit status | `-check` | `-fix` and stdin |
|---|---|---|
| 0 | Nothing to change | Success |
| 1 | Files need formatting or have fixable problems | Not used |
| 2 | A syntax error that cannot be fixed, or a file could not be read | A syntax error that cannot be fixed, or a file could not be read or written |

Statements that use syntax pgfmt does not lay out are reported on stderr in every mode. They do not affect the exit status.

## Settings

The defaults are the house style described in [rules.md](rules.md). A `.pgfmt.json` file in a repository changes individual settings for every file below it; settings the file does not mention keep their defaults:

```json
{
  "indent": 2,
  "clauseListLimit": 3,
  "parenListLimit": 3,
  "keepComments": false,
  "align": true,
  "blankLinesAroundSetOperators": true,
  "migrationPaths": ["migration"]
}
```

| Setting | Default | Meaning |
|---|---|---|
| `indent` | `2` | Spaces per indentation level, from 1 to 8 |
| `clauseListLimit` | `3` | Most items `select`, `group by`, `order by`, and `returning` keep on the keyword's line; `0` always splits |
| `parenListLimit` | `3` | Most items a parenthesized list, such as function arguments or an `in` list, keeps on one line; `0` always splits |
| `keepComments` | `false` | Keep comments in place instead of removing them |
| `align` | `true` | Align `create table` columns, `set` assignments, and select aliases |
| `blankLinesAroundSetOperators` | `true` | Blank lines before and after `union`, `intersect`, and `except` |
| `migrationPaths` | `["migration"]` | A file whose path contains any entry, ignoring case, is a migration: a `--;;` line separates its statements, and every statement ends with a semicolon |

pgfmt uses the nearest `.pgfmt.json` in the formatted file's directory or a parent directory. Reading from stdin, it uses the path given with `-stdin-path`, which the editor plugins and the configurations below pass, or otherwise the current directory. That path also decides whether the file is a migration. `pgfmt -config file` uses a specific file instead. An unknown setting or an invalid value is an error, so a misspelled setting is reported rather than ignored. [rules.md](rules.md#settings) shows each setting's effect.

## Editor setup

Each editor runs pgfmt on the buffer through stdin and replaces the buffer with stdout.

### VS Code

Install **pgfmt** (`sirmspencer.pgfmt`) from the Visual Studio Marketplace or Open VSX, then make it the SQL formatter:

```json
{
  "[sql]": {
    "editor.defaultFormatter": "sirmspencer.pgfmt",
    "editor.formatOnSave": true
  }
}
```

The extension bundles pgfmt. The `pgfmt.path` setting points it at another build, such as one from `./bin/link`.

### Neovim

Install pgfmt with Homebrew, npm, or a release binary, then configure conform.nvim:

```lua
require("conform").setup({
  formatters = {
    -- run from the file's folder, so the nearest .pgfmt.json applies
    pgfmt = {
      command = "pgfmt",
      args = { "-stdin-path", "$FILENAME" },
      stdin = true,
      cwd = function(_, ctx) return vim.fs.dirname(ctx.filename) end,
    },
  },
  formatters_by_ft = {
    sql = { "pgfmt" },
  },
  format_on_save = { timeout_ms = 2000 },
})
```

### IntelliJ

Install **pgfmt** from the JetBrains Marketplace (Settings, Plugins). It replaces the built-in SQL formatter for Reformat Code, reformat on save, and reformat on commit. It works in IntelliJ IDEA, DataGrip, and the other JetBrains IDEs, including Community editions. The plugin bundles pgfmt; Settings, Tools, pgfmt points it at another build.

### Emacs

Install pgfmt with Homebrew, npm, or a release binary, then configure apheleia, which runs it from the buffer's folder:

```elisp
(with-eval-after-load 'apheleia
  (setf (alist-get 'pgfmt apheleia-formatters) '("pgfmt" "-stdin-path" filepath))
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

The editor plugins use their bundled pgfmt. To test a local change in an editor, run `./bin/link` and set the plugin's pgfmt path (`pgfmt.path` in VS Code, Settings, Tools, pgfmt in IntelliJ) to the linked binary.

Each directory in `testdata/cases` contains an `input.sql` and the `expected.sql` it must format to. The tests also check that formatting `expected.sql` leaves it unchanged and that no comments remain.

`rules.md` is generated. Edit `docs/rules.template.md`, which contains each rule and its input examples, then run `go run ./tools/rules` to write `rules.md` with the formatter's output below every example. `go test ./...` fails if `rules.md` is out of date.

## Releasing

[docs/releasing.md](docs/releasing.md) describes the accounts, tokens, and one-time steps behind the release workflow. Pushing a version tag releases every package:

```sh
git tag v0.1.0
git push origin v0.1.0
```

The `release` workflow tests and builds pgfmt on macOS, Linux, and Windows, creates the GitHub release with the archives and `checksums.txt`, publishes the npm packages, and publishes the VS Code extension to the Visual Studio Marketplace and Open VSX. It also builds the IntelliJ plugin and attaches it to the release; the plugin is published to the JetBrains Marketplace once the repository variable `JETBRAINS_PUBLISH` is `true`, which can be set after the first version is uploaded and approved by hand. The secrets it uses are `TAP_TOKEN`, `NPM_TOKEN`, `VSCE_PAT`, `OVSX_PAT`, and `JETBRAINS_MARKETPLACE_TOKEN`.

It also updates the Homebrew formula in `sirmspencer/homebrew-tap` from the release checksums, after installing and testing it on macOS. `scripts/update-formula.sh <version> <tap>/pgfmt.rb` does the same by hand.

## License

MIT. See [LICENSE](LICENSE).
