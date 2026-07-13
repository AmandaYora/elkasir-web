// Cross-platform runner for the Go `air` live-reload dev server. `air` is installed via
// `go install github.com/air-verse/air@latest`, which places the binary under
// `$(go env GOPATH)/bin` — a directory that isn't guaranteed to be on PATH (common gap on
// Windows). This wrapper resolves it directly instead of depending on the shell's PATH.
import { dirname, join, delimiter } from "node:path";
import { fileURLToPath } from "node:url";
import { existsSync } from "node:fs";
import { spawnSync, spawn } from "node:child_process";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");
const apiDir = join(root, "apps", "api");

function goEnv(key) {
  const res = spawnSync("go", ["env", key], { encoding: "utf8", shell: process.platform === "win32" });
  return res.stdout ? res.stdout.trim() : "";
}

const gobin = goEnv("GOBIN");
const gopath = goEnv("GOPATH");
const extraDirs = [gobin, gopath && join(gopath, "bin")].filter(Boolean);

const pathParts = (process.env.PATH || process.env.Path || "").split(delimiter).filter(Boolean);
for (const dir of extraDirs) {
  if (!pathParts.includes(dir)) pathParts.unshift(dir);
}

const airBin = process.platform === "win32" ? "air.exe" : "air";
const resolved = pathParts.find((dir) => existsSync(join(dir, airBin)));

if (!resolved) {
  console.error(
    "`air` not found on PATH or under $(go env GOPATH)/bin.\n" +
      "Install it with:\n  go install github.com/air-verse/air@latest\n"
  );
  process.exit(1);
}

const env = { ...process.env, PATH: pathParts.join(delimiter) };
const child = spawn(airBin, [], { cwd: apiDir, stdio: "inherit", env, shell: process.platform === "win32" });

for (const sig of ["SIGINT", "SIGTERM"]) {
  process.on(sig, () => child.kill(sig));
}
child.on("exit", (code, signal) => {
  if (signal) process.kill(process.pid, signal);
  else process.exit(code ?? 1);
});
