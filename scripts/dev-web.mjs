import { spawn } from "node:child_process";
import process from "node:process";

const server = spawn("go", ["run", "./cmd/web3-service-agent"], {
  cwd: process.cwd(),
  stdio: "inherit"
});

for (const signal of ["SIGINT", "SIGTERM"]) {
  process.on(signal, () => server.kill(signal));
}

server.on("exit", (code) => process.exit(code ?? 0));
