package redisstore

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const scheduleKey string = "monitor:schedule"
const inflightKey string = "monitor:inflight"

func (c *Client) Schedule(ctx context.Context, monitorID string, nextRun time.Time) error {
	return retry(ctx, 3, func() error {
		return c.rdb.ZAdd(ctx, scheduleKey, redis.Z{
			Score:  float64(nextRun.UnixMilli()),
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

func (c *Client) FetchDueMonitors(ctx context.Context, script string, now time.Time, limit int) ([]string, error) {
	nowMillis := now.UnixMilli()

	result, err := c.rdb.Eval(ctx, script, []string{scheduleKey}, nowMillis, limit).Result()
	if err != nil {
		return nil, err
	}

	rawItems, ok := result.([]any)
	if !ok {
		return nil, nil
	}

	jobs := make([]string, 0, len(rawItems))
	for _, items := range rawItems {
		str, ok := items.(string)
		if ok {
			jobs = append(jobs, str)
		}
	}

	return jobs, nil
}

func (c *Client) FetchAndMoveToInflight(ctx context.Context, script string, now time.Time, limit int, visibilityTimeout time.Duration) ([]string, error) {

	nowMillis := now.UnixMilli()
	visibilityMillis := visibilityTimeout.Milliseconds()

	result, err := c.rdb.Eval(
		ctx,
		script,
		[]string{scheduleKey, inflightKey},
		nowMillis,
		limit,
		visibilityMillis,
	).Result()

	if err != nil {
		return nil, err
	}

	rawItems, ok := result.([]any)
	if !ok {
		return nil, nil
	}

	jobs := make([]string, 0, len(rawItems))

	for _, item := range rawItems {
		if str, ok := item.(string); ok {
			jobs = append(jobs, str)
		}
	}

	return jobs, nil
}

func (c *Client) AckJob(ctx context.Context, monitorID string) error {
	return c.rdb.ZRem(ctx, inflightKey, monitorID).Err()
}
