-- name: GetWebhookEvent :one
SELECT * FROM webhook_events WHERE provider = ? AND event_id = ? LIMIT 1;

-- name: CreateWebhookEvent :exec
INSERT INTO webhook_events (id, provider, event_id) VALUES (?, ?, ?);
