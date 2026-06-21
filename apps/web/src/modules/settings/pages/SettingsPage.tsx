import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Loader2, Save } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/shared/components/ui/card";
import { Input } from "@/shared/components/ui/input";
import { Button } from "@/shared/components/ui/button";
import { Checkbox } from "@/shared/components/ui/checkbox";
import { Label } from "@/shared/components/ui/label";
import { settingsService } from "@/modules/settings/services/settings.service";
import { settingsSchema } from "@/modules/settings/schemas/settings.schema";
import type { Settings } from "@/modules/settings/types/settings.types";

const errMsg = (e: unknown) => (e instanceof Error ? e.message : "Terjadi kesalahan.");

// Pengaturan toko: pajak & layanan (PPN), fitur, dan ambang kontrol.
export default function SettingsPage() {
  const [form, setForm] = useState<Settings | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    let active = true;
    settingsService
      .get()
      .then((s) => active && setForm(s))
      .catch((e) => active && toast.error(errMsg(e)))
      .finally(() => active && setLoading(false));
    return () => {
      active = false;
    };
  }, []);

  if (loading || !form) {
    return (
      <div className="flex min-h-[60vh] items-center justify-center">
        <Loader2 className="h-6 w-6 animate-spin text-muted" />
      </div>
    );
  }

  const set = <K extends keyof Settings>(key: K, value: Settings[K]) =>
    setForm((f) => (f ? { ...f, [key]: value } : f));

  const num = (key: keyof Settings) => (e: React.ChangeEvent<HTMLInputElement>) =>
    set(key, (Number.parseInt(e.target.value, 10) || 0) as Settings[typeof key]);

  const save = async () => {
    const parsed = settingsSchema.safeParse(form);
    if (!parsed.success) {
      toast.error(parsed.error.issues[0]?.message ?? "Nilai tidak valid.");
      return;
    }
    setSaving(true);
    try {
      const updated = await settingsService.update(parsed.data);
      setForm(updated);
      toast.success("Pengaturan berhasil disimpan");
    } catch (e) {
      toast.error(errMsg(e));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="mx-auto max-w-2xl space-y-5 p-4">
      <div>
        <h1 className="text-lg font-semibold">Pengaturan</h1>
        <p className="text-sm text-muted">Atur pajak, biaya layanan, fitur, dan ambang kontrol toko.</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Pajak & Layanan</CardTitle>
          <CardDescription>
            Biaya layanan (termasuk biaya payment gateway untuk QRIS) dan PPN tampil sebagai
            rincian terpisah ke pelanggan.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <label className="flex items-center gap-3">
            <Checkbox
              checked={form.taxEnabled}
              onChange={(e) => set("taxEnabled", e.target.checked)}
            />
            <span className="text-sm font-medium">Aktifkan PPN</span>
          </label>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor="taxPercent">PPN (%)</Label>
              <Input
                id="taxPercent"
                type="number"
                min={0}
                max={100}
                value={form.taxPercent}
                disabled={!form.taxEnabled}
                onChange={num("taxPercent")}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="servicePercent">Biaya layanan (%)</Label>
              <Input
                id="servicePercent"
                type="number"
                min={0}
                max={100}
                value={form.servicePercent}
                onChange={num("servicePercent")}
              />
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Fitur</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <label className="flex items-center gap-3">
            <Checkbox
              checked={form.featureSelfOrder}
              onChange={(e) => set("featureSelfOrder", e.target.checked)}
            />
            <span className="text-sm font-medium">Self-order pelanggan (QR meja)</span>
          </label>
          <label className="flex items-center gap-3">
            <Checkbox checked={form.featureQris} onChange={(e) => set("featureQris", e.target.checked)} />
            <span className="text-sm font-medium">Pembayaran QRIS</span>
          </label>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Ambang Kontrol</CardTitle>
          <CardDescription>Batas yang memicu persetujuan supervisor.</CardDescription>
        </CardHeader>
        <CardContent className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <div className="space-y-1.5">
            <Label htmlFor="maxDiscountPercent">Diskon maks (%)</Label>
            <Input
              id="maxDiscountPercent"
              type="number"
              min={0}
              max={100}
              value={form.maxDiscountPercent}
              onChange={num("maxDiscountPercent")}
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="maxOperationalExpense">Biaya operasional maks (Rp)</Label>
            <Input
              id="maxOperationalExpense"
              type="number"
              min={0}
              value={form.maxOperationalExpense}
              onChange={num("maxOperationalExpense")}
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="cashVarianceTolerance">Toleransi selisih kas (Rp)</Label>
            <Input
              id="cashVarianceTolerance"
              type="number"
              min={0}
              value={form.cashVarianceTolerance}
              onChange={num("cashVarianceTolerance")}
            />
          </div>
        </CardContent>
      </Card>

      <div className="flex justify-end">
        <Button onClick={save} disabled={saving} className="gap-2">
          {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
          Simpan
        </Button>
      </div>
    </div>
  );
}
