-- 002_alternative_searches.sql
-- Persistent cache of PA-API search results keyed on source ASIN.
-- Forward-compatible with pgvector: embedding column is nullable TEXT for now,
-- change type to vector(1536) after enabling the extension.

CREATE TABLE IF NOT EXISTS alternative_searches (
  id            BIGSERIAL    PRIMARY KEY,
  source_asin   VARCHAR(16)  NOT NULL,
  search_query  TEXT         NOT NULL,
  results_json  JSONB        NOT NULL,
  result_count  SMALLINT     NOT NULL DEFAULT 0,
  created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  expires_at    TIMESTAMPTZ  NOT NULL,
  -- RAG-forward fields (all nullable, unused until pgvector phase)
  embedding     TEXT         NULL,
  embed_model   VARCHAR(64)  NULL,
  embed_at      TIMESTAMPTZ  NULL
);

-- Fast lookup by source ASIN (the primary cache key)
CREATE UNIQUE INDEX IF NOT EXISTS idx_alt_searches_asin
  ON alternative_searches (source_asin);

-- Allow efficient purge of expired rows
CREATE INDEX IF NOT EXISTS idx_alt_searches_expires
  ON alternative_searches (expires_at);
