package config

type AuthConfig struct {
	PublicKeyPath string `mapstructure:"public_key_path"`
}

type RabbitMQConfig struct {
	BrokerLink   string `mapstructure:"broker_link"`
	ExchangeName string `mapstructure:"exchange_name"`
	ExchangeType string `mapstructure:"exchange_type"`
	QueueName    string `mapstructure:"queue_name"`
	RoutingKey   string `mapstructure:"routing_key"`
	WorkerCount  int    `mapstructure:"worker_count"`
}

type Config struct {
	DBURL       string         `mapstructure:"db_url"`
	Port        int            `mapstructure:"port"`
	Env         string         `mapstructure:"env"`
	ServiceName string         `mapstructure:"service_name"`
	RedisAddr   string         `mapstructure:"redis_addr"`
	Auth        *AuthConfig     `mapstructure:"auth"`
	RabbitMQ    *RabbitMQConfig `mapstructure:"rabbitmq"`
}
