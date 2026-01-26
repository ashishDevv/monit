package result

import (
	"context"
	"sync"

	"project-k/internals/modules/alert"
	"project-k/internals/modules/executor"
	"project-k/internals/modules/monitor"
	"project-k/pkg/redisstore"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type MonitorService interface {
	LoadMonitor(context.Context, uuid.UUID) (monitor.Monitor, error)
}

type ResultProcessor struct {
	// lifecycle
	ctx      context.Context
	workerWG sync.WaitGroup

	// services
	redisSvc     *redisstore.Client
	monitorSvc   MonitorService
	incidentRepo *IncidentRepository

	// channels
	resultChan  chan executor.HTTPResult
	successChan chan executor.HTTPResult
	failureChan chan executor.HTTPResult
	alertChan   chan alert.AlertEvent

	// misc
	logger *zerolog.Logger
}

func NewResultProcessor(
	ctx context.Context,
	redisSvc *redisstore.Client,
	resultChan chan executor.HTTPResult,
	incidentRepo *IncidentRepository,
	monitorSvc MonitorService,
	alertChan chan alert.AlertEvent,
	logger *zerolog.Logger,
) *ResultProcessor {
	return &ResultProcessor{
		ctx:          ctx,
		redisSvc:     redisSvc,
		resultChan:   resultChan,
		incidentRepo: incidentRepo,
		monitorSvc:   monitorSvc,
		alertChan:    alertChan,
		successChan:  make(chan executor.HTTPResult, 50),
		failureChan:  make(chan executor.HTTPResult, 5),
		logger:       logger,
	}
}

// StartResultProcessor starts the Result Processor
func (rp *ResultProcessor) StartResultProcessor() {

	// first
	// start success and failure workers
	rp.workerWG.Add(55) // add for all as we have to wait for each worker to complete

	for range 50 {
		go rp.successWorker()
	}

	for range 5 {
		go rp.failureWorker()
	}

	// now start result router
	go rp.router()
}

func (rp *ResultProcessor) router() {
	for r := range rp.resultChan {
		if r.Success {
			rp.successChan <- r
		} else {
			rp.failureChan <- r
		}
	}

	// closing success and failure channel
	close(rp.failureChan)
	close(rp.successChan)
}

// WorkersClosingWait waits for all workers to complete
func (rp *ResultProcessor) WorkersClosingWait() {
	rp.workerWG.Wait()
}

func (rp *ResultProcessor) cleanupRedis(ctx context.Context, monitorID uuid.UUID) {
	_ = rp.redisSvc.ClearIncident(ctx, monitorID)
	// rp.redisSvc.ClearRetry(ctx, monitorID)
}

// func (rp *ResultProcessor) successWorker() {
// 	for r := range rp.successChan {
// 		rp.storeSuccessInRedis(r)
// 	}
// }

// func (rp *ResultProcessor) failureWorker() {
// 	for r := range rp.failureChan {
// 		rp.handleFailure(r)
// 	}
// }

// func (rp *ResultProcessor) storeSuccessInRedis(httpResult executionworker.HTTPResult) {

// 	// - Update status in Redis
// 	// - Clear retry
// 	// - Clear incident (if any)
// 	// - next_run = now + interval
// 	// - ZADD monitor:schedule next_run
// }

// func (rp *ResultProcessor) handleFailure(httpResult executionworker.HTTPResult) {
// 	if httpResult.Retryable {
// - increment retry count
// - next_run = now + retry_delay
// - Redis ZADD(next_run)
// 	}

// now its a real fault for client, so create a inicident
// - increment incident
// - maybe alert
// - next_run = now + interval
// - Redis ZADD(next_run)
// }
