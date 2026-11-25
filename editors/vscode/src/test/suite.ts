// runs inside vs code: formats real documents through the extension
import * as assert from "assert";
import * as fs from "fs";
import * as os from "os";
import * as path from "path";
import * as vscode from "vscode";

async function formatted(text: string, folder?: string): Promise<string> {
  let document: vscode.TextDocument;
  if (folder) {
    const file = path.join(folder, "query.sql");
    fs.writeFileSync(file, text);
    document = await vscode.workspace.openTextDocument(file);
  } else {
    document = await vscode.workspace.openTextDocument({ language: "sql", content: text });
  }
  await vscode.window.showTextDocument(document);
  await vscode.commands.executeCommand("editor.action.formatDocument");
  return document.getText();
}

export async function run(): Promise<void> {
  // PGFMT_TEST_PATH points at a development build; without it the bundled binary is used
  const executable = process.env.PGFMT_TEST_PATH ?? "";
  await vscode.workspace.getConfiguration("pgfmt").update("path", executable, vscode.ConfigurationTarget.Global);
  console.log(`pgfmt under test: ${executable || "bundled"}`);
  const extension = vscode.extensions.getExtension("sirmspencer.pgfmt");
  assert.ok(extension, "extension is installed");
  await extension.activate();

  assert.strictEqual(await formatted("SELECT id FROM users WHERE id = :id"), "select id\nfrom users\nwhere id = :id\n");

  // a syntax error leaves the document unchanged
  assert.strictEqual(await formatted("select from where"), "select from where");

  // the nearest .pgfmt.json applies
  const folder = fs.mkdtempSync(path.join(os.tmpdir(), "pgfmt-"));
  fs.writeFileSync(path.join(folder, ".pgfmt.json"), '{"indent": 4}');
  assert.strictEqual(await formatted("select a, b, c, d from t", folder), "select\n    a,\n    b,\n    c,\n    d\nfrom t\n");

  // a file in a migrations folder gets --;; between statements
  const migrations = path.join(folder, "migrations");
  fs.mkdirSync(migrations);
  assert.strictEqual(await formatted("drop table a; drop table b", migrations), "drop table a;\n--;;\ndrop table b;\n");

  console.log("all pgfmt extension tests passed");
}
