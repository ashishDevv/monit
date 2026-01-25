package result

import (
	"project-k/internals/modules/alert"
	"project-k/internals/modules/executor"
	"time"
)

func (rp *ResultProcessor) failureWorker() {
	for r := range rp.failureChan {
		rp.handleFailure(r)
	}
}

func (rp *ResultProcessor) handleFailure(r executor.HTTPResult) {
	ctx := rp.ctx

	monitor, err := rp.monitorSvc.GetMonitor(ctx, r.MonitorID)
	if err != nil || monitor.Disabled {
		rp.cleanupRedis(ctx, r.MonitorID)
		return
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
			rp.redisSvc.MarkIncidentAlerted(ctx, r.MonitorID)
			rp.incidentRepo.CreateIncident(ctx, r)
		}
	}

	// NORMAL re-schedule (to detect recovery)
	nextRun := time.Now().Add(time.Duration(monitor.IntervalSec) * time.Second)
	if err := rp.redisSvc.Schedule(ctx, r.MonitorID.String(), nextRun); err != nil {
		// log it
	}
}
