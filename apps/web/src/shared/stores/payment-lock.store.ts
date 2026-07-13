import { create } from "zustand";

// Tracks whether the tenant's subscription package is currently inactive (§2.15) — set by
// http-client.ts's 402 interceptor branch, read by AppLayout to render the reduced/locked
// shell. Cleared only by an explicit "Cek Status Pembayaran" check on the Langganan page
// confirming an active package (manual, no polling — same philosophy as §2.4).
interface PaymentLockState {
  locked: boolean;
  setLocked: (locked: boolean) => void;
}

export const usePaymentLockStore = create<PaymentLockState>((set) => ({
  locked: false,
  setLocked: (locked) => set({ locked }),
}));
