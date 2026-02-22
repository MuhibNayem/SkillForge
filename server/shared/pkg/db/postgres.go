package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// ConnectPostgres creates a production-grade PostgreSQL connection pool.
// dsn format: "postgres://user:pass@host:5432/dbname?sslmode=disable"
func ConnectPostgres(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse postgres dsn: %w", err)
	}

	cfg.MaxConns = 25
	cfg.MinConns = 5
	cfg.MaxConnLifetime = 1 * time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute
	cfg.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	// Verify connection with retry
	var pingErr error
	for i := range 5 {
		pingErr = pool.Ping(ctx)
		if pingErr == nil {
			break
		}
		log.Warn().Err(pingErr).Int("attempt", i+1).Msg("postgres ping failed, retrying...")
		time.Sleep(time.Duration(i+1) * time.Second)
	}
	if pingErr != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres ping after retries: %w", pingErr)
	}

	log.Info().Str("dsn", sanitizeDSN(dsn)).Int32("max_conns", cfg.MaxConns).Msg("PostgreSQL connected")
	return pool, nil
}

// sanitizeDSN removes password from DSN for safe logging.
func sanitizeDSN(dsn string) string {
	// Simple: just show host/db, hide credentials
	return "postgres://***@..."
}
