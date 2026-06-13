-- name: HealthCheck :one
-- Placeholder so sqlc has at least one query to generate from in Phase 1
-- (sqlc errors on an empty queries dir). Replace/extend with real queries in Phase 2.
SELECT 1 AS ok;
