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
	"project-k/pkg/httpclient"
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
	Reclaimer      *scheduler.Reclaimer
	Scheduler      *scheduler.Scheduler
	Executor       *executor.Executor
	ResultPro      *result.ResultProcessor
	AlertSvc       *alert.AlertService
	JobChan        chan scheduler.JobPayload
	ResultChan     chan executor.HTTPResult
	AlertChan      chan alert.AlertEvent
}

func NewContainer(ctx context.Context, db *pgxpool.Pool, cfg *config.Config, logger *zerolog.Logger) (*Container, error) {

	redisClient, err := redisstore.New(&cfg.Redis)
	if err != nil {
		return nil, err
	}
	tokenSvc := security.NewTokenService(&cfg.Auth)

	jobChan := make(chan scheduler.JobPayload, cfg.App.JobChannelSize)      // specify channel size in config
	resultChan := make(chan executor.HTTPResult, cfg.App.ResultChannelSize) // specify channel size in config
	alertChan := make(chan alert.AlertEvent, cfg.App.AlertChannelSize)      // specify channel size in config

	validator := validator.New()

	monitorRepo := monitor.NewRepository(db, logger)
	incidentRepo := result.NewMonitorIncidentRepo(db, logger)
	userRepo := user.NewRepository(db, logger)

	httpClient := httpclient.NewHttpClient()

	userService := user.NewService(userRepo, tokenSvc)
	monitorSvc := monitor.NewService(monitorRepo, redisClient, userService, logger)

	reclaimer := scheduler.NewReclaimer(ctx, &cfg.Reclaimer, redisClient, logger)
	sch := scheduler.NewScheduler(ctx, &cfg.Scheduler, jobChan, redisClient, logger)
	exec := executor.NewExecutor(ctx, &cfg.Executor, jobChan, resultChan, monitorSvc, httpClient, logger)
	resultPro := result.NewResultProcessor(ctx, &cfg.ResultProcessor, redisClient, resultChan, incidentRepo, monitorSvc, alertChan, logger)
	alertSvc := alert.NewAlertService(&cfg.Alert, alertChan, logger)

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
		Reclaimer:      reclaimer,
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
		return err
	}

	// Close DB pool , do not close here , it will be closed in defer in main.go
	//if c.DB != nil {
	//	c.DB.Close()
	//}
	return nil
}
