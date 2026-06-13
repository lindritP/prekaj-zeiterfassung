-- 000003_baustelle: Stammdaten Baustelle (Einsatzort).
-- UUIDs werden in Go erzeugt (uuid v7, zeitgeordnet) — Spalte ist plain `uuid`, kein DB-Default.
-- adresse NOT NULL DEFAULT '' — wie bei arbeiter werden keine nullable-Textspalten verwendet.
CREATE TABLE baustelle (
    id         uuid        PRIMARY KEY,
    name       text        NOT NULL,
    adresse    text        NOT NULL DEFAULT '',
    aktiv      boolean     NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
