package alert

import "log"

type AlertService struct {
	alertChan   chan AlertEvent
	workerCount int
}

func NewAlertService(workerCount int, alertChan chan AlertEvent) *AlertService {
	return &AlertService{
		workerCount: workerCount,
		alertChan:   alertChan,
	}
}

func (s *AlertService) Start() {

	for range s.workerCount {
		go s.handleAlerts()
	}
}

func (s *AlertService) handleAlerts() {

	for alert := range s.alertChan {
		log.Print(alert.MonitorID)
	}
}