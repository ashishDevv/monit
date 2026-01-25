package result

import (
	"context"
	"project-k/internals/modules/executor"
	"project-k/pkg/db"
	"project-k/pkg/utils"

	"github.com/google/uuid"
)

type IncidentRepository struct {
	querier *db.Queries
}

func NewMonitorIncidentRepo(dbExecutor db.DBTX) *IncidentRepository {
	return &IncidentRepository{
		querier: db.New(dbExecutor),
	}
}

func (r *IncidentRepository) Create(ctx context.Context, r executor.HTTPResult) {

}

func (r *IncidentRepository) Get(ctx context.Context, incidentID uuid.UUID) (MonitorIncident, error) {
	mI, err := r.querier.GetMonitorIncidentByID(ctx, utils.ToPgUUID(incidentID))
	if err != nil {
		return MonitorIncident{}, err
	}
	

	return MonitorIncident{
		ID: utils.FromPgUUID(mI.ID),
		MonitorID: utils.FromPgUUID(mI.MonitorID),
		StartTime: ,
	}, nil
}
