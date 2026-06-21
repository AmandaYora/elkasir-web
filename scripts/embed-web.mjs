// Embed the built SPA (apps/web/dist) into the Go binary's embed dir
// (apps/api/internal/webui/dist) for the one-container build. Cross-platform.
//
// Run after `npm run build:web` and before `npm run build:api`. The root `build`
// script chains these: build:web -> embed-web -> build:api.
import { existsSync, rmSync, mkdirSync, cpSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");
const src = join(root, "apps", "web", "dist");
const dest = join(root, "apps", "api", "internal", "webui", "dist");

if (!existsSync(src)) {
  console.error(`[embed-web] missing ${src} — run "npm run build:web" first.`);
  process.exit(1);
}

// Reset the embed dir, then copy fresh build output in. A committed
// placeholder index.html keeps `go build` working when the SPA isn't built.
rmSync(dest, { recursive: true, force: true });
mkdirSync(dest, { recursive: true });
cpSync(src, dest, { recursive: true });

if (!existsSync(join(dest, "index.html"))) {
  writeFileSync(join(dest, "index.html"), "<!doctype html><title>Elkasir</title>");
}

console.log(`[embed-web] copied ${src} -> ${dest}`);
