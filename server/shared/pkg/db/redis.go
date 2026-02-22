package db

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// ConnectRedis creates a production Redis client with health check.
func ConnectRedis(ctx context.Context, addr, password string, db int) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:            addr,
		Password:        password,
		DB:              db,
		MaxRetries:      3,
		MinRetryBackoff: 100 * time.Millisecond,
		MaxRetryBackoff: 2 * time.Second,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		PoolSize:        20,
		MinIdleConns:    5,
	})

	var pingErr error
	for i := range 5 {
		pingErr = client.Ping(ctx).Err()
		if pingErr == nil {
			break
		}
		log.Warn().Err(pingErr).Int("attempt", i+1).Msg("redis ping failed, retrying...")
		time.Sleep(time.Duration(i+1) * time.Second)
	}
	if pingErr != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis ping after retries: %w", pingErr)
	}

	log.Info().Str("addr", addr).Msg("Redis connected")
	return client, nil
}
