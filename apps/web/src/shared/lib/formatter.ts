// Formatting helpers (rupiah & local Asia/Jakarta time) — used across modules.
const idr = new Intl.NumberFormat("id-ID", {
  style: "currency",
  currency: "IDR",
  maximumFractionDigits: 0,
});
const dateTime = new Intl.DateTimeFormat("id-ID", {
  dateStyle: "medium",
  timeStyle: "short",
  timeZone: "Asia/Jakarta",
});
const dateOnly = new Intl.DateTimeFormat("id-ID", {
  dateStyle: "medium",
  timeZone: "Asia/Jakarta",
});

export const formatIDR = (n: number) => idr.format(n || 0);
export const formatDateTime = (iso?: string | null) => (iso ? dateTime.format(new Date(iso)) : "—");
export const formatDate = (iso?: string | null) => (iso ? dateOnly.format(new Date(iso)) : "—");
export const formatNumber = (n: number) => new Intl.NumberFormat("id-ID").format(n || 0);
