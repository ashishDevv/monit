package result

import (
	"time"

	"project-k/internals/modules/executor"
)

func (rp *ResultProcessor) successWorker() {
	for r := range rp.successChan {
		rp.handleSuccess(r)
	}
}

func (rp *ResultProcessor) handleSuccess(r executor.HTTPResult) {
	ctx := rp.ctx

	monitor, err := rp.monitorSvc.GetMonitor(ctx, r.MonitorID)
	if err != nil || monitor.Disabled {
		rp.cleanupRedis(ctx, r.MonitorID)
		return
	}

	if err := rp.redisSvc.StoreStatus(ctx, r.MonitorID, r.Status, r.LatencyMs, r.CheckedAt); err != nil {
		// log it
	}
	// _ = rp.redisSvc.ClearRetry(ctx, r.MonitorID)

	// Close incident if exists
	incident, err := rp.redisSvc.GetIncident(ctx, r.MonitorID)
	if err != nil {
		// log it
	}
	if incident != nil {
		err := rp.redisSvc.ClearIncident(ctx, r.MonitorID)
		if err != nil {
			// log it
		}
	}

	// NORMAL re-schedule
	nextRun := time.Now().Add(monitor.IntervalSec * time.Second)
	if err := rp.redisSvc.Schedule(ctx, r.MonitorID.String(), nextRun); err != nil {
		//log it
	}

}
