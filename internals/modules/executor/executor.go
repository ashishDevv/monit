package executor

import (
	"context"
	"errors"
	"net"
	"net/http"
	"project-k/internals/modules/monitor"
	"project-k/internals/modules/scheduler"
	"sync"
	"time"

	"github.com/google/uuid"
)

type MonitorService interface {
	GetMonitor(context.Context, uuid.UUID) (*monitor.MonitorRecord, error)
}

type Executor struct {
	ctx         context.Context
	workerCount int
	jobChan     chan scheduler.JobPayload
	resultChan  chan HTTPResult
	monitorSvc  MonitorService
	httpSem     chan struct{}
	httpWg      sync.WaitGroup
	httpClient  *http.Client
}

func NewExecutor(
	ctx context.Context,
	workerCount int,
	jobChan chan scheduler.JobPayload,
	resultChan chan HTTPResult,
	monitorSvc MonitorService,
) *Executor {

	return &Executor{
		ctx:         ctx,
		workerCount: workerCount,
		jobChan:     jobChan,
		monitorSvc:  monitorSvc,
		httpSem:     make(chan struct{}, 5000),   // 5k http concurent
		httpClient:  newHttpClient(),
	}
}

func (ew *Executor) StartWorkers() {
	for range ew.workerCount {
		go ew.startWork()
	}
}

func (ew *Executor) startWork() {

	for job := range ew.jobChan {
		// load monitor
		monitor, err := ew.monitorSvc.GetMonitor(ew.ctx, job.MonitorID)
		if err != nil {
			//log it
			return
		}
		if monitor == nil || monitor.Disabled {
			return   // we not have to do this further
		}

		// acquire http semaphore
		ew.httpSem <- struct{}{}
		ew.httpWg.Add(1)

		go func() {
			defer func() {
				<-ew.httpSem
				ew.httpWg.Done()
			}()

			result := ew.executeHTTPCheck(monitor)
			ew.resultChan <- result
		}()
	}
}

func (ew *Executor) executeHTTPCheck(monitor *monitor.MonitorRecord) HTTPResult {

	start := time.Now()

	httpReqCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(httpReqCtx, "GET", monitor.Url, nil)
	if err != nil {
		// this is our server failure, that request cant able to build, because we have validated the client url at the start, when he registered
		// So it deserve re-scheduling again
		return HTTPResult{
			MonitorID: monitor.ID,
			Success:   false,
			Reason:    "INVALID_REQUEST",
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
			- retry count < 3
		-	When not Retry
			- DNS failure  -> its due to bad config -> log it
			- retry count >= 3  // log it VERY IMPORTANT, in future put it in a seperate error channel for debugging
	*/

	defer resp.Body.Close()

	success := resp.StatusCode == monitor.ExpectedStatus && latency <= int64(monitor.LatencyThresholdMS)

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

func (ew *Executor) classifyError(err error) (string, bool) {

	if errors.Is(err, context.DeadlineExceeded) {
		return "TIMEOUT", true
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		// return "DNS_FAILURE", false   // log this error , its our server config mistake
		// dont return , just log it
		return "", true
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
