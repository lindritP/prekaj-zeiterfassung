-- name: GetArbeiterByEmail :one
SELECT * FROM arbeiter WHERE email = $1;

-- name: GetArbeiterByID :one
SELECT * FROM arbeiter WHERE id = $1;

-- name: CreateArbeiter :one
INSERT INTO arbeiter (id, name, email, passwort_hash, rolle, wochenstunden, stundenlohn, aktiv)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: UpsertAdmin :one
-- Idempotenter Seed: legt den Inhaber an oder aktualisiert Name/Passwort, falls die E-Mail existiert.
INSERT INTO arbeiter (id, name, email, passwort_hash, rolle, aktiv)
VALUES ($1, $2, $3, $4, 'admin', true)
ON CONFLICT (email) DO UPDATE
   SET name = EXCLUDED.name,
       passwort_hash = EXCLUDED.passwort_hash,
       rolle = 'admin',
       aktiv = true,
       updated_at = now()
RETURNING *;

-- name: ListArbeiter :many
-- Optionaler aktiv-Filter: NULL => alle. Sortierung nach Name.
SELECT * FROM arbeiter
WHERE (sqlc.narg('aktiv')::boolean IS NULL OR aktiv = sqlc.narg('aktiv')::boolean)
ORDER BY name ASC, created_at ASC;

-- name: UpdateArbeiter :one
-- Partial update (COALESCE). E-Mail wird im Handler normalisiert übergeben.
UPDATE arbeiter
   SET name          = COALESCE(sqlc.narg('name')::text, name),
       email         = COALESCE(sqlc.narg('email')::text, email),
       rolle         = COALESCE(sqlc.narg('rolle')::text, rolle),
       wochenstunden = COALESCE(sqlc.narg('wochenstunden')::numeric, wochenstunden),
       stundenlohn   = COALESCE(sqlc.narg('stundenlohn')::numeric, stundenlohn),
       passwort_hash = COALESCE(sqlc.narg('passwort_hash')::text, passwort_hash),
       aktiv         = COALESCE(sqlc.narg('aktiv')::boolean, aktiv),
       updated_at    = now()
 WHERE id = sqlc.arg('id')
RETURNING *;

-- name: DeactivateArbeiter :one
-- Soft-Delete: kein Hard-Delete (DSGVO-Löschung folgt in Phase 13). Idempotent.
UPDATE arbeiter
   SET aktiv = false, updated_at = now()
 WHERE id = $1
RETURNING *;
