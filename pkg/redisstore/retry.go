package redisstore

import (
	"context"
	"time"
)

func retry(ctx context.Context, attempts int, fn func() error) error {
	var err error

	for i := 0; i < attempts; i++ {
		err = fn()
		if err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(50*(i+1)) * time.Millisecond):
		}
	}

	return err
}
