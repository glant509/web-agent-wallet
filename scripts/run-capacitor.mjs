import { spawn } from "node:child_process";
import path from "node:path";
import process from "node:process";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const argumentsToForward = process.argv.slice(2);
if (argumentsToForward.length === 0) {
  throw new Error("Capacitor command is required");
}

const executable = process.platform === "win32" ? "npx.cmd" : "npx";
const child = spawn(executable, ["cap", ...argumentsToForward], {
  cwd: path.join(root, "clients/mobile"),
  stdio: "inherit"
});

child.on("exit", (code) => process.exit(code ?? 1));
