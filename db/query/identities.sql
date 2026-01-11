-- name: CreateIdentity :one
INSERT INTO identities (
  user_id, provider, provider_user_id, created_at, updated_at
) VALUES (
  $1, $2, $3, NOW(), NOW()
)
RETURNING *;

-- name: GetIdentityByProviderAndUserID :one
SELECT * FROM identities
WHERE provider = $1 AND provider_user_id = $2
LIMIT 1;

-- name: ListIdentitiesByUser :many
SELECT * FROM identities
WHERE user_id = $1;

-- name: DeleteIdentity :exec
DELETE FROM identities WHERE id = $1;
