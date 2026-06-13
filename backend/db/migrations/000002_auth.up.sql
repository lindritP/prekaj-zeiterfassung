-- 000002_auth: core identity tables (Arbeiter = Benutzer) + rotierende Refresh-Tokens.
-- citext: case-insensitive, eindeutige E-Mails auf DB-Ebene (die App lowercased zusätzlich).
-- UUIDs werden in Go erzeugt (uuid v7, zeitgeordnet) — Spalten sind plain `uuid`, kein DB-Default.
CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE arbeiter (
    id            uuid PRIMARY KEY,
    name          text          NOT NULL,
    email         citext        NOT NULL UNIQUE,
    passwort_hash text          NOT NULL,
    rolle         text          NOT NULL DEFAULT 'arbeiter'
                                CHECK (rolle IN ('arbeiter', 'admin')),
    wochenstunden numeric(5,2)  NOT NULL DEFAULT 0,
    stundenlohn   numeric(10,2) NOT NULL DEFAULT 0,
    aktiv         boolean       NOT NULL DEFAULT true,
    created_at    timestamptz   NOT NULL DEFAULT now(),
    updated_at    timestamptz   NOT NULL DEFAULT now()
);

CREATE TABLE refresh_token (
    id          uuid        PRIMARY KEY,
    arbeiter_id uuid        NOT NULL REFERENCES arbeiter(id) ON DELETE CASCADE,
    token_hash  bytea       NOT NULL UNIQUE,   -- sha256(opaque token), 32 Bytes
    expires_at  timestamptz NOT NULL,
    revoked_at  timestamptz,                   -- NULL = aktiv
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_refresh_token_arbeiter ON refresh_token (arbeiter_id);
