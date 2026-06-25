-- name: ListStaff :many
SELECT * FROM staff WHERE store_id = ? ORDER BY created_at DESC;

-- name: GetStaffScoped :one
SELECT * FROM staff WHERE id = ? AND store_id = ? LIMIT 1;

-- name: CreateStaff :exec
INSERT INTO staff (id, store_id, name, username, email, password_hash, role, status)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateStaff :exec
UPDATE staff SET name = ?, username = ?, email = ?, role = ?, status = ?
WHERE id = ? AND store_id = ?;

-- name: UpdateStaffPassword :exec
UPDATE staff SET password_hash = ? WHERE id = ? AND store_id = ?;

-- name: UpdateStaffPin :exec
UPDATE staff SET pin_hash = ? WHERE id = ? AND store_id = ?;

-- name: ListSupervisorPins :many
SELECT id, name, pin_hash FROM staff
WHERE store_id = ? AND role = 'supervisor' AND status = 'active' AND pin_hash IS NOT NULL;

-- name: DeleteStaff :exec
DELETE FROM staff WHERE id = ? AND store_id = ?;
