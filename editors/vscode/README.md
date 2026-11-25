# pgfmt for VS Code

Formats PostgreSQL `.sql` files with [pgfmt](https://github.com/sirmspencer/pgfmt), so that every editor on a team produces the same output. The pgfmt executable is bundled with the extension; nothing else needs to be installed.

## Usage

Run **Format Document** on a `.sql` file, or turn on format on save:

```json
"[sql]": {
  "editor.defaultFormatter": "sirmspencer.pgfmt",
  "editor.formatOnSave": true
}
```

A syntax error leaves the document unchanged and shows pgfmt's message. pgfmt runs from the file's folder, so the nearest `.pgfmt.json` in that folder or a parent folder sets its options. The [rules](https://github.com/sirmspencer/pgfmt/blob/master/rules.md) describe the default style and every setting.

## Settings

- `pgfmt.path`: path to a pgfmt executable to use instead of the bundled one, for example a development build.
