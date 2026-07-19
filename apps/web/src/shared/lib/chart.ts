// Compact, locale-aware formatting for chart axes/labels. Indonesian magnitude suffixes:
// rb (ribu), jt (juta), M (miliar) — so axes read "Rp1,2 jt" instead of the ambiguous "1M".

function trim(n: number): string {
  return (Math.round(n * 10) / 10).toLocaleString("id-ID", { maximumFractionDigits: 1 });
}

function compact(n: number): string {
  const abs = Math.abs(n);
  if (abs >= 1_000_000_000) return `${trim(n / 1_000_000_000)} M`;
  if (abs >= 1_000_000) return `${trim(n / 1_000_000)} jt`;
  if (abs >= 1_000) return `${trim(n / 1_000)} rb`;
  return `${Math.round(n)}`;
}

export const formatCompactIDR = (n: number) => `Rp${compact(n || 0)}`;
export const formatCompactNumber = (n: number) => compact(n || 0);

// Single-hue rank shading: opacity 1.0 at the top of a ranked list, fading toward ~0.5.
export const rankShade = (i: number, n: number) => 1 - (0.5 * i) / Math.max(1, n - 1);

// "YYYY-MM-DD" → "12 Mar". Parsed/formatted in UTC so the calendar day never shifts.
const dayShort = new Intl.DateTimeFormat("id-ID", {
  day: "numeric",
  month: "short",
  timeZone: "UTC",
});
export const formatDayShort = (ymd: string) => {
  const d = new Date(`${ymd}T00:00:00Z`);
  return Number.isNaN(d.getTime()) ? ymd : dayShort.format(d);
};

// "YYYY-MM" → "Mar 2026". Parsed/formatted in UTC so the calendar month never shifts.
const monthShort = new Intl.DateTimeFormat("id-ID", {
  month: "short",
  year: "numeric",
  timeZone: "UTC",
});
export const formatMonthShort = (ym: string) => {
  const d = new Date(`${ym}-01T00:00:00Z`);
  return Number.isNaN(d.getTime()) ? ym : monthShort.format(d);
};
