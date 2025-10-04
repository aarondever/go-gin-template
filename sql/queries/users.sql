-- name: CreateUser :one
INSERT INTO users (username, password_hash)
VALUES ($1, $2)
RETURNING *;

-- name: GetUser :one
SELECT *
FROM users
WHERE id = $1;

-- name: GetUsers :many
SELECT *
FROM users
LIMIT $1 OFFSET $2;

-- name: GetUserCount :one
SELECT COUNT(*)
FROM users;

-- name: GetUserByUsername :one
SELECT *
FROM users
WHERE username = $1;

-- name: UpdateUser :one
UPDATE users
SET username   = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;
