package executor

import (
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
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

func (h HTTPResult) MarshalZerologObject(e *zerolog.Event) {
	e.
		Str("monitor_id", h.MonitorID.String()).
		Bool("success", h.Success).
		Int("status", h.Status).
		Int64("latency_ms", h.LatencyMs).
		Str("reason", h.Reason).
		Bool("retryable", h.Retryable).
		Time("checked_at", h.CheckedAt).
		Int32("interval_sec", h.IntervalSec).
		Str("alert_email", h.AlertEmail)
}
