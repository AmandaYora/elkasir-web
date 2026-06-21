-- name: ListCashMovements :many
SELECT * FROM cash_movements WHERE store_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: CountCashMovements :one
SELECT COUNT(*) FROM cash_movements WHERE store_id = ?;

-- name: GetCashMovement :one
SELECT * FROM cash_movements WHERE id = ? AND store_id = ? LIMIT 1;

-- name: CreateCashMovement :exec
INSERT INTO cash_movements (id, store_id, shift_id, type, amount, notes, created_by, approved_by)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);
