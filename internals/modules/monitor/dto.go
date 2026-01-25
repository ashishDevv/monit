package monitor

type CreateMonitorRequest struct {
	Url                string `json:"url" validate:"required,url"`
	AlertEmail         string `json:"alert_email" validate:"email"`
	IntervalSec        int32  `json:"interval_sec" validate:"required,gte=60"`
	TimeoutSec         int32  `json:"timeout_sec" validate:"required,gte=120"`
	LatencyThresholdMs int32  `json:"latency_threshold_ms" validate:"required,gte=0"`
	ExpectedStatus     int32  `json:"expected_status" validate:"required,gte=100,lte=599"`
}

type GetMonitorResponse struct {
	ID                 string `json:"id"`
	Url                string `json:"url"`
	AlertEmail         string `json:"alert_mail"`
	IntervalSec        int32  `json:"interval_sec"`
	TimeoutSec         int32  `json:"timeout_sec"`
	LatencyThresholdMs int32  `json:"latency_threshold_ms"`
	ExpectedStatus     int32  `json:"expected_status"`
	Enabled            bool   `json:"enabled"`
}

type GetAllMonitorsResponse struct {
	UserID   string               `json:"user_id"`
	Limit    int32                `json:"limit"`
	Offset   int32                `json:"offset"`
	Monitors []GetMonitorResponse `json:"monitors"`
}

type UpdateMonitorStatusRequest struct {
	Enable bool `json:"enable" validate:"required"`
}
