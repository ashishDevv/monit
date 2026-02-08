package executor

import (
	"time"

	"github.com/google/uuid"
)

type HTTPResult struct {
	MonitorID   uuid.UUID
	Success     bool
	Status      int
	LatencyMs   int64
	Reason      string
	Retryable   bool
	CheckedAt   time.Time
	IntervalSec int32
	AlertEmail  string
}
