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
	Scheduler      *scheduler.Scheduler
	Executor       *executor.Executor
	ResultPro      *result.ResultProcessor
	AlertSvc       *alert.AlertService
	JobChan        chan scheduler.JobPayload
	ResultChan     chan executor.HTTPResult
	AlertChan      chan alert.AlertEvent
}

func NewContainer(ctx context.Context, db *pgxpool.Pool, cfg *config.Config, logger *zerolog.Logger) (*Container, error) {

	redisClient, err := redisstore.New(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		return nil, err
	}
	tokenSvc := security.NewTokenService(cfg.Auth)

	jobChan := make(chan scheduler.JobPayload, 1000)
	resultChan := make(chan executor.HTTPResult, 1000)
	alertChan := make(chan alert.AlertEvent, 500)

	validator := validator.New()

	monitorRepo := monitor.NewRepository(db, logger)
	incidentRepo := result.NewMonitorIncidentRepo(db)
	userRepo := user.NewRepository(db, logger)

	userService := user.NewService(userRepo, tokenSvc)
	monitorSvc := monitor.NewService(monitorRepo, redisClient, userService, logger)

	sch := scheduler.NewScheduler(ctx, jobChan, redisClient, logger)
	exec := executor.NewExecutor(ctx, 100, jobChan, resultChan, monitorSvc, logger)
	resultPro := result.NewResultProcessor(ctx, redisClient, resultChan, incidentRepo, monitorSvc, alertChan, logger)
	alertSvc := alert.NewAlertService(50, alertChan, logger)

	monitorHandler := monitor.NewHandler(monitorSvc, validator, logger)
	userHandler := user.NewHandler(userService, validator, logger)

	authMW := middle.NewAuthMiddleware(tokenSvc)

	return &Container{
		DB:             db,
		Logger:         logger,
		RedisClient:    redisClient,
		userSvc:        userService,
		userHandler:    userHandler,
		authMW:         authMW,
		monitorHandler: monitorHandler,
		Scheduler:      sch,
		Executor:       exec,
		ResultPro:      resultPro,
		AlertSvc:       alertSvc,
		JobChan:        jobChan,
		ResultChan:     resultChan,
		AlertChan:      alertChan,
	}, nil
}

func (c *Container) Shutdown() error {

	close(c.JobChan)

	c.Executor.Stop()

	close(c.ResultChan)

	c.ResultPro.WorkersClosingWait()

	close(c.AlertChan)

	c.AlertSvc.WorkerClosingWait()

	// close redis
	err := c.RedisClient.Close()
	if err != nil {
		return  err
	}

	// Close DB pool
	if c.DB != nil {
		c.DB.Close()
	}
	return nil
}
