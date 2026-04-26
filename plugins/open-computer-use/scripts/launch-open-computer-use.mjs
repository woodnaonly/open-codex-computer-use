#!/usr/bin/env node

import { accessSync, constants, existsSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { spawn } from "node:child_process";

const scriptPath = fileURLToPath(import.meta.url);
const pluginRoot = path.resolve(path.dirname(scriptPath), "..");
const pluginInstallRoot = path.dirname(pluginRoot);
const repoRoot = path.resolve(pluginRoot, "..", "..");
const arch = process.arch === "x64" ? "amd64" : process.arch;

function isRunnable(filePath) {
  if (!existsSync(filePath)) {
    return false;
  }
  if (process.platform === "win32") {
    return true;
  }
  try {
    accessSync(filePath, constants.X_OK);
    return true;
  } catch {
    return false;
  }
}

const candidateBinaries = [
  path.join(pluginRoot, "Open Computer Use.app", "Contents", "MacOS", "OpenComputerUse"),
  path.join(pluginRoot, "Open Computer Use (Dev).app", "Contents", "MacOS", "OpenComputerUse"),
  path.join(pluginRoot, "OpenComputerUse.app", "Contents", "MacOS", "OpenComputerUse"),
  path.join(pluginRoot, "open-computer-use"),
  path.join(pluginRoot, "open-computer-use.exe"),
  path.join(pluginInstallRoot, "open-computer-use"),
  path.join(pluginInstallRoot, "open-computer-use.exe"),
  path.join(repoRoot, "dist", "Open Computer Use (Dev).app", "Contents", "MacOS", "OpenComputerUse"),
  path.join(repoRoot, "dist", "Open Computer Use.app", "Contents", "MacOS", "OpenComputerUse"),
  path.join(repoRoot, "dist", "OpenComputerUse.app", "Contents", "MacOS", "OpenComputerUse"),
  path.join(repoRoot, "dist", "linux", arch, "open-computer-use"),
  path.join(repoRoot, "dist", "windows", arch, "open-computer-use.exe"),
];

const selected = candidateBinaries.find(isRunnable);

if (!selected) {
  console.error("open-computer-use could not find a runnable native runtime.");
  console.error("Checked:");
  for (const candidate of candidateBinaries) {
    console.error(`  - ${candidate}`);
  }
  process.exit(1);
}

const child = spawn(selected, ["mcp"], {
  cwd: selected.startsWith(pluginRoot) ? pluginRoot : repoRoot,
  stdio: "inherit",
  windowsHide: true,
});

child.on("error", (error) => {
  console.error(`Failed to start ${selected}: ${error.message}`);
  process.exit(1);
});

child.on("exit", (code, signal) => {
  if (signal) {
    process.exit(1);
  }
  process.exit(code ?? 0);
});
