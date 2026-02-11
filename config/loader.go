package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

func LoadConfig(path string) (*Config, error) {
	v := viper.New()

	// default first
	setDefaults(v)

	// File Config
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	// Env Config
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read File
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	// Validate
	if err := validateConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("env", "development")
	v.SetDefault("service_name", "monitor-service")
	v.SetDefault("port", 8080)

	v.SetDefault("auth.token_ttl", "30m")

	v.SetDefault("scheduler.interval", "10s")
	v.SetDefault("scheduler.batch_size", 10)

	v.SetDefault("redis.dial_timeout", "5s")
	v.SetDefault("redis.read_timeout", "3s")
	v.SetDefault("redis.write_timeout", "3s")
	v.SetDefault("redis.pool_size", 10)
	v.SetDefault("redis.min_idle_conns", 5)
	v.SetDefault("redis.conn_max_lifetime", "2m")
	v.SetDefault("redis.conn_max_idle_time", "30s")

	v.SetDefault("db.max_open_conns", 50)
	v.SetDefault("db.min_idle_conns", 5)
	v.SetDefault("db.conn_max_lifetime", "1h")
	v.SetDefault("db.conn_max_idle_time", "30m")
	v.SetDefault("db.health_timeout", "5s")
}

func validateConfig(cfg *Config) error {

	validate := validator.New()

	if err := validate.Struct(cfg); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			return formatValidationErrors(ve)
		}
		return err
	}
	return nil
}

func formatValidationErrors(ve validator.ValidationErrors) error {
	var sb strings.Builder
	sb.WriteString("config validation failed:\n")

	for _, fe := range ve {
		fmt.Fprintf(&sb, "- field '%s' failed on '%s'\n", fe.Namespace(), fe.Tag())
	}
	return errors.New(sb.String())
}
