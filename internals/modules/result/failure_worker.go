package result

import (
	"project-k/internals/modules/alert"
	"project-k/internals/modules/executor"
	"time"
)

func (rp *ResultProcessor) failureWorker() {
	defer rp.workerWG.Done()

	for r := range rp.failureChan {
		rp.handleFailure(r)
	}
}

func (rp *ResultProcessor) handleFailure(r executor.HTTPResult) {
	ctx := rp.ctx
	reschedule := true
	
	rp.logger.Info().Str("monitor_id", r.MonitorID.String()).Msg("Failure occured in monitor check")

	defer func() {
		if reschedule {
			rp.monitorSvc.ScheduleMonitor(ctx, r.MonitorID, r.IntervalSec, "result.failure_worker")
		}
	}()

	// Case 1 => stop monitoring : No Re-schedule
	if r.Reason == "INVALID_REQUEST" || r.Reason == "DNS_FAILURE" {   // these should have failure type, not String
		rp.logger.Info().Str("monitor_id", r.MonitorID.String()).Msg("Failure is Terminal, notify user")
		if err := rp.redisSvc.StoreStatus(ctx, r.MonitorID, r.Status, r.LatencyMs, r.CheckedAt); err != nil {
			rp.logger.Error().Err(err).Msg("failed to store status in redis")
		}
		reschedule = false
		return
	}

	// Case 2 => retry path : retrying Re-schedule ( 5 sec)
	if r.Retryable {
		rp.logger.Info().Str("monitor_id", r.MonitorID.String()).Msg("Retryable Failure")
		retryCount, err := rp.redisSvc.IncrementRetry(ctx, r.MonitorID)
		if err != nil {
			rp.logger.Error().Err(err).Msg("failed to increment retry count in redis")
			// normal re-schedule,  it can be retry re-schedule as well
			reschedule = false
			rp.monitorSvc.ScheduleMonitor(ctx, r.MonitorID, r.IntervalSec, "result.failure_worker")
			return
		}

		if retryCount <= 2 {    // specify this in config
			reschedule = false
			rp.monitorSvc.ScheduleMonitor(ctx, r.MonitorID, 5, "result.failure_worker")
			// this method handles everything and reliable
			// It try 3 times, if fails after that , it logs and push in a channel for asyncronous scheduling
			return
		}

		// retry budget exhausted → clear retry state
		if err := rp.redisSvc.ClearRetry(ctx, r.MonitorID); err != nil {
			rp.logger.Error().Err(err).Msg("failed to clear retry state in redis")
		}
	}

	// Case 3 => failure path : may Alert and but 100% Re-schedule
	failCount, _, err := rp.redisSvc.IncrementIncident(ctx, r.MonitorID)
	if err != nil {
		rp.logger.Error().Err(err).Msg("failed to increment incident count in redis")
		// rp.monitorSvc.ScheduleMonitor(ctx, r.MonitorID, r.IntervalSec, "result.failure_worker")
		return
	}
	rp.logger.Info().Str("monitor_id", r.MonitorID.String()).Msg("Created a incident in redis")

	if failCount < 3 { // specify this in config
		rp.logger.Info().Str("monitor_id", r.MonitorID.String()).Int64("fail_count", failCount).Msg("Fail count is less than threshold")
		return
	}
	
	rp.logger.Info().Str("monitor_id", r.MonitorID.String()).Int64("fail_count", failCount).Msg("Fail count is greater than threshold, will alert and create DB incident")

	// Atomic alert decision
	shouldAlert, err := rp.redisSvc.MarkIncidentAlertedIfNotSet(ctx, r.MonitorID)
	if err != nil || !shouldAlert {
		rp.logger.Info().Str("monitor_id", r.MonitorID.String()).Msg("Monitor already alerted")
		return
	}
	
	rp.logger.Info().Str("monitor_id", r.MonitorID.String()).Msg("Now we alert Monitor")

	if err := rp.redisSvc.MarkDBIncidentCreated(ctx, r.MonitorID); err != nil {
		rp.logger.Error().Err(err).Msg("failed to mark db_incident")
	}
	
	if err := rp.incidentRepo.Create(ctx, time.Now(), r); err != nil {
		rp.logger.Error().Err(err).Msg("failed to create incident in DB")
	}
	rp.logger.Info().Str("monitor_id", r.MonitorID.String()).Msg("Created incident in DB")
	
	rp.alertChan <- alert.AlertEvent{MonitorID: r.MonitorID}
	rp.logger.Info().Str("monitor_id", r.MonitorID.String()).Msg("Send Alert to alert channel")
}
