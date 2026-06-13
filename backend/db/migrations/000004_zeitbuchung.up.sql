-- 000004_zeitbuchung: Arbeitszeit-Erfassung (Start/Stopp).
-- UUIDs werden in Go erzeugt (uuid v7) — Spalte ist plain `uuid`, kein DB-Default.
-- Zeiten in UTC (timestamptz); Anzeige-Zeitzone (Europe/Vienna) ist Frontend-Sache.
-- end_zeit IS NULL  => Buchung läuft noch.   baustelle_id NULL => keiner Baustelle zugeordnet.
CREATE TABLE zeitbuchung (
    id            uuid        PRIMARY KEY,
    arbeiter_id   uuid        NOT NULL REFERENCES arbeiter(id)  ON DELETE CASCADE,
    baustelle_id  uuid                 REFERENCES baustelle(id) ON DELETE SET NULL,
    start_zeit    timestamptz NOT NULL,
    end_zeit      timestamptz,
    pause_minuten integer     NOT NULL DEFAULT 0 CHECK (pause_minuten >= 0),
    notiz         text        NOT NULL DEFAULT '',
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),

    -- Über Mitternacht ist in UTC kein Sonderfall: end_zeit > start_zeit genügt.
    -- Verhindert negative Dauer bereits auf Speicherebene.
    CONSTRAINT zeitbuchung_end_after_start
        CHECK (end_zeit IS NULL OR end_zeit > start_zeit)
);

-- Max. EINE laufende Buchung pro Arbeiter: Partial Unique Index greift nur,
-- solange end_zeit IS NULL. Zwei abgeschlossene Buchungen sind erlaubt.
CREATE UNIQUE INDEX uq_zeitbuchung_eine_laufende
    ON zeitbuchung (arbeiter_id)
    WHERE end_zeit IS NULL;

CREATE INDEX idx_zeitbuchung_arbeiter_start ON zeitbuchung (arbeiter_id, start_zeit DESC);
CREATE INDEX idx_zeitbuchung_baustelle      ON zeitbuchung (baustelle_id);
