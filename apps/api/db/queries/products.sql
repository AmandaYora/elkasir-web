-- name: GetProduct :one
SELECT * FROM products WHERE id = ? AND store_id = ? LIMIT 1;

-- name: CreateProduct :exec
INSERT INTO products (id, store_id, category_id, sku, name, price, cost, stock, status, image_url)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateProduct :exec
UPDATE products
SET category_id = ?, sku = ?, name = ?, price = ?, cost = ?, stock = ?, status = ?, image_url = ?
WHERE id = ? AND store_id = ?;

-- name: DeleteProduct :exec
DELETE FROM products WHERE id = ? AND store_id = ?;

-- name: AdjustProductStock :execrows
UPDATE products SET stock = stock + ? WHERE id = ? AND store_id = ?;
