-- name: ReportSalesSummary :one
-- revenue = total. Tiga bucket keuangan terpisah: penjualan (subtotal−diskon),
-- layanan (service + biaya gateway), pajak (PPN). Ketiganya berjumlah = revenue.
SELECT
  COUNT(*) AS tx_count,
  CAST(COALESCE(SUM(total), 0) AS SIGNED) AS revenue,
  CAST(COALESCE(SUM(subtotal - discount), 0) AS SIGNED) AS sales_total,
  CAST(COALESCE(SUM(service_charge + gateway_fee), 0) AS SIGNED) AS service_total,
  CAST(COALESCE(SUM(tax), 0) AS SIGNED) AS tax_total,
  CAST(COALESCE(SUM(CASE WHEN payment_method = 'cash' THEN total END), 0) AS SIGNED) AS cash_total,
  CAST(COALESCE(SUM(CASE WHEN payment_method = 'qris' THEN total END), 0) AS SIGNED) AS qris_total
FROM transactions
WHERE store_id = ? AND status = 'completed' AND created_at >= ? AND created_at < ?;

-- name: ReportSalesByDay :many
SELECT
  DATE(created_at) AS day,
  COUNT(*) AS tx_count,
  CAST(COALESCE(SUM(total), 0) AS SIGNED) AS revenue
FROM transactions
WHERE store_id = ? AND status = 'completed' AND created_at >= ? AND created_at < ?
GROUP BY DATE(created_at)
ORDER BY day ASC;

-- name: ReportTopProducts :many
SELECT
  ti.product_name AS product_name,
  CAST(COALESCE(SUM(ti.quantity), 0) AS SIGNED) AS qty,
  CAST(COALESCE(SUM(ti.line_total), 0) AS SIGNED) AS revenue
FROM transaction_items ti
JOIN transactions t ON t.id = ti.transaction_id
WHERE t.store_id = ? AND t.status = 'completed' AND t.created_at >= ? AND t.created_at < ?
GROUP BY ti.product_name
ORDER BY qty DESC
LIMIT ?;

-- name: ReportSalesByCategory :many
SELECT
  ti.category AS category,
  CAST(COALESCE(SUM(ti.line_total), 0) AS SIGNED) AS revenue,
  CAST(COALESCE(SUM(ti.quantity), 0) AS SIGNED) AS qty
FROM transaction_items ti
JOIN transactions t ON t.id = ti.transaction_id
WHERE t.store_id = ? AND t.status = 'completed' AND t.created_at >= ? AND t.created_at < ?
GROUP BY ti.category
ORDER BY revenue DESC;

-- name: ReportPaymentDistribution :one
SELECT
  CAST(COALESCE(SUM(CASE WHEN payment_method = 'cash' THEN total END), 0) AS SIGNED) AS cash_total,
  CAST(COALESCE(SUM(CASE WHEN payment_method = 'qris' THEN total END), 0) AS SIGNED) AS qris_total,
  CAST(COALESCE(SUM(CASE WHEN payment_method = 'cash' THEN 1 ELSE 0 END), 0) AS SIGNED) AS cash_count,
  CAST(COALESCE(SUM(CASE WHEN payment_method = 'qris' THEN 1 ELSE 0 END), 0) AS SIGNED) AS qris_count
FROM transactions
WHERE store_id = ? AND status = 'completed' AND created_at >= ? AND created_at < ?;

-- name: ReportStaffPerformance :many
SELECT
  s.id AS staff_id,
  s.name AS name,
  CAST(COUNT(t.id) AS SIGNED) AS tx_count,
  CAST(COALESCE(SUM(t.total), 0) AS SIGNED) AS revenue
FROM staff s
LEFT JOIN transactions t
  ON t.cashier_id = s.id AND t.status = 'completed' AND t.created_at >= ? AND t.created_at < ?
WHERE s.store_id = ?
GROUP BY s.id, s.name
ORDER BY revenue DESC;

-- name: ReportRecentTransactions :many
SELECT id, code, source, payment_method, total, created_at
FROM transactions
WHERE store_id = ? AND status = 'completed'
ORDER BY created_at DESC
LIMIT ?;
