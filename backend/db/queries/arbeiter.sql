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
