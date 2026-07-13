# Production Migration & Cutover Plan — `feat/tenant-subscription-billing` → production

**Status: analisis selesai, deploy sedang dieksekusi mengikuti checklist di bawah.** Ditulis
karena production masih berjalan di kode **sebelum** seluruh PLAN.md Part 1/2/3 (Konsol
Platform, subscription billing, payment gateway multi-app, external payment API), sementara
branch ini menambahkan perubahan skema besar (migrasi 015–021). Dokumen ini memetakan persis apa
yang terjadi pada tenant yang sudah live saat migrasi dijalankan, dan apa yang wajib disiapkan
dulu sebelum deploy supaya tenant itu tidak rusak/ke-lockout/kehilangan data.

> Detail operasional dengan identitas production asli (hostname, path server, data tenant nyata)
> disimpan terpisah secara lokal (tidak di-commit — lihat `.gitignore`: `*.local.md`), karena repo
> ini public. Dokumen ini adalah versi aman-dipublikasikan: prosedur dan alasan teknisnya lengkap,
> tanpa detail infrastruktur/pelanggan yang tidak perlu terekspos publik.

**Update:** paket backfill migrasi 018 didesain sebagai paket berbayar nyata bernama **"Premium
Contributor"** (Rp1.700.000/tahun, kode `premium-contributor`), **terkunci** (tidak bisa
pindah/upgrade ke paket lain, hanya bisa diperpanjang — ditegakkan di level backend,
`subscription/application.Service.validatePlanSwitch`), dengan masa aktif awal **365 hari** sejak
migrasi dijalankan — bukan paket gratis tak terbatas. Lihat detail keputusan di `PLAN.md`
§2.15/Phase B1.5.

---

## 0. Apa yang terjadi pada tenant existing, langkah demi langkah

Asumsi: seluruh checklist §4.A (backup, `CONFIG_ENCRYPTION_KEY`, cek payment pending) sudah
dikerjakan dulu sebelum deploy dijalankan.

**1) Saat migrasi jalan (015→021), dalam hitungan detik, tanpa intervensi manual:**
- Tenant yang sudah ada **tidak hilang, tidak terduplikasi, tidak diubah namanya**.
- Baris `stores` miliknya otomatis dapat `slug` (diturunkan dari nama toko) dan `status='active'`
  (default kolom baru, bukan `'suspended'`) — **jadi tidak langsung ke-suspend oleh fitur baru**.
- Baris `store_subscriptions` baru otomatis dibuat untuknya oleh migrasi 018 (`status='active'`,
  `current_period_end` +365 hari, paket **"Premium Contributor"** tersembunyi dari pemilihan
  tenant lain, terkunci hanya-bisa-perpanjang) — **inilah yang mencegah tenant ini ke-lockout
  oleh gate langganan yang baru diaktifkan**, dan memberinya 1 tahun pertama gratis sebelum
  perpanjangan berbayar pertama.
- Akun admin & staff yang sudah ada **tidak tersentuh** — hanya kolom baru nullable ditambahkan
  di tabel-tabel terkait, tidak ada baris lama yang berubah nilainya.

**2) Saat container baru start, pada boot pertama:**
- Modul `payment` mendeteksi `payment_gateway_config` kosong → otomatis membaca kredensial
  gateway yang masih ada di `.env` sekarang, mengenkripsinya, menyimpan sebagai baris pertama.
  **Kredensial gateway asli yang sedang dipakai pindah dari `.env` ke database** — mulai saat ini
  perubahan kredensial harus lewat Konsol Platform, bukan edit `.env` lagi.
- Middleware auth mulai mengecek status suspend + status paket pada **setiap request** — untuk
  tenant ini kedua cek itu **lolos** (langkah 1 di atas), jadi secara fungsional tidak ada
  perubahan yang terasa oleh admin/staff.

**3) Yang langsung dirasakan oleh pengguna toko, begitu deploy selesai:**
- ✅ Admin toko login seperti biasa — **tidak ada perubahan**, hanya sekarang ada menu baru
  "Langganan" di sidebar menampilkan paket "Premium Contributor" (Rp1.700.000/tahun) dengan sisa
  waktu ~365 hari, tanpa opsi upgrade/pindah paket sama sekali — SEMUA paket lain tersembunyi
  (ditegakkan eksplisit lewat field `planRenewalOnly` yang dikirim backend), hanya ada tombol
  "Perpanjang".
