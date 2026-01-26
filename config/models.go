package config

type AuthConfig struct {
	Secret    string `mapstructure:"secret"`
	ExpiryMin int    `mapstructure:"expiry_min"`
}

// type RabbitMQConfig struct {
// 	BrokerLink   string `mapstructure:"broker_link"`
// 	ExchangeName string `mapstructure:"exchange_name"`
// 	ExchangeType string `mapstructure:"exchange_type"`
// 	QueueName    string `mapstructure:"queue_name"`
// 	RoutingKey   string `mapstructure:"routing_key"`
// 	WorkerCount  int    `mapstructure:"worker_count"`
// }

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type Config struct {
	DBURL       string       `mapstructure:"db_url"`
	Port        int          `mapstructure:"port"`
	Env         string       `mapstructure:"env"`
	ServiceName string       `mapstructure:"service_name"`
	Redis       *RedisConfig `mapstructure:"redis"`
	Auth        *AuthConfig  `mapstructure:"auth"`
	// RabbitMQ    *RabbitMQConfig `mapstructure:"rabbitmq"`
}
