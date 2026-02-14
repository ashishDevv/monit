package monitor

import (
	"context"
	"project-k/pkg/apperror"
	"project-k/pkg/db"
	"project-k/pkg/utils"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type Repository struct {
	querier *db.Queries
	log     *zerolog.Logger
}

func NewRepository(dbExecutor db.DBTX, logger *zerolog.Logger) *Repository {
	return &Repository{
		querier: db.New(dbExecutor),
		log:     logger,
	}
}

func (r *Repository) Create(ctx context.Context, monitor CreateMonitorCmd) (uuid.UUID, error) {
	const op string = "repo.monitor.create"

	monitorID, err := r.querier.CreateMonitor(ctx, db.CreateMonitorParams{
		UserID:             utils.ToPgUUID(monitor.UserID),
		Url:                monitor.Url,
		IntervalSec:        monitor.IntervalSec,
		TimeoutSec:         monitor.TimeoutSec,
		LatencyThresholdMs: monitor.LatencyThresholdMs,
		ExpectedStatus:     monitor.ExpectedStatus,
		AlertEmail:         utils.ToPgText(monitor.AlertEmail),
	})
	if err == nil {
		return utils.FromPgUUID(monitorID), nil
	}

	return uuid.UUID{}, utils.WrapRepoError(op, err, false, r.log)
}

func (r *Repository) GetByID(ctx context.Context, monitorID uuid.UUID) (Monitor, error) {
	const op string = "repo.monitor.get_by_id"

	monitor, err := r.querier.GetMonitorByID(ctx, utils.ToPgUUID(monitorID))
	if err == nil {
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

	return Monitor{}, utils.WrapRepoError(op, err, true, r.log)
}

func (r *Repository) Get(ctx context.Context, userID, monitorID uuid.UUID) (Monitor, error) {
	const op string = "repo.monitor.get"

	monitor, err := r.querier.GetMonitor(ctx, db.GetMonitorParams{
		ID:     utils.ToPgUUID(monitorID),
		UserID: utils.ToPgUUID(userID),
	})
	if err == nil {
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

	return Monitor{}, utils.WrapRepoError(op, err, true, r.log)
}

func (r *Repository) GetAll(ctx context.Context, userID uuid.UUID, limit int32, offset int32) ([]Monitor, error) {
	const op string = "repo.monitor.get_all"

	monitors, err := r.querier.GetAllMonitorByUserID(ctx, db.GetAllMonitorByUserIDParams{
		UserID: utils.ToPgUUID(userID),
		Limit:  limit,
		Offset: offset,
	})
	if err == nil {
		if len(monitors) == 0 {
			return []Monitor{}, nil
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

	// handle errors

	// if no row present  -> list endpoints of sqlc dont return this err, they return emty list ex: []T{}
	// if errors.Is(err, pgx.ErrNoRows) {
	// 	return nil, &apperror.Error{
	// 		Kind:    apperror.NotFound,
	// 		Op:      op,
	// 		Message: "Monitor not found",
	// 	}
	// }

	return []Monitor{}, utils.WrapRepoError(op, err, false, r.log)
}

// SetEnabled can enable or disable the monitor in DB, To enable, pass enable parameter as true, To disable pass it as false
func (r *Repository) SetEnabled(ctx context.Context, userID, monitorID uuid.UUID, enabled bool) error {
	const op string = "repo.monitor.enable_disable_monitor"

	rows, err := r.querier.UpdateMonitorStatus(ctx, db.UpdateMonitorStatusParams{
		ID:      utils.ToPgUUID(monitorID),
		UserID:  utils.ToPgUUID(userID),
		Enabled: enabled,
	})
	if err == nil {
		if rows == 0 {
			return &apperror.Error{
				Kind:    apperror.NotFound,
				Op:      op,
				Message: "resource not found",
			}
		}
		return nil
	}

	return utils.WrapRepoError(op, err, false, r.log)
}
