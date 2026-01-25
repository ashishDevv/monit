-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    incident_id UUID NOT NULL REFERENCES monitor_incidents(id),
    sent_at TIMESTAMPTZ,
    alert_email TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',     -- 'pending', 'sent', 'failed'
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
    -- alert_type TEXT NOT NULL, -- 'DOWN', 'LATENCY'
    -- channel TEXT NOT NULL,    -- 'email', 'slack'
    -- UNIQUE (incident_id, alert_type, channel)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS alerts;
-- +goose StatementEnd
