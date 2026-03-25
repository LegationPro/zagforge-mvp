-- name: UpsertUser :one
INSERT INTO users (zitadel_user_id, username, email, email_verified, phone, avatar_url)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (zitadel_user_id) DO UPDATE
    SET username       = EXCLUDED.username,
        email          = EXCLUDED.email,
        email_verified = EXCLUDED.email_verified,
        phone          = EXCLUDED.phone,
        avatar_url     = EXCLUDED.avatar_url
RETURNING *;

-- name: GetUserByZitadelID :one
SELECT * FROM users WHERE zitadel_user_id = $1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: UpdateUser :one
UPDATE users
SET username       = COALESCE(NULLIF($2, ''), username),
    email          = COALESCE(NULLIF($3, ''), email),
    email_verified = $4,
    phone          = $5,
    avatar_url     = $6
WHERE id = $1
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;
