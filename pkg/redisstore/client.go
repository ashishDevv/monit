package redisstore

import (
	"context"
	"project-k/config"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ErrKeyNotFound = redis.Nil
)

type Client struct {
	rdb *redis.Client
}

func New(redisCfg *config.RedisConfig) (*Client, error) {
	opt, err := redis.ParseURL(redisCfg.URL)
	if err != nil {
		return nil, err
	}

	// Timeouts
	// opt.DialTimeout = 5 * time.Second
	// opt.ReadTimeout = 3 * time.Second
	// opt.WriteTimeout = 3 * time.Second
	opt.DialTimeout = redisCfg.DialTimeout
	opt.ReadTimeout = redisCfg.ReadTimeout
	opt.WriteTimeout = redisCfg.WriteTimeout

	// Pool tuning
	// opt.PoolSize = 10
	// opt.MinIdleConns = 5
	opt.PoolSize = redisCfg.PoolSize
	opt.MinIdleConns = redisCfg.MinIdleConns

	// Connection lifecycle
	// opt.ConnMaxLifetime = 2 * time.Minute  
	// opt.ConnMaxIdleTime = 30 * time.Second 
	opt.ConnMaxLifetime = redisCfg.ConnMaxLifetime  
	opt.ConnMaxIdleTime = redisCfg.ConnMaxIdleTime

	rdb := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &Client{rdb: rdb}, nil
}

func (c *Client) Close() error {
	return c.rdb.Close()
}
