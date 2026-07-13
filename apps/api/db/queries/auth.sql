-- name: GetAdminUserByEmail :one
SELECT * FROM admin_users WHERE email = ? LIMIT 1;

-- name: GetAdminUserByEmailOrUsername :one
SELECT * FROM admin_users WHERE email = ? OR username = ? LIMIT 1;

-- name: GetAdminUserByID :one
SELECT * FROM admin_users WHERE id = ? LIMIT 1;

-- name: TouchAdminUserLastActive :exec
UPDATE admin_users SET last_active_at = ? WHERE id = ?;

-- name: GetStaffByUsername :one
SELECT * FROM staff WHERE username = ? LIMIT 1;

-- name: GetStaffByID :one
SELECT * FROM staff WHERE id = ? LIMIT 1;

-- name: CreateRefreshToken :exec
INSERT INTO refresh_tokens (id, actor, subject_id, token_hash, expires_at)
VALUES (?, ?, ?, ?, ?);

-- name: GetRefreshToken :one
SELECT * FROM refresh_tokens WHERE token_hash = ? LIMIT 1;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens SET revoked_at = ? WHERE token_hash = ? AND revoked_at IS NULL;

-- name: GetFirstStore :one
SELECT * FROM stores ORDER BY created_at ASC LIMIT 1;

-- name: GetPlatformUserByEmail :one
SELECT * FROM platform_users WHERE email = ? LIMIT 1;

-- name: GetPlatformUserByID :one
SELECT * FROM platform_users WHERE id = ? LIMIT 1;

-- name: GetStoreStatus :one
-- Bacaan langsung ke `stores.status` — pengecualian shared-kernel yang sama classnya dengan
-- kolom profil milik `settings` (lihat MODULE_MAP.md); dipakai utk gerbang suspensi tenant
-- (PLAN.md §2.13), bukan kepemilikan tabel oleh `auth`.
SELECT status FROM stores WHERE id = ? LIMIT 1;

-- name: GetPaymentClientForAppLogin :one
-- Bacaan langsung ke `payment_clients` — pola yang SAMA persis dengan GetPlatformUserByEmail di
-- atas (auth punya query login-lookup sendiri ke tabel identitas modul lain; CRUD tetap milik
-- `payment`, bukan `auth`). Dipakai HANYA untuk POST /auth/app/token (PLAN.md §10.1.2/§10.1.3).
SELECT id, app_id, secret_hash, status FROM payment_clients WHERE app_id = ? AND kind = 'external' LIMIT 1;

-- name: GetPaymentClientStatus :one
-- Bacaan langsung ke `payment_clients.status` — pengecualian shared-kernel yang sama classnya
-- dengan GetStoreStatus di atas; dipakai utk cek status LIVE per-request utk ActorApp
-- (PLAN.md §10.1.4), bukan kepemilikan tabel oleh `auth`.
SELECT status FROM payment_clients WHERE id = ? LIMIT 1;
