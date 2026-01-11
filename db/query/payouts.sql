-- name: UpsertPayout :one
INSERT INTO payouts (
  user_id, chain, address, created_at, updated_at
) VALUES (
  $1, $2, $3, NOW(), NOW()
)
ON CONFLICT (user_id, chain) DO UPDATE
SET address = EXCLUDED.address,
    updated_at = NOW()
RETURNING id, user_id, chain, address, created_at, updated_at;

-- name: ResolvePayoutByChannelID :one
SELECT p.address
FROM social_links sl
JOIN payouts p ON p.user_id = sl.user_id
WHERE sl.platform = $1
  AND sl.platform_user_id = $2
  AND p.chain = $3
LIMIT 1;
