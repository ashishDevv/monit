package scheduler

import (
	"context"
	"project-k/config"
	"project-k/pkg/redisstore"
	"time"

	"github.com/rs/zerolog"
)

// Reclaimer is a background process that reclaims monitors that were not processed, from inflight to schedule
type Reclaimer struct {
	// lifecycle
	ctx               context.Context
	interval          time.Duration
	limit             int

	// services
	redisSvc *redisstore.Client

	// misc
	logger *zerolog.Logger
}

func NewReclaimer(
	ctx context.Context,
	reclaimerConfig *config.ReclaimerConfig,
	redisSvc *redisstore.Client,
	logger *zerolog.Logger,
) *Reclaimer {

	return &Reclaimer{
		ctx:               ctx,
		redisSvc:          redisSvc,
		interval:          reclaimerConfig.Interval,  // its should be 5-10s , 5s is ideal
		limit:             reclaimerConfig.Limit, // it should be 100
		logger:            logger,
	}
}

// Run starts the Reclaimer
func (r *Reclaimer) Run() {
	if r.interval <= 0 {
		panic("reclaim loop interval must be > 0")
	}
	r.logger.Info().Msg("Reclaimer started")
	ticker := time.NewTicker(r.interval)
	defer func() {
		ticker.Stop()
		r.logger.Info().Msg("Reclaimer stopped")
	}()

	for {
		select {
		case <-r.ctx.Done():
			return

		case <-ticker.C:
			// pull jobs from redis
			r.logger.Info().Msg("ReclaimLoop Ticked")
			r.doWork()
		}
	}
}

func (r *Reclaimer) doWork() {
	count, err := r.redisSvc.ReclaimMonitors(r.ctx, reclaimMonitorsScript, time.Now(), r.limit)
	if err != nil {
		// transient redis error → log & move on
		r.logger.Error().Err(err).Msg("error to reclaim monitors from redis")
		return
	}
	if count > 0 {
		r.logger.Info().Msgf("Reclaimed %d monitors", count)
	}
}
