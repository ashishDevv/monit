package db

import (
	"context"
	"fmt"
	"project-k/config"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// type Config struct {
// 	URL             string
// 	MaxConns        int32
// 	MinConns        int32
// 	ConnMaxLifetime time.Duration
// 	ConnMaxIdleTime time.Duration
// 	HealthTimeout   time.Duration
// }

func ConnectToDB(ctx context.Context, DBCfg *config.DBConfig, log *zerolog.Logger) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(DBCfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse db url: %w", err)
	}

	// Pool sizing
	poolCfg.MaxConns = DBCfg.MaxOpenConns
	poolCfg.MinConns = DBCfg.MinIdleConns
	poolCfg.MaxConnLifetime = DBCfg.ConnMaxLifetime
	poolCfg.MaxConnIdleTime = DBCfg.ConnMaxIdleTime

	// Observability hooks
	poolCfg.BeforeConnect = func(ctx context.Context, cfg *pgx.ConnConfig) error {
		log.Debug().Msg("opening new db connection")
		return nil
	}

	poolCfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		log.Debug().Msg("db connection established")
		return nil
	}

	// Create pool (does NOT guarantee connectivity)
	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create db pool: %w", err)
	}

	// Verify connectivity (FAIL FAST)
	healthCtx, cancel := context.WithTimeout(ctx, DBCfg.HealthTimeout)
	defer cancel()

	if err := pool.Ping(healthCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db ping failed: %w", err)
	}

	log.Info().Msg("database connection pool initialized successfully")
	return pool, nil
}


// db.Config{
// 		URL:             cfg.DBURL,
// 		MaxConns:        50,
// 		MinConns:        5,
// 		ConnMaxLifetime: time.Hour,
// 		ConnMaxIdleTime: 30 * time.Minute,
// 		HealthTimeout:   5 * time.Second,
// 	}
