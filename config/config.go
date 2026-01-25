package config

// import (
// 	"log"
// 	"os"
// 	"strconv"

// 	"github.com/joho/godotenv"
// )

// type AuthConfig struct {
// 	PublicKeyPath string
// }

// type RabbitMQConfig struct {
// 	BrokerLink   string
// 	ExchangeName string
// 	ExchangeType string
// 	QueueName    string
// 	RoutingKey   string
// 	WorkerCount  int
// }

// type Config struct {
// 	DB_URL      string
// 	Port        int
// 	Env         string
// 	ServiceName string
// 	RedisAddr   string
// 	AuthConfig  *AuthConfig
// 	RMQConfig   *RabbitMQConfig
// }

// // LoadConfig loads environment variables and returns a Config struct.
// // It validates required env vars and applies default values.
// func LoadConfig() *Config {
// 	// Load .env if present (optional)
// 	_ = godotenv.Load()

// 	cfg := &Config{
// 		DB_URL:      getRequired("DB_URL"),
// 		Env:         getDefault("ENV", "DEV"),
// 		Port:        getEnvAsInt("PORT", 3000),
// 		ServiceName: getDefault("SERVICE_NAME", "cosmic-service"),
// 		RedisAddr:   getDefault("REDIS_ADDR", "localhost:6379"),
// 		AuthConfig: &AuthConfig{
// 			PublicKeyPath: getRequired("PUBLIC_KEY_PATH"),
// 		},
// 		RMQConfig: &RabbitMQConfig{
// 			BrokerLink:   getRequired("BROKER_LINK"),
// 			ExchangeName: getRequired("EXCHANGE_NAME"),
// 			ExchangeType: getDefault("EXCHANGE_TYPE", "direct"),
// 			QueueName:    getRequired("QUEUE_NAME"),
// 			RoutingKey:   getRequired("ROUTING_KEY"),
// 			WorkerCount:  getEnvAsInt("WORKER_COUNT", 10),
// 		},
// 	}

// 	return cfg
// }

// // getRequired returns the environment variable value or logs fatal if missing.
// func getRequired(key string) string {
// 	val := os.Getenv(key)
// 	if val == "" {
// 		log.Fatalf("Required environment variable %s is missing", key)
// 	}
// 	return val
// }

// // getDefault returns the environment variable value or the provided default.
// func getDefault(key string, defaultVal string) string {
// 	val := os.Getenv(key)
// 	if val == "" {
// 		return defaultVal
// 	}
// 	return val
// }

// // getEnvAsInt returns the environment variable as int or default if missing/invalid
// func getEnvAsInt(key string, defaultVal int) int {
// 	val := os.Getenv(key)
// 	if val == "" {
// 		return defaultVal
// 	}
// 	num, err := strconv.Atoi(val)
// 	if err != nil {
// 		log.Printf("Invalid value of %s='%s', using default %d", key, val, defaultVal)
// 		return defaultVal
// 	}
// 	return num
// }

// // Optional: parse bool
// func getEnvAsBool(key string, defaultVal bool) bool {
// 	val := os.Getenv(key)
// 	if val == "" {
// 		return defaultVal
// 	}
// 	b, err := strconv.ParseBool(val)
// 	if err != nil {
// 		log.Printf("Invalid value of %s='%s', using default %v", key, val, defaultVal)
// 		return defaultVal
// 	}
// 	return b
// }
