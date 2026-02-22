//go:build integration

package user_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	user "github.com/amnayem/skillforge/services/user"
)

func setupPostgres(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()
	req := testcontainers.ContainerRequest{
		Image:        "postgres:17-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER": "test", "POSTGRES_PASSWORD": "test", "POSTGRES_DB": "testdb",
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

	schema := `
		CREATE EXTENSION IF NOT EXISTS "pgcrypto";
		CREATE TABLE IF NOT EXISTS tenants (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT NOT NULL UNIQUE, slug TEXT NOT NULL UNIQUE,
			settings JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT NOW(), updated_at TIMESTAMPTZ DEFAULT NOW()
		);
		INSERT INTO tenants (id, name, slug) VALUES ('00000000-0000-0000-0000-000000000001', 'Default', 'default') ON CONFLICT DO NOTHING;
		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(id),
			email TEXT NOT NULL, password_hash TEXT NOT NULL DEFAULT '',
			first_name TEXT NOT NULL DEFAULT '', last_name TEXT NOT NULL DEFAULT '',
			role TEXT NOT NULL DEFAULT 'student', is_active BOOLEAN DEFAULT TRUE,
			avatar_url TEXT DEFAULT '', bio TEXT DEFAULT '',
			created_at TIMESTAMPTZ DEFAULT NOW(), updated_at TIMESTAMPTZ DEFAULT NOW(),
			UNIQUE(email, tenant_id)
		);
	`
	if _, err = pool.Exec(ctx, schema); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}
	return pool
}

func TestIntegration_UserRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	repo := user.NewPostgresRepository(pool)
	tenantID := "00000000-0000-0000-0000-000000000001"

	var createdID string

	t.Run("Create", func(t *testing.T) {
		u, err := repo.Create(ctx, tenantID, "integ-user@test.com", "Jane", "Doe", "student")
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		createdID = u.ID
		if u.Email != "integ-user@test.com" {
			t.Errorf("expected email integ-user@test.com, got %s", u.Email)
		}
	})

	t.Run("GetByID", func(t *testing.T) {
		u, err := repo.GetByID(ctx, createdID)
		if err != nil {
			t.Fatalf("GetByID failed: %v", err)
		}
		if u.FirstName != "Jane" {
			t.Errorf("expected Jane, got %s", u.FirstName)
		}
	})

	t.Run("Update", func(t *testing.T) {
		u, err := repo.Update(ctx, createdID, "Janet", "Updated", "https://avatar.url/me.png", "My bio")
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}
		if u.FirstName != "Janet" {
			t.Errorf("expected Janet, got %s", u.FirstName)
		}
		if u.AvatarURL != "https://avatar.url/me.png" {
			t.Errorf("expected avatar url, got %s", u.AvatarURL)
		}
	})

	t.Run("List", func(t *testing.T) {
		users, total, err := repo.List(ctx, tenantID, 1, 10)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if total < 1 {
			t.Error("expected at least 1 user")
		}
		if len(users) < 1 {
			t.Error("expected at least 1 user in list")
		}
	})

	t.Run("Delete", func(t *testing.T) {
		err := repo.Delete(ctx, createdID)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
		_, err = repo.GetByID(ctx, createdID)
		if err != user.ErrUserNotFound {
			t.Errorf("expected ErrUserNotFound after delete, got %v", err)
		}
	})

	t.Run("GetByID_NotFound", func(t *testing.T) {
		_, err := repo.GetByID(ctx, "00000000-0000-0000-0000-000000000099")
		if err != user.ErrUserNotFound {
			t.Errorf("expected ErrUserNotFound, got %v", err)
		}
	})
}
