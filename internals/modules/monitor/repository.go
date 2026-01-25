package monitor

import (
	"context"
	"project-k/pkg/db"
	"project-k/pkg/utils"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type Repository struct {
	querier *db.Queries
}

func NewRepository(dbExecutor db.DBTX) *Repository {
	return &Repository{
		querier: db.New(dbExecutor),
	}
}

func (r *Repository) Create(ctx context.Context, monitor CreateMonitorCmd) (uuid.UUID, error) {
	monitorID, err := r.querier.CreateMonitor(ctx, db.CreateMonitorParams{
		UserID:             utils.ToPgUUID(monitor.UserID),
		Url:                monitor.Url,
		IntervalSec:        monitor.IntervalSec,
		TimeoutSec:         monitor.TimeoutSec,
		LatencyThresholdMs: monitor.LatencyThresholdMs,
		ExpectedStatus:     monitor.ExpectedStatus,
		AlertEmail:         utils.ToPgText(monitor.AlertEmail),
	})
	if err != nil {
		return uuid.UUID{}, err
	}
	return utils.FromPgUUID(monitorID), nil
}

func (r *Repository) GetByID(ctx context.Context, monitorID uuid.UUID) (Monitor, error) {

	monitor, err := r.querier.GetMonitorByID(ctx, pgtype.UUID{
		Bytes: monitorID,
		Valid: true,
	})
	if err != nil {
		return Monitor{}, err
	}

	return Monitor{
		ID:                 utils.FromPgUUID(monitor.ID),
		UserID:             utils.FromPgUUID(monitor.UserID),
		Url:                monitor.Url,
		IntervalSec:        monitor.IntervalSec,
		TimeoutSec:         monitor.TimeoutSec,
		LatencyThresholdMs: monitor.LatencyThresholdMs,
		ExpectedStatus:     monitor.ExpectedStatus,
		Enabled:            monitor.Enabled,
		AlertEmail:         utils.FromPgText(monitor.AlertEmail),
	}, nil
}

func (r *Repository) Get(ctx context.Context, userID, monitorID uuid.UUID) (Monitor, error) {
	monitor, err := r.querier.GetMonitor(ctx, db.GetMonitorParams{
		ID: utils.ToPgUUID(monitorID),
		UserID: utils.ToPgUUID(userID),
	})
	if err != nil {
		return Monitor{}, err
	}

	return Monitor{
		ID:                 utils.FromPgUUID(monitor.ID),
		UserID:             utils.FromPgUUID(monitor.UserID),
		Url:                monitor.Url,
		IntervalSec:        monitor.IntervalSec,
		TimeoutSec:         monitor.TimeoutSec,
		LatencyThresholdMs: monitor.LatencyThresholdMs,
		ExpectedStatus:     monitor.ExpectedStatus,
		Enabled:            monitor.Enabled,
		AlertEmail:         utils.FromPgText(monitor.AlertEmail),
	}, nil
}


func (r *Repository) GetAll(ctx context.Context, userID uuid.UUID, limit int32, offset int32) ([]Monitor, error) {
	monitors, err := r.querier.GetAllMonitorByUserID(ctx, db.GetAllMonitorByUserIDParams{
		UserID: utils.ToPgUUID(userID),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, err
	}
	m := make([]Monitor, 0, len(monitors))
	for i := range monitors {
		mon := &monitors[i]
		m = append(m, Monitor{
			ID:                 utils.FromPgUUID(mon.ID),
			UserID:             utils.FromPgUUID(mon.UserID),
			Url:                mon.Url,
			IntervalSec:        mon.IntervalSec,
			TimeoutSec:         mon.TimeoutSec,
			LatencyThresholdMs: mon.LatencyThresholdMs,
			ExpectedStatus:     mon.ExpectedStatus,
			Enabled:            mon.Enabled,
			AlertEmail:         utils.FromPgText(mon.AlertEmail),
		})
	}

	return m, nil
}

func (r *Repository) EnableDisableMonitor(ctx context.Context, userID, monitorID uuid.UUID, enabled bool) (bool, error) {
	return r.querier.UpdateMonitorStatus(ctx, db.UpdateMonitorStatusParams{
		ID: utils.ToPgUUID(monitorID),
		UserID: utils.ToPgUUID(userID),
		Enabled: enabled,
	})
}
