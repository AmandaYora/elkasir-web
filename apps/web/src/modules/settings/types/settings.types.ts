// Store settings (camelCase, aligned with backend DTO). Percent fields are integers (11 = 11%).
export interface Settings {
  maxDiscountPercent: number;
  maxOperationalExpense: number;
  cashVarianceTolerance: number;
  featureSelfOrder: boolean;
  featureQris: boolean;
  taxEnabled: boolean;
  taxPercent: number;
  servicePercent: number;
}

export type SettingsInput = Settings;
