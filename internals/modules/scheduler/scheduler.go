package scheduler

import (
	"context"
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
	jobChan   chan JobPayload

	// services
	redisSvc  *redisstore.Client
	
	// misc
	logger    *zerolog.Logger
	
}

func NewScheduler(
	ctx context.Context,
	jobChan chan JobPayload,
	redisSvc *redisstore.Client,
	logger *zerolog.Logger,
) *Scheduler {

	return &Scheduler{
		ctx:       ctx,
		jobChan:   jobChan,
		redisSvc:  redisSvc,
		interval:  2 * time.Second,
		batchSize: 500,
		logger:    logger,
	}
}

// StartScheduler starts the Scheduler
func (sc *Scheduler) StartScheduler() {
	ticker := time.NewTicker(sc.interval)
	sc.ticker = ticker

	go func() {
		for {
			select {
			case <-sc.ctx.Done():
				sc.ticker.Stop()
				sc.logger.Info().Msg("scheduler stopped")
				return

			case <-ticker.C:
				// pull jobs from redis
				sc.doWork()
			}
		}
	}()
}

func (sc *Scheduler) doWork() {
	now := time.Now().Unix()

	items, err := sc.redisSvc.PopDue(sc.ctx, sc.batchSize)
	if err != nil {
		// transient redis error → log & move on
		sc.logger.Error().Err(err).Msg("error to pop scheduled monitors from redis")
		return
	}

	if len(items) == 0 {
		time.Sleep(200 * time.Millisecond)
		return
	}

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
			break
		}

		id, err := uuid.Parse(item.Member.(string))
		if err != nil {
			// corrupted data, skip
			sc.logger.Error().Err(err).Msg("error in parsing the schedule monitor Id to uuid")
			continue
		}

		select {
		case sc.jobChan <- JobPayload{MonitorID: id}:
			// success
		case <-sc.ctx.Done():
			return
		default:
			// jobChan full → backpressure protection
			// reinsert job so it’s not lost
			if err := sc.redisSvc.Schedule(sc.ctx, item.Member.(string), time.Unix(score, 0)); err != nil {
				sc.logger.Error().Err(err).Msg("error in scheduling monitor")
				// enqueue in queue
			}
		}
	}
}
