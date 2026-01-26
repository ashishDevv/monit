-- name: CreateUser :one
INSERT INTO users (name, email, password_hash)
VALUES ($1, $2, $3)
RETURNING id;

-- name: GetUserByID :one
SELECT id, name, email, password_hash, monitors_count, is_paid_user
FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, name, email, password_hash
FROM users
WHERE email = $1;

-- name: IncrementMonitorCount :execrows
UPDATE users
SET monitor_count = monitor_count + 1
WHERE id = $1 AND monitor_count < 10;
