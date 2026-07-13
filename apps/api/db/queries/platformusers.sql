-- name: ListPlatformUsers :many
SELECT * FROM platform_users ORDER BY created_at ASC;

-- name: GetPlatformUser :one
SELECT * FROM platform_users WHERE id = ? LIMIT 1;

-- name: CreatePlatformUser :exec
INSERT INTO platform_users (id, name, email, password_hash, status)
VALUES (?, ?, ?, ?, ?);

-- name: SetPlatformUserStatus :execrows
UPDATE platform_users SET status = ? WHERE id = ?;

-- name: UpdatePlatformUserPassword :exec
UPDATE platform_users SET password_hash = ? WHERE id = ?;
