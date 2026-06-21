import { api } from "@/shared/services/http-client";
import { endpoints } from "@/shared/services/api-endpoints";
import { compressImage } from "@/shared/lib/image";

export interface UploadResult {
  key: string;
  url: string;
}

// Domain-agnostic media upload. Mengompres tahap-1 di browser, lalu mengunggah ke
// /uploads (backend kompres tahap-2 → object storage). `category` menentukan folder
// (elkasir/upload/<category>/<file>).
export const mediaService = {
  async uploadImage(
    file: File,
    category = "product",
    onProgress?: (percent: number) => void,
  ): Promise<UploadResult> {
    const compressed = await compressImage(file);
    const form = new FormData();
    form.append("file", compressed, compressed.name || file.name);
    form.append("category", category);
    return api.upload<UploadResult>(endpoints.uploads, form, {
      onUploadProgress: (e) => {
        if (onProgress && e.total) onProgress(Math.round((e.loaded / e.total) * 100));
      },
    });
  },
};
