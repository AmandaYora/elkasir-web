-- Tabel `stores` dimiliki shared-kernel; kolom siklus-hidup (slug, status) dibaca/ditulis
-- modul `platform` sebagai pengecualian shared-kernel kedua (setelah `settings` untuk kolom
-- profil) — lihat knowledge/MODULE_MAP.md.

-- name: ListStores :many
SELECT id, name, slug, status, created_at FROM stores ORDER BY created_at DESC;

-- name: GetStoreByID :one
SELECT id, name, slug, status, created_at FROM stores WHERE id = ? LIMIT 1;

-- name: UpdateStoreStatus :execrows
UPDATE stores SET status = ? WHERE id = ?;
