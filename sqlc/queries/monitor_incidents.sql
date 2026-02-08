-- name: CreateMonitorIncident :exec
INSERT INTO monitor_incidents (monitor_id, start_time, alerted, http_status, latency_ms)
VALUES ($1, $2, $3, $4, $5);

-- name: GetMonitorIncidentByID :one
SELECT id, monitor_id, start_time, end_time, alerted, http_status, latency_ms, created_at
FROM monitor_incidents
WHERE id = $1;

-- name: CloseMonitorIncident :execrows
UPDATE monitor_incidents
SET end_time = $2
WHERE monitor_id = $1 AND end_time IS NULL;
