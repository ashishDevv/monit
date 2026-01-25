package alert

import "github.com/google/uuid"

type AlertEvent struct {
	MonitorID uuid.UUID
}