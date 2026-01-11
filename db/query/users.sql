-- name: CreateUser :one
INSERT INTO users (created_at, updated_at)
VALUES (NOW(), NOW())
RETURNING id, created_at, updated_at;

-- name: GetUserByID :one
SELECT id, created_at, updated_at
FROM users
WHERE id = $1
LIMIT 1;
