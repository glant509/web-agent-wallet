import { cp, mkdir, readFile, rm, writeFile } from "node:fs/promises";
import { build } from "esbuild";
import path from "node:path";
import process from "node:process";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const sourceDirectory = path.join(root, "internal/httpapi/ui");
const outputRoot = path.join(root, "dist");
const requestedTarget = process.argv[2] || "all";
const targets = requestedTarget === "all" ? ["web", "mobile", "extension"] : [requestedTarget];

if (!targets.every((target) => ["web", "mobile", "extension"].includes(target))) {
  throw new Error(`unknown client target: ${requestedTarget}`);
}

await build({
  entryPoints: [path.join(root, "clients/shared/src/global.ts")],
  bundle: true,
  format: "iife",
  target: "es2022",
  outfile: path.join(sourceDirectory, "shared.js")
});

async function copySharedAssets(destination) {
  const bootstrap = await readFile(path.join(sourceDirectory, "bootstrap.js"), "utf8");
  await Promise.all([
    ...[
      "index.html",
      "app.css",
      "app.js",
      "shared.js",
      "wallet_derivation.js",
      "evm_signer.js",
      "bip39_english.txt",
      "platform_runtime.js"
    ].map((name) => cp(path.join(sourceDirectory, name), path.join(destination, name))),
    writeFile(
      path.join(destination, "bootstrap.js"),
      bootstrap.replace(/const walletScriptBase = .*?;\n/, 'const walletScriptBase = "./";\n')
    )
  ]);
}

async function buildTarget(target) {
  const destination = path.join(outputRoot, target);
  await rm(destination, { recursive: true, force: true });
  await mkdir(destination, { recursive: true });
  await copySharedAssets(destination);
  if (target === "extension") {
    await cp(path.join(root, "clients/extension/static"), destination, { recursive: true });
  }
  process.stdout.write(`built ${target} client at ${path.relative(root, destination)}\n`);
}

await mkdir(outputRoot, { recursive: true });
for (const target of targets) {
  await buildTarget(target);
}
