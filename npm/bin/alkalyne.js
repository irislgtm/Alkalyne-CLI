#!/usr/bin/env node
const { spawn } = require("child_process");
const { existsSync, readdirSync } = require("fs");
const path = require("path");

const binName =
  process.platform === "win32" ? "alkalyne.exe" : "alkalyne";
const packageRoot = path.join(__dirname, "..");
const binPath = path.join(packageRoot, binName);

function findFallbackBinary() {
  const ext = process.platform === "win32" ? ".exe" : "";
  const entries = readdirSync(packageRoot, { withFileTypes: true });
  const match = entries.find(
    (entry) =>
      entry.isFile() &&
      entry.name.startsWith("alkalyne-") &&
      entry.name.endsWith(ext),
  );
  return match ? path.join(packageRoot, match.name) : null;
}

const resolvedBinPath = existsSync(binPath) ? binPath : findFallbackBinary();

if (!resolvedBinPath) {
  console.error("alkalyne: executable not found in package install directory");
  console.error("alkalyne: try reinstalling with 'npm i -g alkalyne'");
  process.exit(1);
}

const child = spawn(resolvedBinPath, process.argv.slice(2), {
  stdio: "inherit",
});

child.on("error", (err) => {
  if (err.code === "ENOENT") {
    console.error(`alkalyne: failed to launch binary at ${resolvedBinPath}`);
    console.error("alkalyne: try reinstalling with 'npm i -g alkalyne'");
    process.exit(1);
  }

  console.error(`alkalyne: failed to launch: ${err.message}`);
  process.exit(1);
});

child.on("exit", (code) => {
  process.exit(code);
});
