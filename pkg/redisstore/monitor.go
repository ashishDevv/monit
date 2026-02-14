package redisstore

import (
	"context"
	"fmt"
	"project-k/pkg/apperror"
	"time"

	"github.com/redis/go-redis/v9"
)

func (c *Client) SetMonitor(ctx context.Context, id string, data []byte, ttl time.Duration) error {
	key := fmt.Sprintf("monitor:%v", id)

	return c.rdb.Set(ctx, key, data, ttl).Err()
}

func (c *Client) GetMonitor(ctx context.Context, id string) ([]byte, error) {
	key := fmt.Sprintf("monitor:%v", id)

	res, err := c.rdb.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, &apperror.Error{
				Kind: apperror.NotFound,
			}
		}
		return nil, err
	}

	return res, nil
}

func (c *Client) DelMonitor(ctx context.Context, id string) error {
	key := fmt.Sprintf("monitor:%v", id)

	return c.rdb.Del(ctx, key).Err()
}
