-- 001_init.sql
-- Initial schema for AMZ free shipping tracker

CREATE TABLE IF NOT EXISTS users (
  id BIGSERIAL PRIMARY KEY,
  email TEXT UNIQUE NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS tracked_items (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  source_url TEXT NOT NULL,
  asin VARCHAR(16),
  country_code CHAR(2) NOT NULL,
  zip_code VARCHAR(16),
  active BOOLEAN NOT NULL DEFAULT TRUE,
  check_interval_minutes INT NOT NULL DEFAULT 360,
  next_check_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tracked_items_due ON tracked_items (next_check_at) WHERE active = TRUE;
CREATE INDEX IF NOT EXISTS idx_tracked_items_user ON tracked_items (user_id);

CREATE TABLE IF NOT EXISTS snapshots (
  id BIGSERIAL PRIMARY KEY,
  tracked_item_id BIGINT NOT NULL REFERENCES tracked_items(id) ON DELETE CASCADE,
  free_shipping BOOLEAN NOT NULL,
  free_shipping_country BOOLEAN NOT NULL,
  signal TEXT,
  method TEXT,
  checked_at TIMESTAMPTZ NOT NULL,
  raw_json JSONB
);

CREATE INDEX IF NOT EXISTS idx_snapshots_item_checked ON snapshots (tracked_item_id, checked_at DESC);

CREATE TABLE IF NOT EXISTS alternatives (
  id BIGSERIAL PRIMARY KEY,
  tracked_item_id BIGINT NOT NULL REFERENCES tracked_items(id) ON DELETE CASCADE,
  asin VARCHAR(16),
  url TEXT NOT NULL,
  score NUMERIC(6,3) NOT NULL DEFAULT 0,
  reason TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS alerts (
  id BIGSERIAL PRIMARY KEY,
  tracked_item_id BIGINT NOT NULL REFERENCES tracked_items(id) ON DELETE CASCADE,
  alert_type TEXT NOT NULL,
  payload JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_alerts_item_created ON alerts (tracked_item_id, created_at DESC);

CREATE TABLE IF NOT EXISTS outbox (
  id BIGSERIAL PRIMARY KEY,
  alert_id BIGINT NOT NULL REFERENCES alerts(id) ON DELETE CASCADE,
  channel TEXT NOT NULL,
  idempotency_key TEXT NOT NULL UNIQUE,
  status TEXT NOT NULL DEFAULT 'pending',
  attempt_count INT NOT NULL DEFAULT 0,
  next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_outbox_due ON outbox (status, next_attempt_at);

CREATE TABLE IF NOT EXISTS notification_attempts (
  id BIGSERIAL PRIMARY KEY,
  outbox_id BIGINT NOT NULL REFERENCES outbox(id) ON DELETE CASCADE,
  status TEXT NOT NULL,
  provider_response TEXT,
  attempted_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notification_attempts_outbox ON notification_attempts (outbox_id, attempted_at DESC);
