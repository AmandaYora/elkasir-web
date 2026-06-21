-- name: ListCategories :many
SELECT c.*, (SELECT COUNT(*) FROM products p WHERE p.category_id = c.id) AS product_count
FROM product_categories c
WHERE c.store_id = ?
ORDER BY c.sort_order ASC, c.name ASC;

-- name: GetCategory :one
SELECT * FROM product_categories WHERE id = ? AND store_id = ? LIMIT 1;

-- name: CreateCategory :exec
INSERT INTO product_categories (id, store_id, name, sort_order) VALUES (?, ?, ?, ?);

-- name: UpdateCategory :exec
UPDATE product_categories SET name = ?, sort_order = ? WHERE id = ? AND store_id = ?;

-- name: DeleteCategory :exec
DELETE FROM product_categories WHERE id = ? AND store_id = ?;
