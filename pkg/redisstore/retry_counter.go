package redisstore

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func (c *Client) IncrementRetry(ctx context.Context, monitorID uuid.UUID) (int64, error) {
	key := fmt.Sprintf("monitor:retry:%v", monitorID)

	var count int64

	err := retry(ctx, 2, func() error {
		var err error
		count, err = c.rdb.Incr(ctx, key).Result()
		if err != nil {
			return err
		}

		c.rdb.Expire(ctx, key, 5*time.Minute)
		return nil
	})

	return count, err
}

func (c *Client) ClearRetry(ctx context.Context, monitorID uuid.UUID) error {
	key := fmt.Sprintf("monitor:retry:%v", monitorID)

	return retry(ctx, 2, func() error {
		return c.rdb.Del(ctx, key).Err()
	})
}
