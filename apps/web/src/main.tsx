// SPA entry (client-only). `vite build` produces static assets (index.html + assets)
// embedded into the Go binary for the one-container deployment.
import { StrictMode } from "react";
import ReactDOM from "react-dom/client";
import { App } from "@/app/App";
import "@/styles/globals.css";

const rootEl = document.getElementById("root");
if (!rootEl) {
  throw new Error("Elemen #root tidak ditemukan di index.html");
}

ReactDOM.createRoot(rootEl).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
