package executor

import (
	"context"
	"errors"
	"net"
	"net/http"
	"project-k/config"
	"project-k/internals/modules/monitor"
	"project-k/internals/modules/scheduler"
	"project-k/pkg/apperror"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type MonitorService interface {
	LoadMonitor(context.Context, uuid.UUID) (monitor.Monitor, error)
	ScheduleMonitor(context.Context, uuid.UUID, int32, string)
}

type Executor struct {
	// lifecycle
	ctx         context.Context
	workerCount int // specify it in config
	workerWg    sync.WaitGroup

	// channels
	jobChan    chan scheduler.JobPayload
	resultChan chan HTTPResult

	// services
	monitorSvc MonitorService

	// http goroutines config
	httpSem    chan struct{}
	httpWg     sync.WaitGroup
	httpClient *http.Client

	// misc
	logger *zerolog.Logger
}

func NewExecutor(
	ctx context.Context,
	executorConfig *config.ExecutorConfig,
	jobChan chan scheduler.JobPayload,
	resultChan chan HTTPResult,
	monitorSvc MonitorService,
	httpClient *http.Client,
	logger *zerolog.Logger,
) *Executor {

	return &Executor{
		ctx:         ctx,
		workerCount: executorConfig.WorkerCount,
		jobChan:     jobChan,
		resultChan:  resultChan,
		monitorSvc:  monitorSvc,
		httpSem:     make(chan struct{}, executorConfig.HTTPSemCount), // 5k http concurrent , specify it in config
		httpClient:  httpClient,
		logger:      logger,
	}
}

// StartWorkers starts the Executor workers
func (ew *Executor) StartWorkers() {
	for range ew.workerCount { // 100  , -> 5mb
		ew.workerWg.Add(1)
		go ew.startWork()
	}

	ew.logger.Info().Int("workers", ew.workerCount).Msg("Executor workers started")
}

func (ew *Executor) startWork() {
	defer ew.workerWg.Done()

	for job := range ew.jobChan {
		ew.logger.Info().Str("monitor_id", job.MonitorID.String()).Msg("New job in Job Channel")
		// load monitor
		monitor, err := ew.monitorSvc.LoadMonitor(ew.ctx, job.MonitorID)
		if err != nil { // if err is monitor not found (may be deleted)or any other err , just log and return
			// if err == not found -> simply return as monitor is deleted
			if apperror.IsKind(err, apperror.NotFound) {
				continue
			}
			// if err anything else -> it critical
			// IT SHOULD BE RE-SCHEDULE
			// log it
			ew.monitorSvc.ScheduleMonitor(ew.ctx, job.MonitorID, 5, "executor.start_work")
			ew.logger.Error().Err(err).Str("monitor_id", job.MonitorID.String()).Msg("error in loading monitor in executor")
			continue
		}
		if !monitor.Enabled { // monitor is disabled, so dont proceed further
			continue // we not have to do this further
		}

		ew.logger.Info().Msg("Monitor Loaded")

		// acquire http semaphore
		ew.httpSem <- struct{}{}
		ew.httpWg.Add(1)

		go func() {
			defer func() {
				<-ew.httpSem
				ew.httpWg.Done()
			}()

			result := ew.executeHTTPCheck(monitor)
			ew.logger.Info().Object("http_result", result).Msg("Got HTTPResult and pushed to result channel")
			ew.resultChan <- result
		}()
	}
}

// Stop waits for all workers and http gourotines to complete
func (ew *Executor) Stop() {

	ew.workerWg.Wait()
	
	ew.httpWg.Wait()
}

func (ew *Executor) executeHTTPCheck(monitor monitor.Monitor) HTTPResult {

	start := time.Now()

	httpReqCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(httpReqCtx, "GET", monitor.Url, nil)
	if err != nil {
		// this is request building error -> means url is wrong,
		// so its clients problem, we should handle it seperately in result processor,
		// and add this in DB/redis (so client get to know about this), and DO NOT RE-SCHEDULE IT
		// log it as well as that we can see it
		ew.logger.Error().
			Err(err).
			Str("monitor_id", monitor.ID.String()).
			Str("monitor_url", monitor.Url).
			Msg("error in building request")

		return HTTPResult{
			MonitorID: monitor.ID,
			Success:   false,
			Reason:    "INVALID_REQUEST", // check with this in result processor
			Retryable: false,
			CheckedAt: time.Now(),
		}
	}
	resp, err := ew.httpClient.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		// this can be DNS err, network err, TLS err and context timeout(because of hanging request)
		reason, isRetryable := ew.classifyError(err)
		return HTTPResult{
			MonitorID: monitor.ID,
			Success:   false,
			Reason:    reason,
			Retryable: isRetryable,
			CheckedAt: time.Now(),
		}
	}

	/*
		when we retry and when not
		-	Retry when its
			- request building err -> its our err, log it as well as it will remain after retry as well
			- network err/timeout -> maybe network is slow
			- TLS timeout err
		-	When not Retry
			- DNS failure  -> its due to bad config -> log it
			- retry count >= 3  // log it VERY IMPORTANT, in future put it in a seperate error channel for debugging
	*/

	defer resp.Body.Close()

	success := resp.StatusCode == int(monitor.ExpectedStatus) && latency <= int64(monitor.LatencyThresholdMs)

	return HTTPResult{
		MonitorID: monitor.ID,
		Status:    resp.StatusCode,
		LatencyMs: latency,
		Success:   success,
		Reason:    "",
		Retryable: false,
		CheckedAt: time.Now(),
	}
}

func (_ *Executor) classifyError(err error) (string, bool) {

	if errors.Is(err, context.DeadlineExceeded) {
		return "TIMEOUT", true
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return "DNS_FAILURE", false
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return "NETWORK_TIMEOUT", true
		}
		return "NETWORK_ERROR", true
	}

	return "UNKNOWN_ERROR", true
}
