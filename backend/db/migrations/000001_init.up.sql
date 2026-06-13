-- 000001_init: enable required PostgreSQL extensions. No tables yet (Phase 1).
-- pgcrypto provides gen_random_uuid() and crypto helpers used in later phases.
CREATE EXTENSION IF NOT EXISTS pgcrypto;
