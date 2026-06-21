-- name: GetAdminUserByEmail :one
SELECT * FROM admin_users WHERE email = ? LIMIT 1;

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

-- name: GetSettingsByStore :one
SELECT * FROM settings WHERE store_id = ? LIMIT 1;
