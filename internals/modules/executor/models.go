package executor

import (
	"time"

	"github.com/google/uuid"
)

type HTTPResult struct {
	MonitorID uuid.UUID
	Status    int
	LatencyMs int64
	Success   bool
	Reason    string
	Retryable bool
	CheckedAt time.Time
}
