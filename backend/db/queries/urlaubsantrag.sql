-- name: CreateUrlaubsantrag :one
-- status defaultet auf 'offen'.
INSERT INTO urlaubsantrag (id, arbeiter_id, von_datum, bis_datum, typ, grund)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListOwnUrlaubsantrag :many
SELECT * FROM urlaubsantrag
WHERE arbeiter_id = $1
ORDER BY von_datum DESC;

-- name: GetUrlaubsantragByIDForArbeiter :one
-- Ownership-Scope (Arbeiter).
SELECT * FROM urlaubsantrag
WHERE id = sqlc.arg('id') AND arbeiter_id = sqlc.arg('arbeiter_id');

-- name: GetUrlaubsantragByID :one
-- Admin: jeder Antrag.
SELECT * FROM urlaubsantrag WHERE id = $1;

-- name: DeleteUrlaubsantrag :exec
-- Handler stellt sicher, dass status = 'offen' ist (Get-first).
DELETE FROM urlaubsantrag
WHERE id = sqlc.arg('id') AND arbeiter_id = sqlc.arg('arbeiter_id');

-- name: AdminListUrlaubsantrag :many
-- Optionale Filter: status, Zeitraum über von_datum. Neueste Anträge zuerst.
SELECT * FROM urlaubsantrag
WHERE (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
  AND (sqlc.narg('von')::date    IS NULL OR von_datum >= sqlc.narg('von')::date)
  AND (sqlc.narg('bis')::date    IS NULL OR von_datum <= sqlc.narg('bis')::date)
ORDER BY created_at DESC;

-- name: DecideUrlaubsantrag :one
-- Genehmigen/Ablehnen. Übergang nur aus 'offen' (atomar via WHERE).
UPDATE urlaubsantrag
   SET status          = sqlc.arg('status')::text,
       entschieden_von = sqlc.arg('entschieden_von')::uuid,
       entschieden_am  = now()
 WHERE id = sqlc.arg('id') AND status = 'offen'
RETURNING *;
