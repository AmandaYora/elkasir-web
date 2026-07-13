-- name: ListWithdrawals :many
SELECT * FROM withdrawals WHERE store_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: CountWithdrawals :one
SELECT COUNT(*) FROM withdrawals WHERE store_id = ?;

-- name: CreateWithdrawal :exec
INSERT INTO withdrawals (id, store_id, amount, bank, account, holder, status, reference, requested_by)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetWithdrawal :one
SELECT * FROM withdrawals WHERE id = ? LIMIT 1;

-- name: ListActiveWithdrawals :many
-- Cross-tenant, superadmin Penarikan page (PLAN.md §2.7) — pending + processing only.
SELECT * FROM withdrawals WHERE status IN ('pending','processing') ORDER BY created_at ASC;

-- name: ListAllWithdrawals :many
-- Cross-tenant, superadmin Riwayat Penarikan page — any status, paginated.
SELECT * FROM withdrawals ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: CountAllWithdrawals :one
SELECT COUNT(*) FROM withdrawals;

-- name: SumSuccessfulWithdrawals :one
-- Cross-tenant total of status='success' withdrawals — feeds Ringkasan (§2.6's AvailableBalance
-- is per-tenant; this is the platform-wide figure for GET /platform/revenue).
SELECT CAST(COALESCE(SUM(amount), 0) AS SIGNED) FROM withdrawals WHERE status = 'success';

-- name: SumSuccessfulWithdrawalsByStore :one
-- §2.6 AvailableBalance basis for one tenant.
SELECT CAST(COALESCE(SUM(amount), 0) AS SIGNED) FROM withdrawals WHERE status = 'success' AND store_id = ?;

-- name: SumSuccessfulWithdrawalsGroupedByStore :many
-- §2.6 AvailableBalance basis, all tenants at once (Revenue Tenant page).
SELECT store_id, CAST(COALESCE(SUM(amount), 0) AS SIGNED) AS total
FROM withdrawals WHERE status = 'success' GROUP BY store_id;

-- name: SumProcessingWithdrawalsByStore :one
-- §2.6 claimable-check basis (narrower than AvailableBalance) for one tenant.
SELECT CAST(COALESCE(SUM(amount), 0) AS SIGNED) FROM withdrawals WHERE status = 'processing' AND store_id = ?;

-- Klaim (pending -> processing), §2.7. Atomic conditional UPDATE — 0 rows affected means the
-- request was no longer pending (already claimed/rejected by someone else).
-- name: ClaimWithdrawal :execrows
UPDATE withdrawals SET status = 'processing', processed_by = ?, claimed_at = ?
WHERE id = ? AND status = 'pending';

-- Tandai Sukses (processing -> success), §2.7. processed_by must match the claimant — 0 rows
-- means either not processing anymore, or the acting principal isn't who claimed it.
-- name: MarkWithdrawalSuccess :execrows
UPDATE withdrawals SET status = 'success', processed_at = ?
WHERE id = ? AND status = 'processing' AND processed_by = ?;

-- Tolak (pending|processing -> failed), §2.7. Any active superadmin, no ownership restriction.
-- name: MarkWithdrawalRejected :execrows
UPDATE withdrawals SET status = 'failed', processed_by = ?, processed_at = ?, rejected_reason = ?
WHERE id = ? AND status IN ('pending','processing');
