-- name: CreateRefreshToken :one
INSERT INTO refresh_token (id, arbeiter_id, token_hash, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetRefreshTokenByHash :one
SELECT * FROM refresh_token WHERE token_hash = $1;

-- name: RevokeRefreshToken :exec
UPDATE refresh_token
   SET revoked_at = now()
 WHERE id = $1 AND revoked_at IS NULL;

-- name: RevokeAllRefreshTokensForArbeiter :exec
UPDATE refresh_token
   SET revoked_at = now()
 WHERE arbeiter_id = $1 AND revoked_at IS NULL;