- ✅ Akun staff login di app POS — **tidak ada perubahan**, tetap bisa jualan.
- 🟢 **QR code self-order fisik belum dipasang/dipakai secara nyata** (dikonfirmasi user) — format
  URL-nya berubah (§3.3), tapi karena belum ada QR fisik yang beredar, **tidak ada pelanggan yang
  terdampak**. Tinggal cetak & pasang kapan pun sebelum self-order mulai dipakai, tidak mendesak.
- ⚪ Data pembayaran lama yang sudah lama `pending` (data uji lama, sudah pasti kedaluwarsa di
  sisi gateway) tetap diam di database sebagai "pending" — bukan masalah baru, sudah begitu sejak
  sebelum deploy.

**4) Setelah `seed` dijalankan sekali (sekarang aman, §3.1):**
- Akun superadmin pertama muncul di `platform_users` — **tenant existing tidak terpengaruh sama
  sekali oleh langkah ini** (fix §3.1 memastikan seed tidak lagi mencoba bikin toko baru karena
  satu toko sudah ada).
- Login Konsol Platform → halaman "Tenant" menampilkan **persis 1 baris** — sesuai syarat "hanya
  1 tenant" yang diminta di awal.
- Halaman "Revenue Tenant"/"Ringkasan" mulai menampilkan angka nyata dari histori transaksi
  tenant ini (bukan data kosong/dummy).

**Ringkasnya:** secara data & akses, tenant existing **selamat sepenuhnya** — tidak hilang, tidak
ke-lockout, tidak terduplikasi, kredensial gateway-nya termigrasi otomatis. Dengan QR meja
dikonfirmasi belum dipakai, **tidak ada satu pun dampak yang benar-benar terasa oleh
pengguna/pelanggan** dari proses migrasi ini — seluruhnya berjalan transparan di belakang layar
bagi admin dan staff yang sudah ada.

---

## 1. State saat ini vs target

| | **Production (sebelum)** | **Target (branch ini)** |
|---|---|---|
| Tenant | **1** tenant existing | tetap 1, sekarang dengan `slug`+`status` |
| `stores.slug`/`status` | tidak ada kolomnya sama sekali | ada, wajib, unik |
| Subscription/billing | tidak ada modulnya | ada — tenant lama di-backfill ke paket berbayar "Premium Contributor" (Rp1.700.000/tahun, terkunci, 365 hari pertama gratis) |
| Konsol Platform (`platform_users`) | tabel tidak ada, 0 akun | ada — superadmin login |
| Payment gateway config | murni dari `.env` | DB-backed, terenkripsi (`CONFIG_ENCRYPTION_KEY`) |
| Self-order QR URL | 1 segmen path, belum dipasang/dipakai | 2 segmen (slug + kode meja) — non-issue, belum ada QR fisik beredar |
| Backup otomatis | belum ada, hanya manual | belum berubah, masih harus disiapkan manual sebelum deploy |

---

## 2. Audit keamanan data per migrasi (015→021)

Dibaca langsung dari `apps/api/db/migrations/`. Kesimpulan per migrasi terhadap baris tenant yang
sudah ada di production:

| # | Migrasi | Efek ke data lama | Aman? |
|---|---|---|---|
| 015 | `subscription_billing` | 3 tabel baru (`subscription_plans` [termasuk kolom `renewal_only`, default `0`], `store_subscriptions`, `subscription_invoices`), tidak menyentuh tabel lama | ✅ Aman |
| 016 | `platform` | `platform_users` baru; `stores` dapat `slug`/`status`. **Slug di-backfill OTOMATIS dalam migrasi yang sama** sebelum constraint `NOT NULL UNIQUE` ditegakkan. `staff.username` diubah dari unik-per-toko jadi unik-global — diverifikasi tidak ada duplikat username di data production → migrasi ini tidak akan gagal. | ✅ Aman (sudah dicek datanya) |
| 017 | `withdrawal_processing` | 4 kolom baru di `withdrawals`, semua `NULL`-able | ✅ Aman |
| 018 | `subscription_legacy_backfill` | Insert plan **"Premium Contributor"** (Rp1.700.000/tahun, `is_active=0` tersembunyi, `renewal_only=1` terkunci) + insert otomatis 1 baris `store_subscriptions status='active'` untuk SETIAP store yang belum punya baris — tenant existing otomatis kebagian, `current_period_end` = **+365 hari** | ✅ Aman — inilah mekanisme yang mencegah tenant lama ke-lockout, diverifikasi ulang dengan migrasi + unit test end-to-end |
| 019 | `payment_gateway_registry` | 3 tabel baru (`payment_clients`, `payment_gateway_config`, `payment_charge_apps`), seed 2 baris internal (self-order & subscription consumer). `payment_gateway_config` **kosong** setelah migrasi — baru terisi saat app boot pertama kali (lihat §3.1) | ✅ Aman secara schema, tapi ada syarat env di §3.2 |
| 020 | `payment_external_api` | 1 kolom baru (`secret_enc`), nullable | ✅ Aman |
| 021 | `payment_charge_provider_ref` | 1 kolom baru (`provider_ref`), nullable | ✅ Aman |

