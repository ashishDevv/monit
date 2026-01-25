-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS monitors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    url TEXT NOT NULL CHECK (length(url) <= 2048),
    alert_email TEXT, 
    interval_sec INT NOT NULL CHECK (interval_sec >= 60),
    timeout_sec INT NOT NULL CHECK (interval_sec > 0),
    latency_threshold_ms INT NOT NULL CHECK (latency_threshold_ms >= 0),
    expected_status INT NOT NULL CHECK (expected_status BETWEEN 100 AND 599),
    enabled BOOLEAN NOT NULL DEFAULT true,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_monitors_user_id ON monitors (user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS monitors;
-- +goose StatementEnd
