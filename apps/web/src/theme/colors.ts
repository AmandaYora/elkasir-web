// Theme palette as TS tokens (mirror of theme.css) for JS consumers like charts.
// Keep in sync with theme.css — that file is the source of truth for CSS utilities.
export const colors = {
  primary: "#2563eb",
  primaryHover: "#1d4ed8",
  primarySoft: "#dbeafe",
  secondary: "#64748b",
  background: "#f8fafc",
  surface: "#ffffff",
  surfaceMuted: "#f1f5f9",
  border: "#e2e8f0",
  text: "#0f172a",
  muted: "#64748b",
  danger: "#dc2626",
  success: "#16a34a",
  warning: "#d97706",
} as const;

// Ordered palette for categorical charts (recharts).
export const chartPalette = [
  colors.primary,
  colors.success,
  colors.warning,
  "#8b5cf6",
  "#06b6d4",
  "#ec4899",
] as const;

export type ColorToken = keyof typeof colors;
