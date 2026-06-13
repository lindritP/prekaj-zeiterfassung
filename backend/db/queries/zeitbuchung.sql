-- name: StartZeitbuchung :one
-- Neue laufende Buchung (end_zeit NULL, pause Default 0). Verstößt gegen den
-- Partial-Unique-Index, wenn bereits eine läuft -> 23505 (Handler -> 409).
INSERT INTO zeitbuchung (id, arbeiter_id, baustelle_id, start_zeit, notiz)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: StopZeitbuchung :one
-- Beendet die EINE laufende Buchung des Arbeiters. 0 Zeilen => keine laufende.
-- pause_minuten wird app-seitig statutarisch berechnet (AZG §11) übergeben.
UPDATE zeitbuchung
   SET end_zeit      = sqlc.arg('end_zeit')::timestamptz,
       pause_minuten = sqlc.arg('pause_minuten')::integer,
       updated_at    = now()
 WHERE arbeiter_id = sqlc.arg('arbeiter_id')
   AND end_zeit IS NULL
RETURNING *;

-- name: GetRunningForArbeiter :one
-- Die aktuell laufende Buchung (falls vorhanden). pgx.ErrNoRows => keine.
SELECT * FROM zeitbuchung
WHERE arbeiter_id = $1 AND end_zeit IS NULL;

-- name: GetZeitbuchungByIDForArbeiter :one
-- Ownership-Scope: liefert nur, wenn die Buchung dem Arbeiter gehört.
SELECT * FROM zeitbuchung
WHERE id = sqlc.arg('id') AND arbeiter_id = sqlc.arg('arbeiter_id');

-- name: ListOwnZeitbuchung :many
-- Eigene Buchungen, optionaler Zeitraum-Filter über start_zeit. Neueste zuerst.
SELECT * FROM zeitbuchung
WHERE arbeiter_id = sqlc.arg('arbeiter_id')
  AND (sqlc.narg('von')::timestamptz IS NULL OR start_zeit >= sqlc.narg('von')::timestamptz)
  AND (sqlc.narg('bis')::timestamptz IS NULL OR start_zeit <  sqlc.narg('bis')::timestamptz)
ORDER BY start_zeit DESC;

-- name: UpdateZeitbuchung :one
-- Partial update (COALESCE). Ownership-gescoped. Dauer-Prüfung app-seitig + DB-CHECK.
UPDATE zeitbuchung
   SET start_zeit    = COALESCE(sqlc.narg('start_zeit')::timestamptz, start_zeit),
       end_zeit      = COALESCE(sqlc.narg('end_zeit')::timestamptz,   end_zeit),
       baustelle_id  = COALESCE(sqlc.narg('baustelle_id')::uuid,      baustelle_id),
       pause_minuten = COALESCE(sqlc.narg('pause_minuten')::integer,  pause_minuten),
       notiz         = COALESCE(sqlc.narg('notiz')::text,             notiz),
       updated_at    = now()
 WHERE id = sqlc.arg('id') AND arbeiter_id = sqlc.arg('arbeiter_id')
RETURNING *;

-- name: AdminListZeitbuchung :many
-- Admin-weite Liste mit optionalen Filtern. Neueste zuerst.
SELECT * FROM zeitbuchung
WHERE (sqlc.narg('arbeiter_id')::uuid  IS NULL OR arbeiter_id  = sqlc.narg('arbeiter_id')::uuid)
  AND (sqlc.narg('baustelle_id')::uuid IS NULL OR baustelle_id = sqlc.narg('baustelle_id')::uuid)
  AND (sqlc.narg('von')::timestamptz   IS NULL OR start_zeit  >= sqlc.narg('von')::timestamptz)
  AND (sqlc.narg('bis')::timestamptz   IS NULL OR start_zeit  <  sqlc.narg('bis')::timestamptz)
ORDER BY start_zeit DESC;

-- name: AdminSumZeitbuchung :one
-- Gesamtsumme der abgeschlossenen Dauer (Minuten) für die gefilterte Menge.
-- Laufende Buchungen (end_zeit NULL) zählen nicht mit.
SELECT
    COALESCE(SUM(
        CAST(EXTRACT(EPOCH FROM (end_zeit - start_zeit)) / 60 AS integer) - pause_minuten
    ), 0)::bigint AS summe_minuten,
    COUNT(*)::bigint AS anzahl
FROM zeitbuchung
WHERE end_zeit IS NOT NULL
  AND (sqlc.narg('arbeiter_id')::uuid  IS NULL OR arbeiter_id  = sqlc.narg('arbeiter_id')::uuid)
  AND (sqlc.narg('baustelle_id')::uuid IS NULL OR baustelle_id = sqlc.narg('baustelle_id')::uuid)
  AND (sqlc.narg('von')::timestamptz   IS NULL OR start_zeit  >= sqlc.narg('von')::timestamptz)
  AND (sqlc.narg('bis')::timestamptz   IS NULL OR start_zeit  <  sqlc.narg('bis')::timestamptz);

-- name: AdminSumZeitbuchungPerArbeiter :many
-- Summe je Arbeiter. Gleiche Filter wie oben.
SELECT
    arbeiter_id,
    COALESCE(SUM(
        CAST(EXTRACT(EPOCH FROM (end_zeit - start_zeit)) / 60 AS integer) - pause_minuten
    ), 0)::bigint AS summe_minuten,
    COUNT(*)::bigint AS anzahl
FROM zeitbuchung
WHERE end_zeit IS NOT NULL
  AND (sqlc.narg('arbeiter_id')::uuid  IS NULL OR arbeiter_id  = sqlc.narg('arbeiter_id')::uuid)
  AND (sqlc.narg('baustelle_id')::uuid IS NULL OR baustelle_id = sqlc.narg('baustelle_id')::uuid)
  AND (sqlc.narg('von')::timestamptz   IS NULL OR start_zeit  >= sqlc.narg('von')::timestamptz)
  AND (sqlc.narg('bis')::timestamptz   IS NULL OR start_zeit  <  sqlc.narg('bis')::timestamptz)
GROUP BY arbeiter_id
ORDER BY arbeiter_id;
