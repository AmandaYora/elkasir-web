import { useMemo, useState } from "react";
import { Building2 } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/shared/components/ui/button";
import { Label } from "@/shared/components/ui/label";
import { Textarea } from "@/shared/components/ui/textarea";
import { Badge } from "@/shared/components/ui/badge";
import { Card } from "@/shared/components/ui/card";
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from "@/shared/components/ui/table";
import { Modal } from "@/shared/components/ui/modal";
import { FieldError } from "@/shared/components/ui/field-error";
import { LoadingState, ErrorState, EmptyState } from "@/shared/components/feedback";
import { useAsync } from "@/shared/hooks/useAsync";
import { ApiError } from "@/shared/types/api";
import { platformService } from "@/modules/platform/services/platform.service";
import { usePlatformAuthStore } from "@/modules/platform/stores/platform-auth.store";
import { PlatformWithdrawalStatusBadge } from "@/modules/platform/components/PlatformWithdrawalStatusBadge";
import type { WithdrawalView } from "@/modules/platform/types/platform.types";

const formatRupiah = (n: number) => `Rp ${n.toLocaleString("id-ID")}`;

// Penarikan — the claim -> complete flow (§2.7). Two-step by design: any superadmin can Klaim a
// pending request, but only the claimant can Tandai Sukses — this is the part of the app most
// likely to be subtly wrong if "simplified" back to one button; see PLAN.md §2.7/§7 item 5.
export default function PlatformWithdrawalsPage() {
  const me = usePlatformAuthStore((s) => s.user);
  const activeQuery = useAsync(() => platformService.listActiveWithdrawals(), []);
  const balanceQuery = useAsync(() => platformService.tenantsRevenue(), []);
  const [rejecting, setRejecting] = useState<WithdrawalView | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);

  const balanceByStore = useMemo(() => {
    const m = new Map<string, number>();
    for (const t of balanceQuery.data ?? []) m.set(t.storeId, t.balance);
    return m;
  }, [balanceQuery.data]);

  const rows = activeQuery.data ?? [];
  const refresh = () => {
    activeQuery.refetch();
    balanceQuery.refetch();
  };

  const errorMessage = (e: unknown, fallback: string) =>
    e instanceof ApiError ? e.message : fallback;

  const claim = async (w: WithdrawalView) => {
    setBusyId(w.id);
    try {
      await platformService.claimWithdrawal(w.id);
      toast.success("Permintaan berhasil diklaim");
      refresh();
    } catch (e) {
      toast.error(errorMessage(e, "Gagal mengklaim permintaan. Coba lagi."));
      refresh();
    } finally {
      setBusyId(null);
    }
  };

  const complete = async (w: WithdrawalView) => {
    setBusyId(w.id);
    try {
      await platformService.completeWithdrawal(w.id);
      toast.success("Penarikan ditandai sukses");
      refresh();
    } catch (e) {
      toast.error(errorMessage(e, "Gagal menandai sukses. Coba lagi."));
      refresh();
    } finally {
      setBusyId(null);
    }
  };

  return (
    <div className="space-y-4 p-4 md:p-6">
      <div>
        <h2 className="text-lg font-semibold text-text">Penarikan</h2>
        <p className="text-sm text-muted">Permintaan pencairan yang menunggu diproses.</p>
      </div>

      <Card className="overflow-hidden">
        {activeQuery.loading ? (
          <LoadingState />
        ) : activeQuery.error ? (
          <ErrorState message="Gagal memuat permintaan. Coba lagi." onRetry={refresh} />
        ) : rows.length === 0 ? (
          <EmptyState
            title="Tidak ada permintaan aktif"
            description="Semua permintaan pencairan sudah diproses."
          />
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Tenant</TableHead>
                <TableHead>Jumlah</TableHead>
                <TableHead>Bank</TableHead>
                <TableHead>Saldo Tersedia</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="w-[220px]" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((w) => {
                const isClaimant = !!me && w.processedBy === me.id;
                return (
                  <TableRow key={w.id}>
                    <TableCell>
                      <div className="flex items-center gap-2.5">
                        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10 text-primary">
                          <Building2 className="h-4 w-4" />
                        </div>
                        <span className="text-sm font-medium">{w.tenantName}</span>
                      </div>
                    </TableCell>
                    <TableCell className="text-sm font-semibold tabular-nums">
                      {formatRupiah(w.amount)}
                    </TableCell>
                    <TableCell className="text-sm text-muted">
                      {w.bank} · {w.account}
                      <div className="text-xs">{w.holder}</div>
                    </TableCell>
                    <TableCell className="text-sm tabular-nums text-muted">
                      {balanceByStore.has(w.storeId)
                        ? formatRupiah(balanceByStore.get(w.storeId)!)
                        : "—"}
                    </TableCell>
                    <TableCell>
                      <div className="flex flex-col items-start gap-1">
                        <PlatformWithdrawalStatusBadge status={w.status} />
                        {w.status === "processing" && (
                          <Badge tone="neutral">Diklaim oleh {w.claimantName || "—"}</Badge>
                        )}
                      </div>
                    </TableCell>
                    <TableCell>
                      <div className="flex flex-wrap justify-end gap-2">
                        {w.status === "pending" && (
                          <Button size="sm" loading={busyId === w.id} onClick={() => claim(w)}>
                            Klaim
                          </Button>
                        )}
                        {w.status === "processing" && (
                          <Button
                            size="sm"
                            loading={busyId === w.id}
                            disabled={!isClaimant}
                            title={
                              isClaimant
                                ? undefined
                                : "Hanya superadmin yang mengklaim yang dapat menandai sukses."
                            }
                            onClick={() => complete(w)}
                          >
                            Tandai Sukses
                          </Button>
                        )}
                        <Button size="sm" variant="outline" onClick={() => setRejecting(w)}>
                          Tolak
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        )}
      </Card>

      <RejectModal
        withdrawal={rejecting}
        onClose={() => setRejecting(null)}
        onDone={() => {
          setRejecting(null);
          refresh();
        }}
      />
    </div>
  );
}

function RejectModal({
  withdrawal,
  onClose,
  onDone,
}: {
  withdrawal: WithdrawalView | null;
  onClose: () => void;
  onDone: () => void;
}) {
  const [reason, setReason] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const submit = async () => {
    if (!withdrawal) return;
    const trimmed = reason.trim();
    if (!trimmed) {
      setError("Alasan penolakan wajib diisi.");
      return;
    }
    setBusy(true);
    try {
      await platformService.rejectWithdrawal(withdrawal.id, trimmed);
      toast.success("Permintaan berhasil ditolak");
      setReason("");
      onDone();
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : "Gagal menolak permintaan. Coba lagi.");
    } finally {
      setBusy(false);
    }
  };

  return (
    <Modal
      open={!!withdrawal}
      onClose={() => {
        setReason("");
        setError("");
        onClose();
      }}
      title="Tolak permintaan pencairan"
      description={
        withdrawal
          ? `Menolak permintaan dari "${withdrawal.tenantName}" sebesar Rp ${withdrawal.amount.toLocaleString("id-ID")}.`
          : ""
      }
    >
      <div className="grid gap-4">
        <div className="grid gap-2">
          <Label>Alasan Penolakan</Label>
          <Textarea
            value={reason}
            onChange={(e) => {
              setReason(e.target.value);
              if (error) setError("");
            }}
            placeholder="mis. Data rekening tidak sesuai"
            aria-invalid={!!error}
          />
          <FieldError msg={error} />
        </div>
        <div className="flex justify-end">
          <Button variant="danger" loading={busy} onClick={submit}>
            Tolak Permintaan
          </Button>
        </div>
      </div>
    </Modal>
  );
}
