//go:build integration

package course_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	course "github.com/amnayem/skillforge/services/course"
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
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(), name TEXT NOT NULL UNIQUE,
			slug TEXT NOT NULL UNIQUE, settings JSONB DEFAULT '{}',
			created_at TIMESTAMPTZ DEFAULT NOW(), updated_at TIMESTAMPTZ DEFAULT NOW()
		);
		INSERT INTO tenants (id, name, slug) VALUES ('00000000-0000-0000-0000-000000000001', 'Default', 'default') ON CONFLICT DO NOTHING;
		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(), tenant_id UUID NOT NULL REFERENCES tenants(id),
			email TEXT NOT NULL, password_hash TEXT DEFAULT '', first_name TEXT DEFAULT '', last_name TEXT DEFAULT '',
			role TEXT DEFAULT 'student', is_active BOOLEAN DEFAULT TRUE, avatar_url TEXT DEFAULT '', bio TEXT DEFAULT '',
			created_at TIMESTAMPTZ DEFAULT NOW(), updated_at TIMESTAMPTZ DEFAULT NOW(), UNIQUE(email, tenant_id)
		);
		INSERT INTO users (id, tenant_id, email, first_name) VALUES ('00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'inst@test.com', 'Instructor') ON CONFLICT DO NOTHING;
		CREATE TABLE IF NOT EXISTS courses (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(), tenant_id UUID NOT NULL REFERENCES tenants(id),
			instructor_id UUID NOT NULL REFERENCES users(id), title TEXT NOT NULL, description TEXT DEFAULT '',
			category TEXT DEFAULT '', difficulty TEXT DEFAULT 'beginner' CHECK (difficulty IN ('beginner','intermediate','advanced')),
			status TEXT DEFAULT 'draft' CHECK (status IN ('draft','published','archived')),
			thumbnail_url TEXT DEFAULT '', created_at TIMESTAMPTZ DEFAULT NOW(), updated_at TIMESTAMPTZ DEFAULT NOW()
		);
		CREATE TABLE IF NOT EXISTS modules (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(), course_id UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
			title TEXT NOT NULL, position INTEGER NOT NULL DEFAULT 0, created_at TIMESTAMPTZ DEFAULT NOW()
		);
		CREATE TABLE IF NOT EXISTS lessons (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(), module_id UUID NOT NULL REFERENCES modules(id) ON DELETE CASCADE,
			title TEXT NOT NULL, type TEXT NOT NULL DEFAULT 'video' CHECK (type IN ('video','text','quiz')),
			content_id TEXT DEFAULT '', position INTEGER NOT NULL DEFAULT 0,
			duration_seconds INTEGER DEFAULT 0, created_at TIMESTAMPTZ DEFAULT NOW()
		);
	`
	if _, err = pool.Exec(ctx, schema); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}
	return pool
}

func TestIntegration_CourseRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	repo := course.NewPostgresRepository(pool)
	tenantID := "00000000-0000-0000-0000-000000000001"
	instructorID := "00000000-0000-0000-0000-000000000002"

	var courseID, moduleID string

	t.Run("Create", func(t *testing.T) {
		c, err := repo.Create(ctx, tenantID, instructorID, "Test Course", "A test", "AI", "beginner")
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		courseID = c.ID
		if c.Status != "draft" {
			t.Errorf("expected draft, got %s", c.Status)
		}
	})

	t.Run("GetByID", func(t *testing.T) {
		c, err := repo.GetByID(ctx, courseID)
		if err != nil {
			t.Fatalf("GetByID failed: %v", err)
		}
		if c.Title != "Test Course" {
			t.Errorf("expected Test Course, got %s", c.Title)
		}
	})

	t.Run("Publish", func(t *testing.T) {
		c, err := repo.Publish(ctx, courseID)
		if err != nil {
			t.Fatalf("Publish failed: %v", err)
		}
		if c.Status != "published" {
			t.Errorf("expected published, got %s", c.Status)
		}
	})

	t.Run("AddModule", func(t *testing.T) {
		mod, err := repo.AddModule(ctx, courseID, "Module A", 1)
		if err != nil {
			t.Fatalf("AddModule failed: %v", err)
		}
		moduleID = mod.ID
		if mod.Title != "Module A" {
			t.Errorf("expected Module A, got %s", mod.Title)
		}
	})

	t.Run("AddLesson", func(t *testing.T) {
		l, err := repo.AddLesson(ctx, courseID, moduleID, "Lesson 1", "video", "", 1)
		if err != nil {
			t.Fatalf("AddLesson failed: %v", err)
		}
		if l.Type != "video" {
			t.Errorf("expected video, got %s", l.Type)
		}
	})

	t.Run("GetModules", func(t *testing.T) {
		mods, err := repo.GetModules(ctx, courseID)
		if err != nil {
			t.Fatalf("GetModules failed: %v", err)
		}
		if len(mods) != 1 {
			t.Errorf("expected 1 module, got %d", len(mods))
		}
	})

	t.Run("GetLessons", func(t *testing.T) {
		lessons, err := repo.GetLessons(ctx, moduleID)
		if err != nil {
			t.Fatalf("GetLessons failed: %v", err)
		}
		if len(lessons) != 1 {
			t.Errorf("expected 1 lesson, got %d", len(lessons))
		}
	})

	t.Run("List", func(t *testing.T) {
		courses, total, err := repo.List(ctx, tenantID, "", 1, 10)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if total < 1 {
			t.Errorf("expected total >= 1, got %d", total)
		}
		if len(courses) < 1 {
			t.Error("expected at least 1 course in list")
		}
	})

	t.Run("Delete", func(t *testing.T) {
		err := repo.Delete(ctx, courseID)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
		_, err = repo.GetByID(ctx, courseID)
		if err != course.ErrCourseNotFound {
			t.Errorf("expected ErrCourseNotFound, got %v", err)
		}
	})
}
