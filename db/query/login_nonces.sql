-- name: UpsertLoginNonce :exec
INSERT INTO login_nonces (address, nonce, expires_at, message, created_at)
VALUES ($1, $2, $3, $4, NOW())
ON CONFLICT (address) DO UPDATE
SET nonce = EXCLUDED.nonce,
    expires_at = EXCLUDED.expires_at,
    message = EXCLUDED.message;

-- name: GetLoginNonce :one
SELECT address, nonce, expires_at, message, created_at
FROM login_nonces
WHERE address = $1
LIMIT 1;

-- name: DeleteLoginNonce :exec
DELETE FROM login_nonces
WHERE address = $1;
