import { api, BASE_URL } from "@/shared/services/http-client";
import { endpoints } from "@/shared/services/api-endpoints";
import type {
  PlaceOrderInput,
  PlaceResult,
  PublicMenu,
  PublicSelfOrderStatus,
  QuoteResult,
} from "@/modules/self-order/types/self-order.types";

// Public (no-auth) order service for the customer self-order page.
// Every call passes { auth: false } so no Bearer token is attached.
// {storeSlug} wajib: kode meja cuma unik per-toko (lihat migration 000016), jadi tenant
// harus di-resolve dari slug toko, bukan cuma kode meja.
const orderPath = (storeSlug: string, tableCode: string) =>
  `${endpoints.publicOrder}/${encodeURIComponent(storeSlug)}/${encodeURIComponent(tableCode)}`;

export const publicOrderService = {
  menu: (storeSlug: string, tableCode: string) =>
    api.get<PublicMenu>(orderPath(storeSlug, tableCode), { auth: false }),
  place: (storeSlug: string, tableCode: string, body: PlaceOrderInput) =>
    api.post<PlaceResult>(orderPath(storeSlug, tableCode), body, { auth: false }),
  quote: (storeSlug: string, tableCode: string, body: PlaceOrderInput) =>
    api.post<QuoteResult>(`${orderPath(storeSlug, tableCode)}/quote`, body, { auth: false }),
  status: (selfOrderId: string) =>
    api.get<PublicSelfOrderStatus>(`${endpoints.publicOrder}/status/${selfOrderId}`, {
      auth: false,
    }),
  // Berlangganan perubahan status pembayaran via Server-Sent Events (pengganti polling).
  // Server mem-push status begitu callback gateway menandai lunas; EventSource otomatis
  // reconnect bila koneksi terputus, dan handler server selalu mengirim snapshot saat
  // tersambung sehingga tidak ada event yang terlewat. Mengembalikan fungsi untuk menutup
  // koneksi. Endpoint publik (tanpa token) — EventSource memang tidak mengirim header auth.
  subscribeStatus(
    selfOrderId: string,
    onStatus: (status: PublicSelfOrderStatus) => void,
  ): () => void {
    const es = new EventSource(`${BASE_URL}${endpoints.publicOrder}/events/${selfOrderId}`);
    const handle = (data: string) => {
      try {
        onStatus(JSON.parse(data) as PublicSelfOrderStatus);
      } catch {
        /* abaikan payload tak valid */
      }
    };
    es.addEventListener("status", (e) => handle((e as MessageEvent).data));
    es.onmessage = (e) => handle(e.data); // fallback bila event tak bernama
    return () => es.close();
  },
  simulatePaid: (selfOrderId: string) =>
    api.post<void>(`${endpoints.publicOrder}/${selfOrderId}/simulate-paid`, undefined, {
      auth: false,
    }),
};
