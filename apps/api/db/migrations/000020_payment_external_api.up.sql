-- Part 3 (PLAN.md §10, EB0): external-facing payment API. `secret_enc` stores the SAME plaintext
-- secret as `secret_hash` (bcrypt), but reversibly encrypted (AES-256-GCM, same
-- CONFIG_ENCRYPTION_KEY-derived helper already built in Part 2 for gateway credentials) — needed
-- because HMAC-signing an outbound webhook relay requires recovering the plaintext, which a
-- one-way bcrypt hash structurally cannot do (§10.1.6). NULL for kind='internal' rows, same
-- convention as secret_hash.
ALTER TABLE payment_clients ADD COLUMN secret_enc VARBINARY(500) NULL AFTER secret_hash;
