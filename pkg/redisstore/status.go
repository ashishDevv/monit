package redisstore

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func (c *Client) StoreStatus(ctx context.Context, monitorID uuid.UUID, statusCode int, latencyMs int64, checkedAt time.Time) error {
	key := fmt.Sprintf("monitor:status:%v",monitorID)

	return retry(ctx, 2, func() error {
		return c.rdb.HSet(ctx, key, map[string]any{
			"status_code": statusCode,
			"latency_ms": latencyMs,
			"checked_at": checkedAt.Unix(),
		}).Err()
	})
}

func (c *Client) GetStatus(ctx context.Context, monitorID uuid.UUID) (map[string]string, error){
	key := fmt.Sprintf("monitor:status:%v",monitorID)

	res, err := c.rdb.HGetAll(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	return res, err
}

func (c *Client) DelStatus(ctx context.Context, monitorID uuid.UUID) error {
	key := fmt.Sprintf("monitor:status:%v",monitorID)

	return c.rdb.Del(ctx, key).Err()
}