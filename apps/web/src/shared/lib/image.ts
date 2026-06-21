import imageCompression from "browser-image-compression";

// URL gambar default (di object storage) untuk produk tanpa gambar. Bucket dipakai
// bersama dev & prod, jadi URL ini stabil di semua environment.
export const DEFAULT_PRODUCT_IMAGE_URL =
  "https://is3.cloudhost.id/elcodelabs/elkasir/upload/defaults/no-image.jpg";

// Kompresi tahap-1 (di browser) sebelum upload — menghemat bandwidth. Backend masih
// melakukan kompresi tahap-2 (resize 1280px + JPEG q82) sebelum menyimpan ke storage,
// jadi nilai di sini cukup longgar dan aman bila gagal (fallback ke file asli).
export async function compressImage(file: File): Promise<File> {
  if (!file.type.startsWith("image/")) return file;
  try {
    return await imageCompression(file, {
      maxSizeMB: 1,
      maxWidthOrHeight: 1280,
      useWebWorker: true,
      initialQuality: 0.8,
    });
  } catch {
    return file; // backend tetap mengompres tahap-2
  }
}
