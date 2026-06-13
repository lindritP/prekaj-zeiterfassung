-- name: CreateDokument :one
INSERT INTO dokument (id, arbeiter_id, typ, jahr, monat, dateiname, storage_key, mime_type, groesse)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: ListOwnDokument :many
SELECT * FROM dokument WHERE arbeiter_id = $1 ORDER BY created_at DESC;

-- name: GetDokumentByID :one
SELECT * FROM dokument WHERE id = $1;

-- name: AdminListDokument :many
SELECT * FROM dokument
WHERE (sqlc.narg('arbeiter_id')::uuid IS NULL OR arbeiter_id = sqlc.narg('arbeiter_id')::uuid)
ORDER BY created_at DESC;
