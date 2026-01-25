package scheduler

import (
	// "time"

	"github.com/google/uuid"
)

type JobPayload struct {
	MonitorID uuid.UUID
	// ScheduleTime time.Time
}