-- name: GetSocialLinkByPlatformUser :one
SELECT id, user_id, platform, platform_user_id, verified_at, created_at, updated_at
FROM social_links
WHERE platform = $1 AND platform_user_id = $2
LIMIT 1;

-- name: CreateSocialLink :one
INSERT INTO social_links (
  user_id, platform, platform_user_id, verified_at, created_at, updated_at
) VALUES (
  $1, $2, $3, $4, NOW(), NOW()
)
RETURNING id, user_id, platform, platform_user_id, verified_at, created_at, updated_at;
