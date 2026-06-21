-- name: ListAdminUsers :many
SELECT * FROM admin_users WHERE store_id = ? ORDER BY created_at DESC;

-- name: GetAdminUserScoped :one
SELECT * FROM admin_users WHERE id = ? AND store_id = ? LIMIT 1;

-- name: CreateAdminUser :exec
INSERT INTO admin_users (id, store_id, name, email, password_hash, role, status)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: UpdateAdminUser :exec
UPDATE admin_users SET name = ?, email = ?, role = ?, status = ?
WHERE id = ? AND store_id = ?;

-- name: UpdateAdminUserPassword :exec
UPDATE admin_users SET password_hash = ? WHERE id = ? AND store_id = ?;

-- name: DeleteAdminUser :exec
DELETE FROM admin_users WHERE id = ? AND store_id = ?;
