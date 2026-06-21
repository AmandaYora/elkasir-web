// Cross-platform golang-migrate runner. Reads DB_* from apps/api/.env and builds
// the MySQL URL, then runs golang-migrate via `go run` (no global install needed).
//
//   node scripts/migrate.mjs up
//   node scripts/migrate.mjs down 1
//   node scripts/migrate.mjs create <name>
import { readFileSync, existsSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { spawnSync } from "node:child_process";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");
const apiDir = join(root, "apps", "api");
const migrationsDir = "db/migrations"; // relative to apiDir (cwd of the child process)
const MIGRATE_PKG = "github.com/golang-migrate/migrate/v4/cmd/migrate@v4.18.1";

function loadEnv(file) {
  const env = {};
  if (!existsSync(file)) return env;
  for (const line of readFileSync(file, "utf8").split(/\r?\n/)) {
    const m = line.match(/^\s*([A-Z0-9_]+)\s*=\s*(.*)\s*$/);
    if (m) env[m[1]] = m[2].replace(/^["']|["']$/g, "");
  }
  return env;
}

const e = { ...loadEnv(join(apiDir, ".env")), ...process.env };
const dbURL =
  e.DB_DSN_URL ||
  `mysql://${e.DB_USERNAME || "root"}:${e.DB_PASSWORD || ""}@tcp(${e.DB_HOST || "localhost"}:${e.DB_PORT || "3306"})/${e.DB_NAME || "elkasir_db"}`;

const [, , cmd, ...rest] = process.argv;
if (!cmd) {
  console.error("usage: node scripts/migrate.mjs <up|down 1|create <name>>");
  process.exit(1);
}

let args;
if (cmd === "create") {
  const name = rest[0];
  if (!name) {
    console.error("usage: node scripts/migrate.mjs create <name>");
    process.exit(1);
  }
  args = ["run", "-tags", "mysql", MIGRATE_PKG, "create", "-ext", "sql", "-dir", migrationsDir, "-seq", name];
} else {
  args = ["run", "-tags", "mysql", MIGRATE_PKG, "-path", migrationsDir, "-database", dbURL, cmd, ...rest];
}

const res = spawnSync("go", args, { cwd: apiDir, stdio: "inherit", shell: process.platform === "win32" });
process.exit(res.status ?? 1);
