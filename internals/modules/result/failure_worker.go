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

	// Case 1 => stop monitoring : No Re-schedule
	if r.Reason == "INVALID_REQUEST" || r.Reason == "DNS_FAILURE" {
		if err := rp.redisSvc.StoreStatus(ctx, r.MonitorID, r.Status, r.LatencyMs, r.CheckedAt); err != nil {
			rp.logger.Error().
				Err(err).
				Msg("failed to store status in redis")
		}
		return
	}

	// Case 2 => retry path : retrying Re-schedule ( 5 sec)
	if r.Retryable {
		retryCount, err := rp.redisSvc.IncrementRetry(ctx, r.MonitorID)
		if err != nil {
			rp.logger.Error().
				Err(err).
				Msg("failed to increment retry count in redis")
			// normal re-schedule,  it can be retry re-schedule as well
			rp.monitorSvc.ScheduleMonitor(ctx, r.MonitorID, r.IntervalSec, "result.failure_worker")
			return
		}

		if retryCount <= 2 {
			rp.monitorSvc.ScheduleMonitor(ctx, r.MonitorID, 5, "result.failure_worker")
			// this method handles everything and reliable
			// It try 3 times, if fails after that , it logs and push in a channel for asyncronous scheduling
			return
		}

		// retry budget exhausted → clear retry state
		if err := rp.redisSvc.ClearRetry(ctx, r.MonitorID); err != nil {
			rp.logger.Error().
				Err(err).
				Msg("failed to clear retry state in redis")
		}
	}

	// Case 3 => failure path : may Alert and but 100% Re-schedule
	failCount, _, err := rp.redisSvc.IncrementIncident(ctx, r.MonitorID)
	if err != nil {
		rp.logger.Error().
			Err(err).
			Msg("failed to increment incident count in redis")
		rp.monitorSvc.ScheduleMonitor(ctx, r.MonitorID, r.IntervalSec, "result.failure_worker")
		return
	}

	// Alerting
	if failCount >= 3 {
		alerted, err := rp.redisSvc.GetIncidentAlerted(ctx, r.MonitorID)
		if err != nil {
			rp.logger.Err(err).Msg("failed to read incident alerted flag")
			// normal re-Schedule
			rp.monitorSvc.ScheduleMonitor(ctx, r.MonitorID, r.IntervalSec, "result.failure_worker")
			return
		}

		if !alerted {
			// Alert
			rp.alertChan <- alert.AlertEvent{MonitorID: r.MonitorID}

			// Mark Alerted
			if err := rp.redisSvc.MarkIncidentAlerted(ctx, r.MonitorID); err != nil {
				rp.logger.Error().
					Err(err).
					Msg("failed to mark incident alerted in redis")
			}

			// presist incident Alert
			if err := rp.incidentRepo.Create(ctx, time.Now(), r); err != nil {
				rp.logger.Error().
					Err(err).
					Msg("failed to create incident in DB")
			}
		}
	}
	
	// Re-Schedule at last
	rp.monitorSvc.ScheduleMonitor(ctx, r.MonitorID, r.IntervalSec, "result.failure_worker")
}
