package db

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ConnectMongo creates a production MongoDB client with connection pooling.
func ConnectMongo(ctx context.Context, uri string) (*mongo.Client, error) {
	opts := options.Client().
		ApplyURI(uri).
		SetMaxPoolSize(25).
		SetMinPoolSize(5).
		SetMaxConnIdleTime(30 * time.Minute).
		SetConnectTimeout(10 * time.Second).
		SetServerSelectionTimeout(5 * time.Second)

	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, fmt.Errorf("create mongo client: %w", err)
	}

	var pingErr error
	for i := range 5 {
		pingErr = client.Ping(ctx, nil)
		if pingErr == nil {
			break
		}
		log.Warn().Err(pingErr).Int("attempt", i+1).Msg("mongo ping failed, retrying...")
		time.Sleep(time.Duration(i+1) * time.Second)
	}
	if pingErr != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("mongo ping after retries: %w", pingErr)
	}

	log.Info().Msg("MongoDB connected")
	return client, nil
}