**Kesimpulan:** ketujuh migrasi secara *schema* 100% aman untuk dijalankan di atas data production
saat ini — tidak ada `NOT NULL` tanpa default/backfill yang bisa gagal, tidak ada data yang
tertimpa. Risiko yang nyata semuanya ada di **level aplikasi**, dijelaskan di §3.

---

## 3. Temuan kritis — WAJIB ditangani sebelum deploy

Diurutkan dari paling berbahaya.

### 3.1 🔴 [SUDAH DIPERBAIKI] Bug di `bootstrap.Seed()` — bisa membuat tenant hantu ke-2

**File:** `apps/api/internal/platform/bootstrap/seed.go`

`ensureStore()` yang lama mencari toko dengan nama hardcode bawaan seed (`"Elkasir"`). Karena
toko production bernama sesuatu yang lain, menjalankan `seed` di production akan:

1. Tidak menemukan baris manapun yang cocok → **membuat toko BARU dengan nama bootstrap default**
   → melanggar langsung syarat "hanya 1 tenant".
2. Lalu insert admin bootstrap baru (email/username default, password default) — kredensial ini
   **belum dipakai** di production sehingga insert-nya akan berhasil tanpa error dan diam-diam
   menanam akun owner berkredensial default publik.

Ini bukan skenario hipotetis — `seed` adalah salah satu dari 4 subcommand resmi image production,
jadi siapa pun yang suatu saat berpikir "perlu jalankan seed untuk isi superadmin" akan memicu bug
ini tanpa sadar.

**Perbaikan (sudah diterapkan, `go build`/`go vet` bersih, diuji idempoten):** `Seed()` sekarang
cek apakah **sudah ada toko apa pun** — bila ya, langkah bootstrap toko contoh **dilewati total**.
Katalog `subscription_plans` dan **superadmin platform tetap selalu di-upsert** (idempoten by
email) di kedua kondisi — inilah jalur yang justru dibutuhkan di production untuk mengisi akun
superadmin pertama.

➡️ **Konsekuensi untuk deploy:** setelah migrasi jalan, jalankan `seed` sekali di production —
**ini sekarang aman** dan diperlukan untuk mengisi akun superadmin pertama. Wajib ganti password
default itu segera setelah login pertama.

### 3.2 🔴 `CONFIG_ENCRYPTION_KEY` — wajib diset sebelum boot pertama

Config ini mengenkripsi kredensial payment gateway yang disimpan di database (AES-256-GCM). Kode
punya nilai default untuk kemudahan development yang **secara sengaja tidak aman untuk
production** dan **tertulis terbuka di source code** — kalau env var ini tidak diset eksplisit di
production, validasi konfigurasi tetap lolos (tidak fatal/crash) tapi kredensial gateway asli
akan terenkripsi memakai kunci yang bisa diturunkan siapa pun yang membaca repo ini.

➡️ **Wajib sebelum deploy:** generate key acak sungguhan dan tambahkan ke `.env` production
**SEBELUM** boot pertama kode baru (sekali `payment_gateway_config` terisi dengan kunci yang
salah, perlu migrasi ulang manual untuk memperbaikinya):

```bash
openssl rand -base64 32
# tambahkan ke .env production:
CONFIG_ENCRYPTION_KEY=<hasil-di-atas>
```

