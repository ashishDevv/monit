package alert

import (
	"log"
	"sync"

	"github.com/rs/zerolog"
)

type AlertService struct {
	// lifecycle
	workerCount int
	workerWG    sync.WaitGroup

	// channels
	alertChan   chan AlertEvent

	// misc
	logger      *zerolog.Logger
}

func NewAlertService(workerCount int, alertChan chan AlertEvent, logger *zerolog.Logger) *AlertService {
	return &AlertService{
		workerCount: workerCount,   // specify in config
		alertChan:   alertChan,
		logger:      logger,
	}
}

// Starts starts the Alert Service
func (s *AlertService) Start() {
	
	s.workerWG.Add(s.workerCount)

	for range s.workerCount {
		go s.handleAlerts()
	}
	s.logger.Info().Msg("Alert workers started")
}

func (s *AlertService) handleAlerts() {
	defer s.workerWG.Done()

	for alert := range s.alertChan {
		s.logger.Info().Msg("Alert Recieved")
		log.Print(alert.MonitorID)
	}
}

// WorkerClosingWait waits for alert workers to complete
func (s *AlertService) WorkerClosingWait() {
	s.workerWG.Wait()
}
