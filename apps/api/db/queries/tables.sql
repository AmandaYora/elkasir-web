-- name: ListTables :many
SELECT * FROM dining_tables WHERE store_id = ? ORDER BY code ASC;

-- name: GetTable :one
SELECT * FROM dining_tables WHERE id = ? AND store_id = ? LIMIT 1;

-- name: GetTableByCode :one
SELECT * FROM dining_tables WHERE store_id = ? AND code = ? LIMIT 1;

-- name: CreateTable :exec
INSERT INTO dining_tables (id, store_id, code, name, area, seats, status)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: UpdateTable :exec
UPDATE dining_tables SET code = ?, name = ?, area = ?, seats = ?, status = ?
WHERE id = ? AND store_id = ?;

-- name: DeleteTable :exec
DELETE FROM dining_tables WHERE id = ? AND store_id = ?;
