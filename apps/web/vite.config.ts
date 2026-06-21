import { defineConfig } from "vite";
import { fileURLToPath, URL } from "node:url";
import viteReact from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

// Pure static SPA: React 19 + react-router-dom + Vite (no SSR, no TanStack). `vite build`
// produces `dist/` (index.html + assets) which is embedded into the Go binary for the
// one-container deployment. VITE_* env vars are exposed via import.meta.env.
export default defineConfig({
  plugins: [tailwindcss(), viteReact()],
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
  server: {
    host: "::",
    port: 8080,
  },
});