### 3.3 🟢 [DIKONFIRMASI NON-ISSUE] URL QR self-order berubah format

Format URL self-order publik berubah dari 1 segmen path (kode meja saja) menjadi 2 segmen (slug
toko + kode meja) — kode meja saja tidak cukup unik lintas-tenant, jadi slug toko wajib jadi
bagian URL (lihat PLAN.md §1). Slug tenant existing diturunkan otomatis dari nama tokonya oleh
migrasi 016.

**Dikonfirmasi user: QR code fisik di meja BELUM dipasang/dipakai secara nyata saat ini** — jadi
perubahan format URL ini tidak berdampak ke pelanggan mana pun, tidak ada QR fisik yang perlu
ditarik/diganti mendadak. Tidak blocking, tidak perlu disiapkan sebelum deploy. Cukup cetak &
pasang QR (format baru) kapan pun sebelum fitur self-order mulai benar-benar dipakai — generate
dari halaman Meja di admin setelah deploy, otomatis memakai format baru.

**Tidak ada migrasi data untuk baris `dining_tables` itu sendiri, dan memang tidak diperlukan** —
skemanya (`dining_tables.store_id`) sudah benar sejak awal. Pencarian meja publik yang baru
bekerja lewat JOIN `dining_tables.store_id = stores.id AND stores.slug = ?` — begitu migrasi 016
mem-backfill slug tenant ini, baris meja yang ada otomatis langsung bisa ditemukan lewat slug
baru, tanpa satu baris pun di `dining_tables` perlu disentuh.

### 3.4 🟡 Celah dispatch webhook untuk pembayaran yang "in-flight" saat cutover

Dispatch webhook yang baru mencari `order_ref` di tabel indeks baru (kosong di production sampai
charge PERTAMA dibuat lewat kode baru). Tidak ada fallback: charge QRIS yang **dibuat sebelum
deploy** (lewat kode lama, tidak tercatat di index baru) tapi **baru dibayar/webhook masuk
setelah deploy** akan gagal di-dispatch — gateway akan retry webhook itu berulang kali lalu
menyerah, dan transaksi/stok tidak akan pernah ter-update meski pembayaran sukses di sisi
gateway.

**Dicek langsung ke production:** hanya ada 1 baris pembayaran lama berstatus `pending`, sudah
sangat lama dan pasti kedaluwarsa di sisi gateway (sisa data uji awal, bukan transaksi nyata).
**Tidak ada risiko nyata saat ini**, tapi ini kondisi *pada saat audit* — harus dicek ulang tepat
sebelum eksekusi (lihat checklist §4).

➡️ **Mitigasi:** (a) jalankan deploy di jam tutup/sepi pelanggan, (b) cek ulang query di §4 tepat
sebelum deploy — pastikan 0 baris `pending` yang baru (dibuat dalam beberapa jam terakhir).

### 3.5 🟢 Item non-blocking (dicek, aman/opsional)

- **SMTP** — tidak dikonfigurasi di production. Fitur notifikasi email withdrawal no-op dengan
  aman bila kosong, tidak menghambat deploy. Rekomendasi: isi nanti agar superadmin dapat
  notifikasi email permintaan pencairan.
- **`JWT_APP_TOKEN_TTL`** — tidak ada di `.env` production, tapi kode punya default 1 jam → aman,
  tidak fatal.
- **Backup otomatis** — belum pernah dipasang di production. Ada 1 backup manual lama, sudah
  cukup lawas. Tidak blocking, tapi harus dibuat backup baru manual tepat sebelum deploy (§4) dan
  disarankan pasang cron nightly backup sekalian.

---

## 4. Checklist eksekusi (urutan wajib)

### A. Sebelum deploy (di server production)

- [ ] **Backup database segar** sebelum migrasi apa pun dijalankan.
- [ ] **Tambahkan `CONFIG_ENCRYPTION_KEY`** ke `.env` production (§3.2) — generate baru dengan
  `openssl rand -base64 32`, JANGAN pakai nilai contoh mana pun.
- [ ] **Cek ulang tidak ada pembayaran QRIS `pending` yang baru** (§3.4) — kalau ada baris baru
  (dalam beberapa jam terakhir), tunggu selesai/kedaluwarsa dulu atau jadwalkan deploy di jam lain.
