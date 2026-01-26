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

	monitor, err := rp.monitorSvc.LoadMonitor(ctx, r.MonitorID)
	if err != nil { // if err is monitor not found (may be deleted)or any other err , just log and return
		rp.logger.Error().Err(err).Msg("error in loading monitor in result processor worker")
		rp.cleanupRedis(ctx, r.MonitorID)
		return
	}
	if !monitor.Enabled { // monitor is disabled, so dont proceed further
		rp.cleanupRedis(ctx, r.MonitorID)
		return // we not have to do this further
	}

	if err := rp.redisSvc.StoreStatus(ctx, r.MonitorID, r.Status, r.LatencyMs, r.CheckedAt); err != nil {
		// log it
		rp.logger.Err(err).Msg("error in storing status in redis")
	}
	// _ = rp.redisSvc.ClearRetry(ctx, r.MonitorID)

	// Close incident if exists
	incident, err := rp.redisSvc.GetIncident(ctx, r.MonitorID)
	if err != nil {
		// log 
		rp.logger.Err(err).Msg("error in getting incident from redis")
	}
	if incident != nil {
		err := rp.redisSvc.ClearIncident(ctx, r.MonitorID)
		if err != nil {
			// log it
			rp.logger.Err(err).Msg("error in clearing incident")
		}
	}

	// NORMAL re-schedule
	nextRun := time.Now().Add(time.Duration(monitor.IntervalSec)* time.Second)
	if err := rp.redisSvc.Schedule(ctx, r.MonitorID.String(), nextRun); err != nil {
		//log it
		rp.logger.Error().Err(err).Msg("error in scheduling monitor")
	}

}
