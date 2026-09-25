import { constants } from "node:fs";
import { copyFile, cp, mkdir, mkdtemp, rm } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import process from "node:process";
import { spawn } from "node:child_process";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const args = process.argv.slice(2);
const archArgument = args.find((arg) => arg.startsWith("--arch="));
const arch = archArgument ? archArgument.slice("--arch=".length) : "amd64";
if (args.some((arg) => arg !== archArgument) || !["amd64", "arm64"].includes(arch)) {
  throw new Error("usage: npm run package:web-server -- [--arch=amd64|arm64]");
}

function run(command, commandArgs, options = {}) {
  return new Promise((resolve, reject) => {
    const child = spawn(command, commandArgs, {
      cwd: root,
      stdio: "inherit",
      ...options
    });
    child.on("error", reject);
    child.on("close", (code) => {
      if (code === 0) resolve();
      else reject(new Error(`${command} exited with status ${code}`));
    });
  });
}

async function publishArchive(source, outputDirectory) {
  for (let index = 0; ; index += 1) {
    const suffix = index === 0 ? "" : `-${index}`;
    const destination = path.join(outputDirectory, `web3agent${suffix}.tar.gz`);
    try {
      await copyFile(source, destination, constants.COPYFILE_EXCL);
      return destination;
    } catch (error) {
      if (error && error.code === "EEXIST") {
        continue;
      }
      throw error;
    }
  }
}

const stagingRoot = await mkdtemp(path.join(os.tmpdir(), "web3agent-package-"));
try {
  const packageName = "web3agent";
  const packageRoot = path.join(stagingRoot, packageName);
  const outputDirectory = path.join(root, "dist", "releases");
  const temporaryArchive = path.join(stagingRoot, "archive.tar.gz");

  await mkdir(path.join(packageRoot, "etc"), { recursive: true });
  await run("npm", ["run", "typecheck"]);
  await run("npm", ["run", "build:shared"]);
  await run("go", ["build", "-trimpath", "-o", path.join(packageRoot, "web3-service-agent"), "./cmd/web3-service-agent"], {
    env: { ...process.env, GOOS: "linux", GOARCH: arch, CGO_ENABLED: "0" }
  });
  await Promise.all([
    cp(path.join(root, "deploy", "application.example.properties"), path.join(packageRoot, "etc", "application.example.properties")),
    cp(path.join(root, "deploy", "WEB_SERVER_DEPLOY.md"), path.join(packageRoot, "DEPLOY.md")),
    cp(path.join(root, "deploy", "start.sh"), path.join(packageRoot, "start.sh")),
    cp(path.join(root, "deploy", "stop.sh"), path.join(packageRoot, "stop.sh")),
    cp(path.join(root, "deploy", "restart.sh"), path.join(packageRoot, "restart.sh"))
  ]);
  await run("tar", ["-czf", temporaryArchive, "-C", stagingRoot, packageName]);
  await mkdir(outputDirectory, { recursive: true });
  const archivePath = await publishArchive(temporaryArchive, outputDirectory);
  process.stdout.write(`Package ready: ${archivePath}\n`);
} finally {
  await rm(stagingRoot, { recursive: true, force: true });
}
