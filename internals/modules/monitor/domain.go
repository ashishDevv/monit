package monitor

import (
	"time"

	"github.com/google/uuid"
)

type CreateMonitorCmd struct {
	UserID             uuid.UUID
	Url                string
	IntervalSec        int32
	TimeoutSec         int32
	LatencyThresholdMs int32
	ExpectedStatus     int32
	AlertEmail         string
}

type Monitor struct {
	ID                 uuid.UUID
	UserID             uuid.UUID
	Url                string
	AlertEmail         string
	IntervalSec        int32
	TimeoutSec         int32
	LatencyThresholdMs int32
	ExpectedStatus     int32
	Enabled            bool
}

type MonitorRecord struct {
	ID                 uuid.UUID
	UserID             uuid.UUID
	Url                string
	IntervalSec        time.Duration
	TimeoutSec         time.Duration
	LatencyThresholdMS time.Duration
	ExpectedStatus     int
	Enabled            bool
	Disabled           bool
}
