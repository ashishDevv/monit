package redisstore

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const scheduleKey string = "monitor:schedule"

func (c *Client) Schedule(ctx context.Context, monitorID string, runAt time.Time) error {
	return retry(ctx, 3, func() error {
		return c.rdb.ZAdd(ctx, scheduleKey, redis.Z{
			Score: float64(runAt.Unix()),
			Member: monitorID,
		}).Err()
	})
}

func (c *Client) ScheduleBatch(ctx context.Context, items []redis.Z) error {
    if len(items) == 0 {
        return nil
    }

    return retry(ctx, 3, func() error {
        return c.rdb.ZAdd(ctx, scheduleKey, items...).Err()
    })
}

func (c *Client) PopDue(ctx context.Context, batchCount int) ([]redis.Z, error) {
	var res []redis.Z

	err := retry(ctx, 3, func() error {
		var err error
		res, err = c.rdb.ZPopMin(ctx, scheduleKey, int64(batchCount)).Result()
		return err
	})

	return res, err
}

func (c *Client) DelSchedule(ctx context.Context, monitorID string) error {
	
	return c.rdb.ZRem(ctx, scheduleKey, monitorID).Err()
}