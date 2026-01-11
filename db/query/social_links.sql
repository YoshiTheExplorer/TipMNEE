-- name: CreateSocialLink :one
INSERT INTO social_links (
  user_id, platform, platform_user_id, verified_at, created_at, updated_at
) VALUES (
  $1, $2, $3, NOW(), NOW(), NOW()
)
RETURNING *;

-- name: GetSocialLinkByPlatformUserID :one
SELECT * FROM social_links
WHERE platform = $1 AND platform_user_id = $2
LIMIT 1;

-- name: ListSocialLinksByUser :many
SELECT * FROM social_links
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: UpdateSocialLinkVerification :exec
UPDATE social_links
SET verified_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: DeleteSocialLink :exec
DELETE FROM social_links WHERE id = $1;
