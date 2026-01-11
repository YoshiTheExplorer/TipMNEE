-- name: CreatePayout :one
INSERT INTO payouts (
  user_id, chain, address, created_at, updated_at
) VALUES (
  $1, $2, $3, NOW(), NOW()
)
RETURNING *;

-- name: GetPayoutByUserAndChain :one
SELECT * FROM payouts
WHERE user_id = $1 AND chain = $2
LIMIT 1;

-- name: UpdatePayoutAddress :one
UPDATE payouts
SET address = $2, updated_at = NOW()
WHERE user_id = $1 AND chain = 'ethereum'
RETURNING *;

-- name: ListPayoutsByUser :many
SELECT * FROM payouts
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: DeletePayout :exec
DELETE FROM payouts WHERE id = $1;
