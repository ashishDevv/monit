package config

// import (
// 	"log"
// 	"strings"

// 	"github.com/spf13/viper"
// )

// func LoadConfig(path string) *Config {
// 	v := viper.New()
// 	v.SetConfigFile(path)   // path to your YAML file
// 	v.SetConfigType("yaml") // optional if file has .yaml extension

// 	// Set defaults
// 	v.SetDefault("port", 3000)
// 	v.SetDefault("env", "development")
// 	// v.SetDefault("rabbitmq.worker_count", 10)
// 	v.SetDefault("service_name", "cosmic-user-service")

// 	// Env overrides
// 	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
// 	v.AutomaticEnv()

// 	if err := v.ReadInConfig(); err != nil {
// 		log.Fatalf("Error reading config file: %v", err)
// 	}

// 	var cfg Config
// 	if err := v.Unmarshal(&cfg); err != nil {
// 		log.Fatalf("Error unmarshaling config: %v", err)
// 	}

// 	// Validation
// 	if cfg.DBURL == "" {
// 		log.Fatal("DB_URL is required")
// 	}

// 	if cfg.Auth == nil || cfg.Auth.Secret == "" || cfg.Auth.ExpiryMin == 0 {
// 		log.Fatal("Auth config required is required")
// 	}
	
// 	if cfg.RedisURL == "" {
// 		log.Fatal("Redis configrations are required")
// 	}

// 	return &cfg
// }
