// registers pgfmt as the formatter for sql documents. the document is piped
// through pgfmt from its own folder, so the nearest .pgfmt.json applies.
import * as vscode from "vscode";
import { spawn } from "child_process";
import * as fs from "fs";
import * as path from "path";

const TIMEOUT_MS = 30_000;

// apps started from the dock may not see the shell path, so check homebrew directly
const HOMEBREW_PATHS = ["/opt/homebrew/bin/pgfmt", "/usr/local/bin/pgfmt"];

export function activate(context: vscode.ExtensionContext) {
  const output = vscode.window.createOutputChannel("pgfmt");
  context.subscriptions.push(
    output,
    vscode.languages.registerDocumentFormattingEditProvider("sql", {
      provideDocumentFormattingEdits: (document) => format(context, output, document),
    }),
  );
}

export function deactivate() {}

// the configured path, then the bundled binary, then pgfmt on the path or in homebrew
function findExecutable(context: vscode.ExtensionContext): string {
  const configured = vscode.workspace.getConfiguration("pgfmt").get<string>("path", "").trim();
  if (configured) {
    return configured;
  }
  const bundled = context.asAbsolutePath(path.join("bin", process.platform === "win32" ? "pgfmt.exe" : "pgfmt"));
  if (fs.existsSync(bundled)) {
    return bundled;
  }
  return HOMEBREW_PATHS.find((candidate) => fs.existsSync(candidate)) ?? "pgfmt";
}

function workingDirectory(document: vscode.TextDocument): string | undefined {
  if (document.uri.scheme === "file") {
    return path.dirname(document.uri.fsPath);
  }
  return vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
}

async function format(
  context: vscode.ExtensionContext,
  output: vscode.OutputChannel,
  document: vscode.TextDocument,
): Promise<vscode.TextEdit[]> {
  const executable = findExecutable(context);
  const text = document.getText();
  try {
    const formatted = await run(executable, text, workingDirectory(document), output);
    if (formatted === text) {
      return [];
    }
    const whole = new vscode.Range(document.positionAt(0), document.positionAt(text.length));
    return [vscode.TextEdit.replace(whole, formatted)];
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    output.appendLine(message);
    vscode.window.showErrorMessage(`pgfmt: ${message}`);
    return [];
  }
}

// runs pgfmt with text on stdin and resolves with stdout; any notes pgfmt
// prints on stderr while succeeding, such as fixed commas, go to the output channel
function run(executable: string, text: string, cwd: string | undefined, output: vscode.OutputChannel): Promise<string> {
  return new Promise((resolve, reject) => {
    const child = spawn(executable, [], { cwd });
    let stdout = "";
    let stderr = "";
    const timer = setTimeout(() => {
      child.kill();
      reject(new Error(`did not finish within ${TIMEOUT_MS / 1000} seconds`));
    }, TIMEOUT_MS);
    child.stdout.on("data", (chunk) => (stdout += chunk));
    child.stderr.on("data", (chunk) => (stderr += chunk));
    child.on("error", (error) => {
      clearTimeout(timer);
      reject(new Error(`could not run ${executable}: ${error.message}`));
    });
    child.on("close", (code) => {
      clearTimeout(timer);
      if (stderr.trim()) {
        output.appendLine(stderr.trim());
      }
      if (code === 0) {
        resolve(stdout);
      } else {
        reject(new Error(stderr.trim() || `exited with status ${code}`));
      }
    });
    child.stdin.end(text);
  });
}
