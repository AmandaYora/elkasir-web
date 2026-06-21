-- name: ListWithdrawals :many
SELECT * FROM withdrawals WHERE store_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: CountWithdrawals :one
SELECT COUNT(*) FROM withdrawals WHERE store_id = ?;

-- name: CreateWithdrawal :exec
INSERT INTO withdrawals (id, store_id, amount, bank, account, holder, status, reference, requested_by)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);
