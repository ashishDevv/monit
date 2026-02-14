package result

import (
	"context"
	"project-k/internals/modules/executor"
	"project-k/pkg/apperror"
	"project-k/pkg/db"
	"project-k/pkg/utils"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
)

type MonitorIncidentRepository struct {
	querier *db.Queries
	logger  *zerolog.Logger
}

func NewMonitorIncidentRepo(dbExecutor db.DBTX, logger *zerolog.Logger) *MonitorIncidentRepository {
	return &MonitorIncidentRepository{
		querier: db.New(dbExecutor),
		logger:  logger,
	}
}

func (r *MonitorIncidentRepository) Create(ctx context.Context, startTime time.Time, e executor.HTTPResult) error {
	const op string = "repo.monitor_incident.create"

	err := r.querier.CreateMonitorIncident(ctx, db.CreateMonitorIncidentParams{
		MonitorID:  utils.ToPgUUID(e.MonitorID),
		Alerted:    true,
		HttpStatus: int32(e.Status),
		LatencyMs:  int32(e.LatencyMs),
		StartTime: pgtype.Timestamptz{
			Time:  startTime,
			Valid: true,
		},
	})
	if err == nil {
		return nil
	}

	return utils.WrapRepoError(op, err, false, r.logger)
}

func (r *MonitorIncidentRepository) GetByID(ctx context.Context, incidentID uuid.UUID) (MonitorIncident, error) {
	const op string = "repo.monitor_incident.get"

	mI, err := r.querier.GetMonitorIncidentByID(ctx, utils.ToPgUUID(incidentID))
	if err == nil {
		return MonitorIncident{
			ID:         utils.FromPgUUID(mI.ID),
			MonitorID:  utils.FromPgUUID(mI.MonitorID),
			Alerted:    mI.Alerted,
			HttpStatus: mI.HttpStatus,
			LatencyMs:  mI.LatencyMs,
			StartTime:  utils.FromPgTimestamptz(mI.StartTime),
			CreatedAt:  utils.FromPgTimestamptz(mI.CreatedAt),
			EndTime:    utils.FromPgTimestamptz(mI.EndTime),
		}, nil
	}

	return MonitorIncident{}, utils.WrapRepoError(op, err, true, r.logger)
}

func (r *MonitorIncidentRepository) CloseIncident(ctx context.Context, monitorID uuid.UUID, endTime time.Time) error {
	const op string = "repo.monitor_incident.close_incident"

	rowsAffected, err := r.querier.CloseMonitorIncident(ctx, db.CloseMonitorIncidentParams{
		MonitorID: utils.ToPgUUID(monitorID),
		EndTime:   utils.ToPgTimestamptz(endTime),
	})
	if err == nil {
		if rowsAffected == 0 {
			return &apperror.Error{
				Kind:    apperror.NotFound,
				Op:      op,
				Message: "resource not found",
			}
		}
		return nil
	}

	return utils.WrapRepoError(op, err, false, r.logger)
}
