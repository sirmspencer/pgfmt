// downloads a test copy of vs code and runs suite.ts inside it with this extension loaded
import * as path from "path";
import { runTests } from "@vscode/test-electron";

async function main() {
  const extensionDevelopmentPath = path.resolve(__dirname, "..", "..");
  const extensionTestsPath = path.resolve(__dirname, "suite.js");
  await runTests({ extensionDevelopmentPath, extensionTestsPath, launchArgs: ["--disable-extensions"] });
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
