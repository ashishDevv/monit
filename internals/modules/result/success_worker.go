package result

import (
	"time"

	"project-k/internals/modules/executor"
)

func (rp *ResultProcessor) successWorker() {
	defer rp.workerWG.Done()

	for r := range rp.successChan {
		rp.handleSuccess(r)
	}
}

func (rp *ResultProcessor) handleSuccess(r executor.HTTPResult) {
	ctx := rp.ctx

	// store success in redis
	if err := rp.redisSvc.StoreStatus(ctx, r.MonitorID, r.Status, r.LatencyMs, r.CheckedAt); err != nil {
		rp.logger.Error().
			Err(err).
			Str("monitor_id", r.MonitorID.String()).
			Msg("failed to store success status in redis")
	}

	// Close incident if exists
	cleared, err := rp.redisSvc.ClearIncidentIfExists(ctx, r.MonitorID)
	if err != nil {
		rp.logger.Error().
			Err(err).
			Msg("failed to clear incident from redis")
	}
	if cleared {
		if err := rp.incidentRepo.CloseIncident(ctx, r.MonitorID, time.Now()); err != nil {
			rp.logger.Error().
				Err(err).
				Msg("failed to close incident in DB")
		}
	}

	// clear retry state
	if err := rp.redisSvc.ClearRetry(ctx, r.MonitorID); err != nil {
		rp.logger.Error().
			Err(err).
			Msg("failed to clear retry state from redis")
	}

	// Re-schedule monitor
	rp.monitorSvc.ScheduleMonitor(ctx, r.MonitorID, r.IntervalSec, "result.success_worker")
}