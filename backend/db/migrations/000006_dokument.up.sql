-- 000006_dokument: hochgeladene Dokumente (Lohnzettel etc.).
-- UUIDs in Go erzeugt. Datei liegt im Storage (storage_key); DB hält nur Metadaten.
CREATE TABLE dokument (
    id          uuid        PRIMARY KEY,
    arbeiter_id uuid        NOT NULL REFERENCES arbeiter(id) ON DELETE CASCADE,
    typ         text        NOT NULL DEFAULT 'lohnzettel'
                            CHECK (typ IN ('lohnzettel', 'sonstige')),
    jahr        integer     NOT NULL,
    monat       integer     NOT NULL CHECK (monat BETWEEN 1 AND 12),
    dateiname   text        NOT NULL,            -- Original-Dateiname (Anzeige/Download)
    storage_key text        NOT NULL UNIQUE,     -- interner Speicher-Key (uuid-basiert)
    mime_type   text        NOT NULL DEFAULT 'application/pdf',
    groesse     bigint      NOT NULL DEFAULT 0,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_dokument_arbeiter ON dokument (arbeiter_id, created_at DESC);
