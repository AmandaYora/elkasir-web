import { api } from "@/shared/services/http-client";
import { endpoints } from "@/shared/services/api-endpoints";
import type { Settings, SettingsInput } from "@/modules/settings/types/settings.types";

export const settingsService = {
  get: () => api.get<Settings>(endpoints.settings),
  update: (body: SettingsInput) => api.patch<Settings>(endpoints.settings, body),
};
