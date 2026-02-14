package config

import "time"

type AuthConfig struct {
	Secret string `mapstructure:"secret" validate:"required"`
	// it parsed as minutes
	TokenTTL time.Duration `mapstructure:"token_ttl" validate:"gt=0"`
}

type AppConfig struct {
	JobChannelSize    int `mapstructure:"job_channel_size" validate:"gte=100,lte=5000"`
	ResultChannelSize int `mapstructure:"result_channel_size" validate:"gte=100,lte=5000"`
	AlertChannelSize  int `mapstructure:"alert_channel_size" validate:"gte=100,lte=5000"`
}

type SchedulerConfig struct {
	Interval          time.Duration `mapstructure:"interval" validate:"gte=5"` // should be in sec
	BatchSize         int           `mapstructure:"batch_size" validate:"gt=0"`
	VisibilityTimeout time.Duration `mapstructure:"visibility_timeout" validate:"gt=0"` // should be in sec
}

type ReclaimerConfig struct {
	Interval time.Duration `mapstructure:"interval" validate:"gte=5"` // should be in sec
	Limit    int           `mapstructure:"limit" validate:"gt=0"`
}

type ExecutorConfig struct {
	WorkerCount  int `mapstructure:"worker_count" validate:"gte=5,lte=200"`
	HTTPSemCount int `mapstructure:"http_semaphore_count" validate:"gte=5,lte=6000"`
}

type AlertConfig struct {
	WorkerCount int    `mapstructure:"worker_count" validate:"gte=5"`
	OwnerEmail  string `mapstructure:"owner_email" validate:"required"`
	AccessKey   string `mapstructure:"access_key" validate:"required"`
}

type ResultProcessorConfig struct {
	SuccessWorkerCount int `mapstructure:"success_worker_count" validate:"gte=5"`
	SuccessChannelSize int `mapstructure:"success_channel_size" validate:"gte=5"`
	FailureWorkerCount int `mapstructure:"failure_worker_count" validate:"gte=5"`
	FailureChannelSize int `mapstructure:"failure_channel_size" validate:"gte=5"`
}

type RedisConfig struct {
	URL             string        `mapstructure:"url" validate:"required,url"`
	DialTimeout     time.Duration `mapstructure:"dial_timeout" validate:"gt=0"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout" validate:"gt=0"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout" validate:"gt=0"`
	PoolSize        int           `mapstructure:"pool_size" validate:"gte=1,lte=1000"`
	MinIdleConns    int           `mapstructure:"min_idle_conns" validate:"gt=0"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime" validate:"gt=0"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time" validate:"gt=0"`
}

type DBConfig struct {
	URL             string        `mapstructure:"url" validate:"required,url"`
	MaxOpenConns    int32         `mapstructure:"max_open_conns" validate:"gt=0"`
	MinIdleConns    int32         `mapstructure:"min_idle_conns" validate:"gt=0"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime" validate:"gt=0"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time" validate:"gt=0"`
	HealthTimeout   time.Duration `mapstructure:"health_timeout" validate:"gt=0"`
}

type Config struct {
	Env         string `mapstructure:"env" validate:"required,oneof=development staging production"`
	ServiceName string `mapstructure:"service_name"`
	Port        int    `mapstructure:"port" validate:"gte=1,lte=65535"`

	Auth            AuthConfig            `mapstructure:"auth" validate:"required"`
	App             AppConfig             `mapstructure:"app" validate:"required"`
	Scheduler       SchedulerConfig       `mapstructure:"scheduler" validate:"required"`
	Reclaimer       ReclaimerConfig       `mapstructure:"reclaimer" validate:"required"`
	Executor        ExecutorConfig        `mapstructure:"executor" validate:"required"`
	Alert           AlertConfig           `mapstructure:"alert" validate:"required"`
	ResultProcessor ResultProcessorConfig `mapstructure:"result_processor" validate:"required"`
	Redis           RedisConfig           `mapstructure:"redis" validate:"required"`
	DB              DBConfig              `mapstructure:"db" validate:"required"`
}
