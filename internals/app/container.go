package app

import (
	"context"
	"project-k/config"
	middle "project-k/internals/middleware"
	"project-k/internals/modules/alert"
	"project-k/internals/modules/executor"
	"project-k/internals/modules/monitor"
	"project-k/internals/modules/result"
	"project-k/internals/modules/scheduler"
	"project-k/internals/modules/user"
	"project-k/internals/security"
	"project-k/pkg/redisstore"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type Container struct {
	DB             *pgxpool.Pool
	RedisClient    *redisstore.Client
	Logger         *zerolog.Logger
	userSvc        *user.Service
	userHandler    *user.Handler
	monitorHandler *monitor.Handler
	authMW         *middle.AuthMiddleware
	scheduler      *scheduler.Scheduler
	executor       *executor.Executor
	resultPro      *result.ResultProcessor
	alertSvc       *alert.AlertService
}

func NewContainer(ctx context.Context, db *pgxpool.Pool, cfg *config.Config, logger *zerolog.Logger) (*Container, error) {

	redisClient, err := redisstore.New(cfg.RedisCfg)
	if err != nil {
		return nil, err
	}

	jobChan := make(chan scheduler.JobPayload, 1000)
	resultChan := make(chan executor.HTTPResult, 1000)
	alertChan := make(chan alert.AlertEvent, 500)

	validator := validator.New()

	monitorRepo := monitor.NewRepository()
	incidentRepo := result.NewIncidentRepository()
	userRepo := user.NewRepository(db)

	userService := user.NewService(userRepo)
	monitorSvc := monitor.NewService(monitorRepo)

	sch := scheduler.NewScheduler(ctx, jobChan, redisClient)
	exec := executor.NewExecutor(ctx, 100, jobChan, resultChan, monitorSvc)
	resultPro := result.NewResultProcessor(ctx, redisClient, resultChan, incidentRepo, monitorSvc, alertChan)
	alertSvc := alert.NewAlertService(50, alertChan)

	tokenSvc, err := security.NewTokenService(cfg.Auth)
	if err != nil {
		return nil, err
	}

	monitorHandler := monitor.NewHandler(monitorSvc, validator)
	userHandler := user.NewHandler(userService, validator)

	authMW := middle.NewAuthMiddleware(tokenSvc)

	return &Container{
		DB:             db,
		Logger:         logger,
		userSvc:        userService,
		userHandler:    userHandler,
		authMW:         authMW,
		monitorHandler: monitorHandler,
		scheduler:      sch,
		executor:       exec,
		resultPro:      resultPro,
		alertSvc:       alertSvc,
	}, nil
}

func (c *Container) Shutdown(ctx context.Context) error {
	// 3. Close DB pool
	if c.DB != nil {
		c.DB.Close()
	}
	return nil
}
