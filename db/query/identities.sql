-- name: GetIdentity :one
SELECT id, user_id, provider, provider_user_id, created_at, updated_at
FROM identities
WHERE provider = $1 AND provider_user_id = $2
LIMIT 1;

-- name: CreateIdentity :one
INSERT INTO identities (
  user_id, provider, provider_user_id, created_at, updated_at
) VALUES (
  $1, $2, $3, NOW(), NOW()
)
RETURNING id, user_id, provider, provider_user_id, created_at, updated_at;

-- name: ListIdentitiesByUser :many
SELECT id, user_id, provider, provider_user_id, created_at, updated_at
FROM identities
WHERE user_id = $1
ORDER BY id DESC;