- [ ] Pilih jam tutup/sepi untuk eksekusi (mitigasi §3.4 dan jendela downtime singkat saat
  restart container).

### B. Deploy (mengikuti runbook `deploy.sh` yang sudah ada, tidak berubah)

- [ ] CI hijau di `main`.
- [ ] Jalankan `deploy.sh` dengan git sha yang sudah lulus CI — otomatis: pull image → migrate up
  (015-021) → restart container → cek `/readyz`.
  - Skrip ini abort-on-error: kalau migrasi gagal, container lama TIDAK diganti (masih serve
    versi lama) — aman, tapi database bisa "dirty", harus diperiksa manual sebelum retry.
  - Kalau migrasi sukses tapi `/readyz` gagal: container sudah terlanjur diganti — situs down,
    harus manual rollback (lihat §5), tidak otomatis.
- [ ] Jalankan `seed` **sekali** untuk mengisi akun superadmin pertama (sekarang aman, §3.1).

### C. Setelah deploy — verifikasi

- [ ] Healthcheck endpoint publik → 200.
- [ ] Login admin toko existing dengan kredensial lama — harus tetap bisa login, tidak ke-402/403
  (memverifikasi backfill "Premium Contributor" migrasi 018 jalan benar).
- [ ] Cek `store_subscriptions` production: harus ada 1 baris `status='active'`,
  `current_period_end` ~365 hari ke depan, plan mengarah ke `premium-contributor`.
- [ ] Buka halaman Langganan sebagai admin toko: nama paket "Premium Contributor" tampil, harga
  Rp1.700.000 tampil benar, tidak ada section "Upgrade Paket", tombol "Perpanjang" muncul.
- [ ] Login staff di app POS — pastikan tidak ke-block oleh gate paket-tidak-aktif.
- [ ] Login superadmin baru di Konsol Platform → segera ganti password default.
- [ ] *(Tidak mendesak, §3.3)* Cetak & pasang QR code format baru kapan pun sebelum self-order
  mulai dipakai nyata.
- [ ] Buat 1 transaksi self-order QRIS uji nyata (nominal kecil) dari QR baru → bayar → pastikan
  webhook masuk, transaksi tercatat, stok berkurang (uji end-to-end jalur dispatch baru §3.4).
- [ ] Cek `payment_gateway_config` production terisi 1 baris dengan field non-secret sesuai
  (jangan pernah select kolom terenkripsi untuk verifikasi, hanya untuk cek baris ada).
- [ ] Konsol Platform → Konfigurasi Pembayaran → pastikan field tersamarkan tampil benar, BUKAN
  kosong/`undefined` (indikasi `CONFIG_ENCRYPTION_KEY` salah/berubah).

---

## 5. Rencana rollback

- **Rollback image** (bukan DB): deploy ulang dengan sha lama — migrasi forward-only, jadi image
  lama akan jalan di atas schema baru. Semua migrasi 015–021 di atas aman backward-compatible
  (kolom baru nullable/berdefault, tabel baru tidak disentuh kode lama) — kode lama akan tetap
  berjalan normal, hanya tidak akan memakai fitur-fitur baru.
- **Rollback DB** — tidak direkomendasikan/tidak disiapkan di skenario ini karena migrasi 018
  sudah menulis data nyata (baris subscription legacy) yang kalau di-migrate-down akan terhapus
  lalu perlu backfill ulang. Kalau harus benar-benar mundur, restore dari backup §4.A, bukan
  migrate down.
- QR meja tidak relevan untuk skenario rollback saat ini — belum ada QR fisik yang beredar
  (§3.3), jadi tidak ada tautan lama yang perlu dijaga tetap berfungsi.

---

## 6. Ringkasan — hal yang harus disiapkan di luar kode

1. Generate & pasang `CONFIG_ENCRYPTION_KEY` di `.env` production (§3.2).
2. Putuskan jam deploy (idealnya jam tutup/sepi, untuk mitigasi §3.4).
3. Review checklist §4 lengkap sebelum eksekusi deploy.
4. *(Tidak mendesak)* Cetak & pasang QR meja format baru (§3.3) kapan pun sebelum self-order
   mulai dipakai nyata.

Perbaikan kode di §3.1 (`bootstrap.Seed`) sudah diterapkan di branch ini — tidak perlu tindakan
tambahan selain memastikan perubahan itu ikut ter-deploy.
