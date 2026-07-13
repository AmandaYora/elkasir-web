-- name: ListTables :many
SELECT * FROM dining_tables WHERE store_id = ? ORDER BY code ASC;

-- name: GetTable :one
SELECT * FROM dining_tables WHERE id = ? AND store_id = ? LIMIT 1;

-- name: GetTableByCode :one
SELECT * FROM dining_tables WHERE store_id = ? AND code = ? LIMIT 1;

-- Entry point self-order publik (QR discan pelanggan): store BELUM diketahui, jadi resolve
-- lewat slug toko dulu. `code` sendiri cuma unik per-toko (lihat CreateTable), sehingga tanpa
-- join ke stores.slug ini rentan salah-tenant bila 2 toko kebetulan pakai kode meja yang sama.
-- Join ke `stores` sah di sini karena `stores` adalah shared-kernel (lihat DATABASE_GUIDE §2),
-- bukan pelanggaran batas modul.
-- name: FindTableByStoreSlugAndCode :one
SELECT dt.* FROM dining_tables dt
JOIN stores s ON s.id = dt.store_id
WHERE s.slug = ? AND dt.code = ? LIMIT 1;

-- name: CreateTable :exec
INSERT INTO dining_tables (id, store_id, code, name, area, seats, status)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: UpdateTable :exec
UPDATE dining_tables SET code = ?, name = ?, area = ?, seats = ?, status = ?
WHERE id = ? AND store_id = ?;

-- name: DeleteTable :exec
DELETE FROM dining_tables WHERE id = ? AND store_id = ?;
