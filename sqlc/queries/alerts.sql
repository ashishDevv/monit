-- name: CreateAlert :exec
INSERT INTO
alerts (incident_id, alert_email) 
VALUES ($1, $2);

-- name: UpdateAlertStatus :execrows
UPDATE alerts
SET
    status = $2
WHERE id = $1;