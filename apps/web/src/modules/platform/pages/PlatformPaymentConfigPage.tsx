import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Loader2, Save, ShieldCheck } from "lucide-react";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@/shared/components/ui/card";
import { Input } from "@/shared/components/ui/input";
import { Select } from "@/shared/components/ui/select";
import { Checkbox } from "@/shared/components/ui/checkbox";
import { Label } from "@/shared/components/ui/label";
import { Button } from "@/shared/components/ui/button";
import { ApiError } from "@/shared/types/api";
import { platformService } from "@/modules/platform/services/platform.service";
import type { GatewayConfig } from "@/modules/platform/types/platform.types";

// Konfigurasi Pembayaran (PLAN.md §9.1.2/§9.3 PF0) — SATU dompet gateway (§9.1.1), disimpan
// terenkripsi di database, diedit di sini alih-alih di .env server. Field secret (API Key,
// Private Key, Server Key) SENGAJA write-only: nilai asli tidak pernah dikirim balik dari
// server (hanya placeholder termask, mis. "••••9BXm") — mengosongkan field itu dan menyimpan
// TIDAK menghapus nilai yang tersimpan; hanya mengetik nilai baru yang benar-benar mengubahnya.
export default function PlatformPaymentConfigPage() {
  const [cfg, setCfg] = useState<GatewayConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  const [provider, setProvider] = useState<"tripay" | "midtrans" | "">("");
  const [sandbox, setSandbox] = useState(true);
  const [tripayMethod, setTripayMethod] = useState("QRIS");
  const [tripayMerchantCode, setTripayMerchantCode] = useState("");
  // Secret fields: undefined = belum disentuh (kirim apa adanya = "jangan ubah" di backend).
  const [tripayApiKey, setTripayApiKey] = useState<string | undefined>(undefined);
  const [tripayPrivateKey, setTripayPrivateKey] = useState<string | undefined>(undefined);
  const [midtransServerKey, setMidtransServerKey] = useState<string | undefined>(undefined);

  // ElProof (PLAN.md §11) — dompet TERPISAH, hanya untuk billing subscription tenant; selalu
  // aktif berdampingan dengan Provider di atas, bukan bagian dari pilihan Tripay/Midtrans.
  const [elproofAppId, setElproofAppId] = useState<string | undefined>(undefined);
  const [elproofSecret, setElproofSecret] = useState<string | undefined>(undefined);
  const [elproofBaseUrl, setElproofBaseUrl] = useState("");

  const load = () => {
    setLoading(true);
    platformService
      .getPaymentConfig()
      .then((c) => {
        setCfg(c);
        setProvider(c.provider);
        setSandbox(c.sandbox);
        setTripayMethod(c.tripayMethod || "QRIS");
        setTripayMerchantCode(c.tripayMerchantCode);
        setTripayApiKey(undefined);
        setTripayPrivateKey(undefined);
        setMidtransServerKey(undefined);
        setElproofAppId(undefined);
        setElproofSecret(undefined);
        setElproofBaseUrl(c.elproofBaseUrl);
      })
      .catch(() => toast.error("Gagal memuat konfigurasi pembayaran. Coba lagi."))
      .finally(() => setLoading(false));
  };

  useEffect(load, []);

  const save = async () => {
    setSaving(true);
    try {
      const updated = await platformService.updatePaymentConfig({
        provider,
        sandbox,
        tripayMethod,
        tripayMerchantCode,
        tripayApiKey,
        tripayPrivateKey,
        midtransServerKey,
        elproofAppId,
        elproofSecret,
        elproofBaseUrl,
      });
      setCfg(updated);
      setTripayApiKey(undefined);
      setTripayPrivateKey(undefined);
      setMidtransServerKey(undefined);
      setElproofAppId(undefined);
      setElproofSecret(undefined);
      toast.success("Konfigurasi pembayaran berhasil disimpan");
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : "Gagal menyimpan konfigurasi. Coba lagi.");
    } finally {
      setSaving(false);
    }
  };

  if (loading || !cfg) {
    return (
      <div className="flex min-h-[60vh] items-center justify-center">
        <Loader2 className="h-6 w-6 animate-spin text-muted" />
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-2xl space-y-5 p-4 md:p-6">
      <div>
        <h1 className="text-lg font-semibold text-text">Konfigurasi Pembayaran</h1>
        <p className="text-sm text-muted">
          Dompet Tripay/Midtrans untuk self-order pelanggan, dan (terpisah) kredensial ElProof untuk
          billing subscription tenant — ganti kredensial di sini tanpa perlu edit server.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Gateway Aktif</CardTitle>
          <CardDescription>Pilih satu provider yang aktif memproses pembayaran.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="provider">Provider</Label>
            <Select
              id="provider"
              value={provider}
              onChange={(e) => setProvider(e.target.value as typeof provider)}
            >
              <option value="">Simulasi (tanpa gateway aktif)</option>
              <option value="tripay">Tripay</option>
              <option value="midtrans">Midtrans</option>
            </Select>
          </div>
          <label className="flex items-center gap-3">
            <Checkbox checked={sandbox} onChange={(e) => setSandbox(e.target.checked)} />
            <span className="text-sm font-medium">Mode sandbox (bukan production)</span>
          </label>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Kredensial Tripay</CardTitle>
          <CardDescription>
            Merchant Code tampil apa adanya (bukan rahasia). API Key & Private Key tersimpan
            terenkripsi — biarkan kosong untuk mempertahankan nilai yang sudah ada.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="tripayMerchantCode">Merchant Code</Label>
            <Input
              id="tripayMerchantCode"
              value={tripayMerchantCode}
              onChange={(e) => setTripayMerchantCode(e.target.value)}
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="tripayMethod">Kode channel QRIS default</Label>
            <Input
              id="tripayMethod"
              value={tripayMethod}
              onChange={(e) => setTripayMethod(e.target.value)}
              placeholder="QRIS"
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="tripayApiKey">API Key</Label>
            <Input
              id="tripayApiKey"
              type="password"
              value={tripayApiKey ?? ""}
              onChange={(e) => setTripayApiKey(e.target.value)}
              placeholder={cfg.tripayApiKeyMasked || "Belum diatur"}
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="tripayPrivateKey">Private Key</Label>
            <Input
              id="tripayPrivateKey"
              type="password"
              value={tripayPrivateKey ?? ""}
              onChange={(e) => setTripayPrivateKey(e.target.value)}
              placeholder={cfg.tripayPrivateKeyMasked || "Belum diatur"}
            />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Kredensial Midtrans</CardTitle>
          <CardDescription>Cadangan — hanya perlu diisi bila provider = Midtrans.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="midtransServerKey">Server Key</Label>
            <Input
              id="midtransServerKey"
              type="password"
              value={midtransServerKey ?? ""}
              onChange={(e) => setMidtransServerKey(e.target.value)}
              placeholder={cfg.midtransServerKeyMasked || "Belum diatur"}
            />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>ElProof — Billing Subscription</CardTitle>
          <CardDescription>
            Dompet TERPISAH, hanya dipakai untuk tagihan langganan tenant — selalu aktif
            berdampingan dengan Gateway Aktif di atas (bukan bagian dari pilihan Tripay/Midtrans).
            Elkasir terdaftar di ElProof sebagai app eksternal "Elkasir-Billing"; appId & secret di
            bawah didapat dari tim ElProof lewat Platform Console mereka.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="elproofAppId">App ID</Label>
            <Input
              id="elproofAppId"
              value={elproofAppId ?? cfg.elproofAppId}
              onChange={(e) => setElproofAppId(e.target.value)}
              placeholder="Elkasir-Billing"
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="elproofSecret">Secret</Label>
            <Input
              id="elproofSecret"
              type="password"
              value={elproofSecret ?? ""}
              onChange={(e) => setElproofSecret(e.target.value)}
              placeholder={cfg.elproofSecretMasked || "Belum diatur"}
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="elproofBaseUrl">Base URL</Label>
            <Input
              id="elproofBaseUrl"
              value={elproofBaseUrl}
              onChange={(e) => setElproofBaseUrl(e.target.value)}
              placeholder="https://elproof.elcodelabs.com/api/v1"
            />
          </div>
        </CardContent>
      </Card>

      <div className="flex items-center justify-between rounded-xl border border-border bg-surface-muted px-4 py-3 text-xs text-muted">
        <div className="flex items-center gap-2">
          <ShieldCheck className="h-4 w-4 shrink-0" />
          <span>
            Perubahan berlaku langsung untuk tagihan berikutnya — tidak perlu restart server.
          </span>
        </div>
      </div>

      <div className="flex justify-end">
        <Button onClick={save} disabled={saving} className="gap-2">
          {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
          Simpan
        </Button>
      </div>
    </div>
  );
}
