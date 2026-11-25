#!/usr/bin/env node
// runs the pgfmt binary from the platform package npm installed for this machine
"use strict";
const { spawnSync } = require("child_process");
const path = require("path");

const platforms = {
  "darwin arm64": "@sirmspencer/pgfmt-darwin-arm64",
  "darwin x64": "@sirmspencer/pgfmt-darwin-x64",
  "linux arm64": "@sirmspencer/pgfmt-linux-arm64",
  "linux x64": "@sirmspencer/pgfmt-linux-x64",
  "win32 x64": "@sirmspencer/pgfmt-win32-x64",
};

const name = platforms[`${process.platform} ${process.arch}`];
if (!name) {
  console.error(`pgfmt: no binary for ${process.platform} ${process.arch}`);
  process.exit(2);
}

let binary;
try {
  const root = path.dirname(require.resolve(`${name}/package.json`));
  binary = path.join(root, process.platform === "win32" ? "pgfmt.exe" : "pgfmt");
} catch {
  console.error(`pgfmt: ${name} is not installed; reinstall pgfmt without --no-optional`);
  process.exit(2);
}

const result = spawnSync(binary, process.argv.slice(2), { stdio: "inherit" });
if (result.error) {
  console.error(`pgfmt: ${result.error.message}`);
  process.exit(2);
}
process.exit(result.status ?? 2);
