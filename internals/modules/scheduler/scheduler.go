package scheduler

import (
	"context"
	"time"

	"project-k/pkg/redisstore"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Scheduler struct {
	ctx       context.Context
	jobChan   chan JobPayload
	redisSvc  *redisstore.Client
	interval  time.Duration
	batchSize int
}

func NewScheduler(
	ctx context.Context,
	jobChan chan JobPayload,
	redisSvc *redisstore.Client,
) *Scheduler {

	return &Scheduler{
		ctx:       ctx,
		jobChan:   jobChan,
		redisSvc:  redisSvc,
		interval:  2 * time.Second,
		batchSize: 500,
	}
}

func (sc *Scheduler) StartScheduler() {
	ticker := time.NewTicker(sc.interval)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-sc.ctx.Done():
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
				Score: item.Score,
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
					Score: future.Score,
					Member: future.Member.(string),
				})
			}
			// put all reinserts in one call  -> this is more optimsed
			if err := sc.redisSvc.ScheduleBatch(sc.ctx, reinsert); err != nil {
				// log it
			}
			break
		}

		id, err := uuid.Parse(item.Member.(string))
		if err != nil {
			// corrupted data, skip
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
			_ = sc.redisSvc.Schedule(
				sc.ctx,
				item.Member.(string),
				time.Unix(score, 0),
			)
		}
	}
}
