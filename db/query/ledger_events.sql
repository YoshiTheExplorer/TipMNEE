-- name: GetEarningsSummaryForUser :one
SELECT
  COALESCE(SUM(CASE WHEN event_type IN ('TIP_DIRECT','TIP_ESCROW') THEN amount_raw ELSE 0 END), 0)::text AS earned_raw,
  COALESCE(SUM(CASE WHEN event_type = 'WITHDRAW' THEN amount_raw ELSE 0 END), 0)::text AS withdrawn_raw,
  (
    COALESCE(SUM(CASE WHEN event_type IN ('TIP_DIRECT','TIP_ESCROW') THEN amount_raw ELSE 0 END), 0)
    -
    COALESCE(SUM(CASE WHEN event_type = 'WITHDRAW' THEN amount_raw ELSE 0 END), 0)
  )::text AS pending_raw
FROM ledger_events
WHERE user_id = $1::bigint;

-- name: ListTipsForUser :many
SELECT
  id, platform, platform_user_id, user_id, event_type, amount_raw, message,
  tx_hash, log_index, block_time, created_at, updated_at
FROM ledger_events
WHERE user_id = $1
  AND event_type IN ('TIP_DIRECT', 'TIP_ESCROW')
ORDER BY block_time DESC
LIMIT $2 OFFSET $3;

-- name: BackfillLedgerEventsUserIDForChannel :exec
UPDATE ledger_events
SET user_id = $1,
    updated_at = NOW()
WHERE platform = $2
  AND platform_user_id = $3
  AND user_id IS NULL;

-- name: InsertLedgerEvent :one
INSERT INTO ledger_events (
  platform, platform_user_id, user_id, event_type, amount_raw, message,
  tx_hash, log_index, block_time, created_at, updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6,
  $7, $8, $9, NOW(), NOW()
)
ON CONFLICT (tx_hash, log_index) DO NOTHING
RETURNING
  id, platform, platform_user_id, user_id, event_type, amount_raw, message,
  tx_hash, log_index, block_time, created_at, updated_at;