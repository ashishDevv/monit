package redisstore

import (
	"context"
	"time"
)

func (c *Client) ReclaimMonitors(ctx context.Context, script string, now time.Time, limit int) (int64, error) {
	
	nowMillis := now.UnixMilli()

	result, err := c.rdb.Eval(ctx, script, []string{inflightKey, scheduleKey}, nowMillis, limit).Result()
	if err != nil {
		return 0, err
	}
	count, ok := result.(int64)
	if !ok {
		return 0, nil
	}

	return count, nil
}
