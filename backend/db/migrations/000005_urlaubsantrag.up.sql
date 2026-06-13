-- 000005_urlaubsantrag: Urlaubs-/Abwesenheitsanträge.
-- UUIDs in Go erzeugt (uuid v7). Ganze Tage (date, keine Uhrzeit/Zeitzone).
-- Status-Automat: offen -> genehmigt | abgelehnt (Übergang nur aus 'offen').
CREATE TABLE urlaubsantrag (
    id              uuid        PRIMARY KEY,
    arbeiter_id     uuid        NOT NULL REFERENCES arbeiter(id) ON DELETE CASCADE,
    von_datum       date        NOT NULL,
    bis_datum       date        NOT NULL,
    typ             text        NOT NULL DEFAULT 'urlaub'
                                CHECK (typ IN ('urlaub', 'krankheit', 'sonstige')),
    status          text        NOT NULL DEFAULT 'offen'
                                CHECK (status IN ('offen', 'genehmigt', 'abgelehnt')),
    grund           text        NOT NULL DEFAULT '',
    entschieden_von uuid                 REFERENCES arbeiter(id),  -- der entscheidende Admin
    entschieden_am  timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT urlaubsantrag_bis_nach_von CHECK (bis_datum >= von_datum)
);

CREATE INDEX idx_urlaubsantrag_arbeiter ON urlaubsantrag (arbeiter_id, von_datum DESC);
CREATE INDEX idx_urlaubsantrag_status   ON urlaubsantrag (status);
