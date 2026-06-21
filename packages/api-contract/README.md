# @elkasir/contract

**`openapi.yaml` adalah sumber kebenaran tipe API Elkasir.** Web (sekarang) dan
mobile (nanti) meng-generate client dari berkas ini — bukan menulis ulang tipe.

## Alur "ubah spec → regenerate → pakai"

1. Edit `openapi.yaml` (tambah/ubah path & schema).
2. `task gen` (atau `npm run gen` di paket ini) → meng-generate `generated/ts/schema.d.ts`.
3. Di `apps/web`, import client typed:

   ```ts
   import { api } from "@elkasir/contract";
   const { data, error } = await api.GET("/products", { params: { query: { status: "active" } } });
   ```

`generated/ts/` **di-commit** agar web bisa di-build tanpa langkah generate.

## Isi

| Berkas | Peran |
| --- | --- |
| `openapi.yaml` | spesifikasi OpenAPI 3.1 (tulis manual) |
| `generated/ts/schema.d.ts` | tipe `paths`/`components` hasil `openapi-typescript` |
| `src/index.ts` | factory `createApiClient()` (openapi-fetch) + re-export tipe |

## Lint spec (opsional)

```
npx @redocly/cli lint openapi.yaml
```
