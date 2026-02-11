package config

// type AuthConfig struct {
// 	Secret    string `mapstructure:"secret"`
// 	ExpiryMin int    `mapstructure:"expiry_min"`
// }

// type AppConfig struct {
// 	JobChannelSize    int // size of JobChan
// 	ResultChannelSize int // size of ResultChan
// 	AlertChannelSize  int // size of AlertChan
// }

// type SchedulerConfig struct {
// 	IntervalSec int // at what interval scheduler will tick
// 	BatchSize   int // amount of monitors scheduler will pull from redis
// }

// type ExecutorConfig struct {
// 	WorkerCount  int // count of executor workers
// 	HTTPSemCount int // count of HTTP semaphores
// }

// type AlertConfig struct {
// 	WorkerCount  int
// 	OwnerEmailID string
// 	AccessKey    string
// }

// type ResultProcessor struct {
// 	SuccessWorkersCount int
// 	SuccessChannelSize  int
// 	FailureWorkerCount  int
// 	FailureChannelSize  int
// }

// type RedisConfig struct {
// 	RedisURL           string
// 	DialTimeoutSec     int32
// 	ReadTimeoutSec     int32
// 	WriteTimeoutSec    int32
// 	PoolSize           int32
// 	MinIdleConns       int32
// 	ConnMaxLifeTimeMin int32
// 	ConnMaxIdleTimeSec int32
// }

// type DBConfig struct {
// 	DBURL              string
// 	MaxConns           int32
// 	MinConns           int32
// 	ConnMaxLifeTimeMin int32
// 	ConnMaxIdleTimeSec int32
// 	HealthTimeout      int32
// }

// type Config struct {
// 	DBURL       string `mapstructure:"db_url"`
// 	Port        int    `mapstructure:"port"`
// 	Env         string `mapstructure:"env"`
// 	ServiceName string `mapstructure:"service_name"`
// 	RedisURL    string `mapstructure:"redis_url"`
// 	// Redis       *RedisConfig `mapstructure:"redis"`
// 	Auth *AuthConfig `mapstructure:"auth"`
// 	// RabbitMQ    *RabbitMQConfig `mapstructure:"rabbitmq"`
// }
