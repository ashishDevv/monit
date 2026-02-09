package redisstore

import (
	"context"
	// "fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ErrKeyNotFound = redis.Nil
)

type Client struct {
	rdb *redis.Client
}

func New(redisURL string) (*Client, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	// Timeouts
	opt.DialTimeout = 5 * time.Second
	opt.ReadTimeout = 3 * time.Second
	opt.WriteTimeout = 3 * time.Second

	// Pool tuning
	opt.PoolSize = 10
	opt.MinIdleConns = 5

	// Connection lifecycle
	opt.ConnMaxLifetime = 2 * time.Minute  
	opt.ConnMaxIdleTime = 30 * time.Second 

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

// func (r *Redis) GetAndRemoveMonitors(ctx context.Context, key string, batchSize int) ([]redis.Z, error) {
// 	val, err := r.redisClient.ZPopMin(ctx, key, int64(batchSize)).Result()
// 	if err != nil {
// 		return nil, err
// 	}
// 	return val, nil
// }

// func (r *Redis) ZAdd(ctx context.Context, key string, nextSchedule float64  , monitorID any) error {
// 	err := r.redisClient.ZAdd(ctx, key, redis.Z{
// 		Score: nextSchedule,
// 		Member: monitorID,
// 	}).Err()
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

// func (r *Redis) Get(ctx context.Context, key string) (string, error) {
// 	val, err := r.redisClient.Get(ctx, key).Result()
//     if err != nil {
//         return "", err
//     }
// 	return val, nil
// }

// func (r *Redis) Set(ctx context.Context, key, val string) error {
// 	return r.redisClient.Set(ctx, key, val, 0).Err()
// }

// func (r *Redis) HSet(ctx context.Context, hash, userID, deviceID, email string, ttl time.Duration) error {
// 	key := fmt.Sprintf("refresh:%v", hash)
// 	err := r.redisClient.HSet(ctx, key, map[string]any{
// 		"user_id":   userID,
// 		"device_id": deviceID,
// 		"email": email,
// 	}).Err()

// 	if err != nil {
// 		return err
// 	}
// 	err = r.redisClient.Expire(ctx, key, ttl).Err()
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

// func (r *Redis) HSetAny(ctx context.Context, key string, ttl time.Duration, payload map[string]any) error {

// 	err := r.redisClient.HSet(ctx, key, payload).Err()
// 	if err != nil {
// 		return err
// 	}
// 	err = r.redisClient.Expire(ctx, key, ttl).Err()
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

// func (r *Redis) HGet(ctx context.Context, hash string) (string, error){
// 	key := fmt.Sprintf("refresh:%v", hash)
// 	val, err := r.redisClient.HGet(ctx, key, "user_id").Result()
// 	if err != nil {
// 		return "", err
// 	}
// 	return val, nil
// }

// func (r *Redis) HGetAll(ctx context.Context, hash string) (map[string]string, error){
// 	key := fmt.Sprintf("refresh:%v", hash)
// 	return r.redisClient.HGetAll(ctx, key).Result()
// }

// func (r *Redis) Del(ctx context.Context, hash string) error {
// 	key := fmt.Sprintf("refresh:%v", hash)
// 	return r.redisClient.Del(ctx, key).Err()
// }
