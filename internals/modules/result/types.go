package result

import (
	"time"

	"github.com/google/uuid"
)

type MonitorIncident struct {
	ID         uuid.UUID
	MonitorID  uuid.UUID
	StartTime  time.Time
	EndTime    time.Time
	Alerted    bool
	HttpStatus int32
	LatencyMs  int32
	CreatedAt  time.Time
}