package redisstore

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// func (c *Client) CreateIncident(ctx context.Context, monitorID uuid.UUID) error {
// 	key := fmt.Sprintf("monitor:incident:%v", monitorID)

// 	return retry(ctx, 3, func() error {
// 		return c.rdb.HSet(ctx, key, map[string]any{
// 			"failure_count": 1,
// 			"first_failure_at": time.Now().Unix(),
// 			"last_failure_at": time.Now().Unix(),
// 			"alerted": false,
// 		}).Err()
// 	})
// }

func (c *Client) IncrementIncident(ctx context.Context, monitorID uuid.UUID) (int64, bool, error) {
	key := fmt.Sprintf("monitor:incident:%v", monitorID)
	now := time.Now().Unix()

	var failureCount int64
	var firstTime bool

	err := retry(ctx, 3, func() error {
		var err error

		// 1. Increment failure count (creates hash if missing)
		failureCount, err = c.rdb.HIncrBy(ctx, key, "failure_count", 1).Result()
		if err != nil {
			return err
		}

		// 2. Set timestamps
		if failureCount == 1 {
			// first incident -> new inicident
			c.rdb.HSet(ctx, key,
				"first_failure_at", now,
				"last_failure_at", now,
				"alerted", false,
			)
			firstTime = true
		} else {
			// existing increment
			c.rdb.HSet(ctx, key, "last_failure_at", now)
		}

		return nil
	})

	return failureCount, firstTime, err
}

func (c *Client) ClearIncident(ctx context.Context, monitorID uuid.UUID) error {
	key := fmt.Sprintf("monitor:incident:%v", monitorID)

	return retry(ctx, 2, func() error {
		return c.rdb.Del(ctx, key).Err()
	})
}

func (c *Client) MarkIncidentAlerted(ctx context.Context, monitorID uuid.UUID) error {
	key := fmt.Sprintf("monitor:incident:%v", monitorID)
    return c.rdb.HSet(ctx,
        key,
        "alerted", true,
    ).Err()
}

func (c *Client) GetIncidentAlerted(ctx context.Context, monitorID uuid.UUID) (bool, error) {
	key := fmt.Sprintf("monitor:incident:%v", monitorID)
	
	resp, err := c.rdb.HGet(ctx, key, "alerted").Result()
	if err != nil {
		return false, err
	}
	if resp == "true" {
		return true, nil
	} else {
		return false, nil
	}
}

func (c *Client) GetIncident(ctx context.Context, monitorID uuid.UUID) (map[string]string, error) {
	key := fmt.Sprintf("monitor:incident:%v", monitorID)

	resp, err := c.rdb.HGetAll(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	return resp, err
} 
