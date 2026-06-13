-- name: ListBaustellen :many
-- Optionaler aktiv-Filter: NULL => alle. Stabile Sortierung nach Name.
SELECT * FROM baustelle
WHERE (sqlc.narg('aktiv')::boolean IS NULL OR aktiv = sqlc.narg('aktiv')::boolean)
ORDER BY name ASC, created_at ASC;

-- name: GetBaustelleByID :one
SELECT * FROM baustelle WHERE id = $1;

-- name: CreateBaustelle :one
INSERT INTO baustelle (id, name, adresse, aktiv)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateBaustelle :one
-- Partial update: nur gesetzte (non-NULL) Felder werden geändert (COALESCE).
UPDATE baustelle
   SET name       = COALESCE(sqlc.narg('name')::text, name),
       adresse    = COALESCE(sqlc.narg('adresse')::text, adresse),
       aktiv      = COALESCE(sqlc.narg('aktiv')::boolean, aktiv),
       updated_at = now()
 WHERE id = sqlc.arg('id')
RETURNING *;

-- name: DeactivateBaustelle :one
-- Soft-Delete: idempotent (mehrfaches Deaktivieren ändert nur updated_at).
UPDATE baustelle
   SET aktiv = false, updated_at = now()
 WHERE id = $1
RETURNING *;
