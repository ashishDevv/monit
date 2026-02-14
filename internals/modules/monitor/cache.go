package monitor

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Cache interface {
	GetMonitor(ctx context.Context, id string) ([]byte, error)
	SetMonitor(ctx context.Context, id string, data []byte, ttl time.Duration) error
	Schedule(ctx context.Context, monitorID string, runAt time.Time) error 
	ClearIncident(ctx context.Context, monitorID uuid.UUID) error
	DelMonitor(ctx context.Context, id string) error
	DelStatus(ctx context.Context, monitorID uuid.UUID) error
	DelSchedule(ctx context.Context, monitorID string) error
}
