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

	defer func() {
		// 1. Acknowledge Job (Remove from inflight)
		if err := rp.redisSvc.AckJob(ctx, r.MonitorID.String()); err != nil {
			rp.logger.Error().Err(err).Str("monitor_id", r.MonitorID.String()).Msg("failed to ack job in redis")
		}
		// 2. Schedule next run
		rp.monitorSvc.ScheduleMonitor(ctx, r.MonitorID, r.IntervalSec, "result.success_worker")
	}()

	// store success in redis
	if err := rp.redisSvc.StoreStatus(ctx, r.MonitorID, r.Status, r.LatencyMs, r.CheckedAt); err != nil {
		rp.logger.Error().
			Err(err).
			Str("monitor_id", r.MonitorID.String()).
			Msg("failed to store success status in redis")
	}
	rp.logger.Info().Str("monitor_id", r.MonitorID.String()).Msg("Success status stored in redis")

	// Fetch incident state from Redis
	incident, err := rp.redisSvc.GetIncident(ctx, r.MonitorID)
	if err != nil {
		// Redis unreliable → skip recovery logic
		rp.logger.Error().
			Err(err).
			Msg("failed to get incident from redis, skipping recovery")
		return
	}
	if incident == nil { // No incident → nothing to recover
		rp.logger.Info().Str("monitor_id", r.MonitorID.String()).Msg("No old incident found in redis")
		return
	}

	rp.logger.Info().Str("monitor_id", r.MonitorID.String()).Msg("old incident found in redis")

	// Close DB incident IF it was ever created
	dbIncident := incident["db_incident"] == "true"

	if dbIncident {
		if err := rp.incidentRepo.CloseIncident(ctx, r.MonitorID, time.Now()); err != nil {
			rp.logger.Error().
				Err(err).
				Msg("failed to close incident in DB, keeping redis incident")
		}
	}

	// Clear Redis incident (safe now)
	if err := rp.redisSvc.ClearIncident(ctx, r.MonitorID); err != nil {
		rp.logger.Error().
			Err(err).
			Msg("failed to clear incident from redis")
	}

	rp.logger.Info().Str("monitor_id", r.MonitorID.String()).Msg("Old incident is cleared from redis")

	// Clear retry state (if exists)
	if err := rp.redisSvc.ClearRetry(ctx, r.MonitorID); err != nil {
		rp.logger.Debug().
			Err(err).
			Msg("failed to clear retry state from redis")
	}
}
