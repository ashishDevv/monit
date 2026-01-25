package redisstore

import (
	"context"
	"encoding/json"
	"fmt"
	"project-k/internals/modules/monitor"
	"time"

	"github.com/google/uuid"
)

func (c *Client) SetMonitor(ctx context.Context, m monitor.Monitor) error {
	key := fmt.Sprintf("monitor:%v", m.ID.String())

	jsonM, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, key, jsonM, 24*time.Hour).Err()
}

func (c *Client) GetMonitor(ctx context.Context, id uuid.UUID) (monitor.Monitor, bool) {
	key := fmt.Sprintf("monitor:%v", id.String())

	res, err := c.rdb.Get(ctx, key).Bytes()
	if err != nil {
		return monitor.Monitor{}, false
	}
	var m monitor.Monitor
	if err := json.Unmarshal(res, &m); err != nil {
		return monitor.Monitor{}, false
	}

	return m, true
}

func (c *Client) DelMonitor(ctx context.Context, id uuid.UUID) error {
	key := fmt.Sprintf("monitor:%v", id.String())

	return c.rdb.Del(ctx, key).Err()
}
