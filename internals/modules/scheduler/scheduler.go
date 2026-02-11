package scheduler

import (
	"context"
	"project-k/config"
	"time"

	"project-k/pkg/redisstore"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

type Scheduler struct {
	// lifecycle
	ctx       context.Context
	interval  time.Duration
	ticker    *time.Ticker
	batchSize int

	// channels
	jobChan chan JobPayload

	// services
	redisSvc *redisstore.Client

	// misc
	logger *zerolog.Logger
}

func NewScheduler(
	ctx context.Context,
	schedulerConfig *config.SchedulerConfig,
	jobChan chan JobPayload,
	redisSvc *redisstore.Client,
	logger *zerolog.Logger,
) *Scheduler {

	return &Scheduler{
		ctx:       ctx,
		jobChan:   jobChan,
		redisSvc:  redisSvc,
		interval:  schedulerConfig.Interval, // It will be in sec
		batchSize: schedulerConfig.BatchSize,
		logger:    logger,
	}
}

// StartScheduler starts the Scheduler
func (sc *Scheduler) StartScheduler() {
	sc.logger.Info().Msg("Scheduler started")
	ticker := time.NewTicker(sc.interval)
	sc.ticker = ticker

	go func() {
		for {
			select {
			case <-sc.ctx.Done():
				sc.ticker.Stop()
				sc.logger.Info().Msg("Scheduler stopped")
				return

			case <-ticker.C:
				// pull jobs from redis
				sc.logger.Info().Msg("Scheduler Ticked")
				sc.doWork() // 400 ms
			}
		}
	}()
}

func (sc *Scheduler) doWork() {
	now := time.Now().Unix()

	items, err := sc.redisSvc.PopDue(sc.ctx, sc.batchSize) // 5000,  pr => 95%,  100 non valid
	if err != nil {
		// transient redis error → log & move on
		sc.logger.Error().Err(err).Msg("error to pop scheduled monitors from redis")
		return
	}

	if len(items) == 0 {
		return
	}

	sc.logger.Info().Msgf("Scheduler popped %v items", len(items))

	reinsert := make([]redis.Z, 0, 10)

	for i, item := range items {
		score := int64(item.Score)

		if score > now {
			// Not due yet → put back and STOP
			reinsert = append(reinsert, redis.Z{
				Score:  item.Score,
				Member: item.Member.(string),
			})

			// _ = sc.redisSvc.Schedule(      This code is just for reference
			// 	sc.ctx,
			// 	item.Member.(string),
			// 	time.Unix(score, 0),
			// )

			// reinsert ALL remaining popped items
			for _, future := range items[i+1:] {
				reinsert = append(reinsert, redis.Z{
					Score:  future.Score,
					Member: future.Member.(string),
				})
			}
			// put all reinserts in one call  -> this is more optimsed
			if err := sc.redisSvc.ScheduleBatch(sc.ctx, reinsert); err != nil {
				// log it
				sc.logger.Error().Err(err).Msg("error to schedule in batch")
			}
			sc.logger.Info().Msgf("Scheduler reinserted %v items", len(reinsert))
			break
		}

		id, err := uuid.Parse(item.Member.(string))
		if err != nil {
			// corrupted data, skip
			sc.logger.Error().Err(err).Msg("error in parsing the schedule monitor Id to uuid")
			continue
		}

		select {
		case sc.jobChan <- JobPayload{MonitorID: id}: // has space
			// success
		case <-sc.ctx.Done():
			return
		default:
			// jobChan full → backpressure protection
			// reinsert job so it’s not lost
			sc.logger.Info().Msg("Applying backpressure and re-scheduling")
			backoff := time.Unix(score, 0).Add(2 * time.Second)
			if err := sc.redisSvc.Schedule(sc.ctx, item.Member.(string), backoff); err != nil {
				sc.logger.Error().Err(err).Msg("error in scheduling monitor")
				// enqueue in queue
			}
		}
	}
}
