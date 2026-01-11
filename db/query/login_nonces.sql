-- name: UpsertLoginNonce :exec
INSERT INTO login_nonces (address, nonce, expires_at, created_at)
VALUES ($1, $2, $3, NOW())
ON CONFLICT (address) DO UPDATE
SET nonce = EXCLUDED.nonce,
    expires_at = EXCLUDED.expires_at;

-- name: GetLoginNonce :one
SELECT address, nonce, expires_at, created_at
FROM login_nonces
WHERE address = $1
LIMIT 1;

-- name: DeleteLoginNonce :exec
DELETE FROM login_nonces
WHERE address = $1;
