//go:build integration

package auth_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	auth "github.com/amnayem/skillforge/services/auth"
)

// setupPostgres starts a PostgreSQL test container and returns a pgxpool.Pool.
func setupPostgres(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()
	req := testcontainers.ContainerRequest{
		Image:        "postgres:17-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(60 * time.Second),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req, Started: true,
	})
	if err != nil {
		t.Fatalf("failed to start postgres: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "5432")

	dsn := fmt.Sprintf("postgres://test:test@%s:%s/testdb?sslmode=disable", host, port.Port())
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	// Create schema
	schema := `
		CREATE EXTENSION IF NOT EXISTS "pgcrypto";
		CREATE TABLE IF NOT EXISTS tenants (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT NOT NULL UNIQUE,
			slug TEXT NOT NULL UNIQUE,
			settings JSONB DEFAULT '{}',
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		);
		INSERT INTO tenants (id, name, slug) VALUES ('00000000-0000-0000-0000-000000000001', 'Default', 'default') ON CONFLICT DO NOTHING;
		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(id),
			email TEXT NOT NULL,
			password_hash TEXT NOT NULL DEFAULT '',
			first_name TEXT NOT NULL DEFAULT '',
			last_name TEXT NOT NULL DEFAULT '',
			role TEXT NOT NULL DEFAULT 'student',
			is_active BOOLEAN DEFAULT TRUE,
			avatar_url TEXT DEFAULT '',
			bio TEXT DEFAULT '',
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			UNIQUE(email, tenant_id)
		);
	`
	_, err = pool.Exec(ctx, schema)
	if err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}
	return pool
}

// setupRedis starts a Redis test container and returns a redis.Client.
func setupRedis(t *testing.T, ctx context.Context) *redis.Client {
	t.Helper()
	req := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections").WithStartupTimeout(30 * time.Second),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req, Started: true,
	})
	if err != nil {
		t.Fatalf("failed to start redis: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "6379")

	rdb := redis.NewClient(&redis.Options{Addr: fmt.Sprintf("%s:%s", host, port.Port())})
	t.Cleanup(func() { _ = rdb.Close() })
	return rdb
}

func TestIntegration_AuthRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	rdb := setupRedis(t, ctx)

	repo := auth.NewPostgresRepository(pool, rdb)
	tenantID := "00000000-0000-0000-0000-000000000001"

	// Test CreateUser
	t.Run("CreateUser", func(t *testing.T) {
		user, err := repo.CreateUser(ctx, tenantID, "integ@example.com", "$2a$10$fakehash", "Test", "User")
		if err != nil {
			t.Fatalf("CreateUser failed: %v", err)
		}
		if user.ID == "" {
			t.Error("expected non-empty ID")
		}
		if user.Email != "integ@example.com" {
			t.Errorf("expected integ@example.com, got %s", user.Email)
		}
		if user.Role != "student" {
			t.Errorf("expected student role, got %s", user.Role)
		}
	})

	// Test GetUserByEmail
	t.Run("GetUserByEmail", func(t *testing.T) {
		user, err := repo.GetUserByEmail(ctx, "integ@example.com")
		if err != nil {
			t.Fatalf("GetUserByEmail failed: %v", err)
		}
		if user.FirstName != "Test" {
			t.Errorf("expected Test, got %s", user.FirstName)
		}
	})

	// Test GetUserByEmail_NotFound
	t.Run("GetUserByEmail_NotFound", func(t *testing.T) {
		_, err := repo.GetUserByEmail(ctx, "nonexistent@example.com")
		if err != auth.ErrUserNotFound {
			t.Errorf("expected ErrUserNotFound, got %v", err)
		}
	})

	// Test duplicate user
	t.Run("CreateUser_Duplicate", func(t *testing.T) {
		_, err := repo.CreateUser(ctx, tenantID, "integ@example.com", "$2a$10$fakehash", "Dup", "User")
		if err == nil {
			t.Error("expected error for duplicate email")
		}
	})

	// Test refresh token
	t.Run("RefreshToken", func(t *testing.T) {
		err := repo.StoreRefreshToken(ctx, "user-123", "refresh-abc", 5*time.Minute)
		if err != nil {
			t.Fatalf("StoreRefreshToken failed: %v", err)
		}
		valid, err := repo.ValidateRefreshToken(ctx, "user-123", "refresh-abc")
		if err != nil {
			t.Fatalf("ValidateRefreshToken failed: %v", err)
		}
		if !valid {
			t.Error("expected valid=true")
		}
		// Wrong token
		valid, _ = repo.ValidateRefreshToken(ctx, "user-123", "wrong-token")
		if valid {
			t.Error("expected valid=false for wrong token")
		}
		// Delete
		err = repo.DeleteRefreshToken(ctx, "user-123")
		if err != nil {
			t.Fatalf("DeleteRefreshToken failed: %v", err)
		}
		valid, _ = repo.ValidateRefreshToken(ctx, "user-123", "refresh-abc")
		if valid {
			t.Error("expected valid=false after delete")
		}
	})
}
