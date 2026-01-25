-- name: CreateMonitorIncident :exec
INSERT INTO monitor_incidents (monitor_id, start_time, alerted, http_status, latency_ms)
VALUES ($1, $2, $3, $4, $5);

-- name: GetMonitorIncidentByID :one
SELECT id, monitor_id, start_time, end_time, alerted, http_status, latency_ms, created_at
FROM monitor_incidents
WHERE id = $1;
