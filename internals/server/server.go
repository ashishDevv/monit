package server

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

type Server struct {
	httpServer *http.Server
	logger     *zerolog.Logger
}

func New(addr string, router chi.Router, logger *zerolog.Logger) *Server{
	return &Server{
		httpServer: &http.Server{
			Addr: addr,
			Handler: router,
			// Add other configrations
		},
		logger: logger,
	}
}

func (s *Server) Start() {
	// start http server in a seperate goroutine
	go func() {
		s.logger.Info().Msgf("HTTP server listening on %s", s.httpServer.Addr)

		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatal().Err(err).Msg("HTTP server crashed")
		}
	}()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info().Msg("shutting down HTTP server")

	shutdownCtx, cancle := context.WithTimeout(ctx, 10 * time.Second)
	defer cancle()

	return s.httpServer.Shutdown(shutdownCtx)
}
