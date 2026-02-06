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

	monitor, err := rp.monitorSvc.LoadMonitor(ctx, r.MonitorID)
	if err != nil { // if err is monitor not found (may be deleted)or any other err , just log and return
		//log it
		rp.cleanupRedis(ctx, r.MonitorID)
		return
	}
	if !monitor.Enabled { // monitor is disabled, so dont proceed further
		rp.cleanupRedis(ctx, r.MonitorID)
		return // we not have to do this further
	}

	// --- RETRY PATH ---
	if r.Retryable {
		retryCount, _ := rp.redisSvc.IncrementRetry(ctx, r.MonitorID)

		if retryCount <= 2 {
			nextRun := time.Now().Add(5 * time.Second)
			if err := rp.redisSvc.Schedule(ctx, r.MonitorID.String(), nextRun); err != nil {
				// log it
			}
			return
		}
	}

	// --- REAL FAILURE PATH ---
	failCount, _, err := rp.redisSvc.IncrementIncident(ctx, r.MonitorID)
	if err != nil {
		// log it
		return
	}

	if failCount >= 3 {
		alerted, _ := rp.redisSvc.GetIncidentAlerted(ctx, r.MonitorID)
		if !alerted {
			rp.alertChan <- alert.AlertEvent{MonitorID: r.MonitorID}
			_ = rp.redisSvc.MarkIncidentAlerted(ctx, r.MonitorID)
			if err := rp.incidentRepo.Create(ctx, time.Now(), r); err != nil {
				
			}
		}
	}

	// NORMAL re-schedule (to detect recovery)
	nextRun := time.Now().Add(time.Duration(monitor.IntervalSec) * time.Second)
	if err := rp.redisSvc.Schedule(ctx, r.MonitorID.String(), nextRun); err != nil {
		// log it
		rp.logger.Error().Err(err).Msg("error in scheduling monitor")
	}
}
