-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS monitor_incidents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    monitor_id UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NULL,
    alerted BOOLEAN NOT NULL DEFAULT false,
    http_status INT NOT NULL CHECK (status BETWEEN 100 AND 599),
    latency_ms INT NOT NULL CHECK (latency_ms >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (end_time IS NULL OR end_time >= start_time)
);

CREATE INDEX idx_monitor_incidents_monitor_id
ON monitor_incidents (monitor_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS monitor_incidents;
-- +goose StatementEnd
