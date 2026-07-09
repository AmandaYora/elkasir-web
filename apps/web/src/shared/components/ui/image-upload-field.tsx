import { useRef, useState } from "react";
import { toast } from "sonner";
import { Upload, X, Image as ImageIcon } from "lucide-react";
import { Button } from "@/shared/components/ui/button";
import { Label } from "@/shared/components/ui/label";
import { mediaService } from "@/shared/services/media.service";

// Domain-agnostic image upload: kompres di browser, unggah ke /uploads (kategori
// menentukan folder object storage), lalu balikan URL-nya lewat onChange.
export function ImageUploadField({
  label,
  value,
  onChange,
  uploadingChange,
  category,
  fallback,
  helpText = "Format JPG, PNG, atau WebP. Gambar otomatis diperkecil agar tetap ringan.",
}: {
  label: string;
  value: string;
  onChange: (url: string) => void;
  uploadingChange: (busy: boolean) => void;
  category: string;
  fallback?: string;
  helpText?: string;
}) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [uploading, setUploading] = useState(false);
  const [progress, setProgress] = useState(0);

  const onFile = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    e.target.value = ""; // izinkan memilih file yang sama lagi
    if (!file) return;
    if (!file.type.startsWith("image/")) {
      toast.error("File harus berupa gambar.");
      return;
    }
    setUploading(true);
    uploadingChange(true);
    setProgress(0);
    try {
      const res = await mediaService.uploadImage(file, category, setProgress);
      onChange(res.url);
      toast.success("Gambar berhasil diunggah");
    } catch {
      toast.error("Gagal mengunggah gambar. Coba lagi.");
    } finally {
      setUploading(false);
      uploadingChange(false);
    }
  };

  const src = value || fallback;

  return (
    <div className="grid gap-2">
      <Label>{label}</Label>
      <div className="flex items-center gap-3">
        <div className="relative h-20 w-20 shrink-0 overflow-hidden rounded-lg border border-border bg-surface-muted">
          {src ? (
            <img
              src={src}
              alt=""
              onError={(e) => {
                if (fallback && e.currentTarget.src !== fallback) e.currentTarget.src = fallback;
              }}
              className="h-full w-full object-cover"
            />
          ) : (
            <div className="flex h-full w-full items-center justify-center text-muted">
              <ImageIcon className="h-7 w-7" />
            </div>
          )}
          {uploading && (
            <div className="absolute inset-0 flex items-center justify-center bg-black/50 text-xs font-medium text-white">
              {progress}%
            </div>
          )}
        </div>
        <div className="flex flex-col items-start gap-1.5">
          <input ref={inputRef} type="file" accept="image/*" className="hidden" onChange={onFile} />
          <Button
            type="button"
            variant="outline"
            size="sm"
            loading={uploading}
            onClick={() => inputRef.current?.click()}
          >
            <Upload className="h-3.5 w-3.5" /> {value ? "Ganti gambar" : "Unggah gambar"}
          </Button>
          {value && !uploading && (
            <Button type="button" variant="ghost" size="sm" onClick={() => onChange("")}>
              <X className="h-3.5 w-3.5" /> Hapus gambar
            </Button>
          )}
        </div>
      </div>
      <p className="text-xs text-muted">{helpText}</p>
    </div>
  );
}
