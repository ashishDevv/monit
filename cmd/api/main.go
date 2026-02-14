package main

import (
	"context"
	"log"
	"os/signal"
	"project-k/config"
	"project-k/internals/app"
	"project-k/internals/server"
	"project-k/pkg/db"
	"project-k/pkg/logger"
	"syscall"
	"time"
)

func main() {
	// Load envs
	cfg, err := config.LoadConfig("env.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	// Get Context with signals attached -> when ever a signal occurs , then `Done` channel of ctx will get closed
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Initialize Base/global logger
	log := logger.Init(cfg)
	log.Info().Msg("logger initialized")

	// Initialize DB Pool
	dbPool, err := db.ConnectToDB(ctx, &cfg.DB, log)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize db pool")
	}
	log.Info().Msg("database pool initialized")
	defer dbPool.Close()

	// Inject Dependencies
	container, err := app.NewContainer(ctx, dbPool, cfg, log)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize dependencies")
	}
	log.Info().Msg("dependencies initialized")

	// start our heroes
	// start reclaimer
	go container.Reclaimer.Run()
	// start scheduler
	go container.Scheduler.Run()
	// start executor
	container.Executor.StartWorkers()
	// start result processor
	container.ResultPro.StartResultProcessor()
	// start alert service
	container.AlertSvc.Run()

	// all heroes are initialized
	log.Info().Msg("all heroes initialized")

	// Register Routes
	router := app.RegisterRoutes(container)
	log.Info().Msg("routes registered")

	// Start HTTP Server -> Runs in a seperate goroutines in background and receive requests
	srv := server.New(":8080", router, log)
	srv.Start()

	// main goroutine is for gracefull shutdown

	<-ctx.Done() // WAIT FOR SIGNAL (waiting for closure of Done channel, when it closes, it run forward from here)
	log.Info().Msg("shutdown signal received")

	// 1. Stop HTTP server (stop accepting requests)
	if err := srv.Shutdown(context.Background()); err != nil {
		log.Error().Err(err).Msg("server shutdown failed")
	}

	// 2. Shutdown background workers & infra
	_, cancel := context.WithTimeout(context.Background(), 30*time.Second) // this is new context, acts as buffer time to close all resources
	defer cancel()

	if err := container.Shutdown(); err != nil {
		log.Error().Err(err).Msg("dependecies shutdown failed")
	}

	// Shutdown done
	log.Info().Msg("graceful shutdown complete")
}
