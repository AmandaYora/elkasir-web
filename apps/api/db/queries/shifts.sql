-- name: GetOpenShift :one
SELECT * FROM shifts WHERE store_id = ? AND status = 'open' ORDER BY opened_at DESC LIMIT 1;

-- name: GetShift :one
SELECT * FROM shifts WHERE id = ? AND store_id = ? LIMIT 1;

-- name: ListShifts :many
SELECT * FROM shifts WHERE store_id = ? ORDER BY opened_at DESC LIMIT ? OFFSET ?;

-- name: CountShifts :one
SELECT COUNT(*) FROM shifts WHERE store_id = ?;

-- name: CreateShift :exec
INSERT INTO shifts (id, store_id, staff_id, status, initial_cash, opened_at)
VALUES (?, ?, ?, 'open', ?, ?);

-- name: CloseShift :execrows
UPDATE shifts SET
  status = 'closed', cash_sales = ?, qris_sales = ?, additional_capital = ?,
  expenses = ?, withdrawals = ?, adjustments = ?, drawer_open_count = ?,
  expected_cash = ?, actual_cash = ?, variance = ?, close_approved_by = ?, closed_at = ?
WHERE id = ? AND store_id = ? AND status = 'open';

-- name: ShiftSalesSummary :one
SELECT
  CAST(COALESCE(SUM(CASE WHEN payment_method = 'cash' THEN total END), 0) AS SIGNED) AS cash_sales,
  CAST(COALESCE(SUM(CASE WHEN payment_method = 'qris' THEN total END), 0) AS SIGNED) AS qris_sales
FROM transactions WHERE shift_id = ? AND status = 'completed';

-- name: ShiftCashMovementSummary :one
SELECT
  CAST(COALESCE(SUM(CASE WHEN type = 'capital' THEN amount END), 0) AS SIGNED) AS capital,
  CAST(COALESCE(SUM(CASE WHEN type = 'expense' THEN amount END), 0) AS SIGNED) AS expense,
  CAST(COALESCE(SUM(CASE WHEN type = 'adjustment' THEN amount END), 0) AS SIGNED) AS adjustment
FROM cash_movements WHERE shift_id = ?;
