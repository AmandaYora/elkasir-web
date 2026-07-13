// Store settings (camelCase, aligned with backend DTO). Percent fields are integers (11 = 11%).
export interface Settings {
  storeName: string;
  // Identitas tenant di URL self-order publik (/order/<slug>/<kodeMeja>) — read-only di sini;
  // dikelola modul `platform` (superadmin), bukan lewat halaman Pengaturan ini.
  storeSlug: string;
  storePhone: string;
  storeAddress: string;
  storeLogoUrl: string;
  maxDiscountPercent: number;
  maxOperationalExpense: number;
  cashVarianceTolerance: number;
  featureSelfOrder: boolean;
  featureQris: boolean;
  featurePayAtCashier: boolean;
  taxEnabled: boolean;
  taxPercent: number;
  servicePercent: number;
}

// storeSlug is read-only (managed by the platform/superadmin module, not this page).
export type SettingsInput = Omit<Settings, "storeSlug">;
