package tenant

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TenantRecord struct {
	ID        string
	Name      string
	Domain    string
	Status    string
	Theme     string
	LogoURL   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Repository interface {
	Create(ctx context.Context, name, domain, adminEmail string) (TenantRecord, error)
	GetByID(ctx context.Context, id string) (TenantRecord, error)
	UpdateSettings(ctx context.Context, id, theme, logoURL string) (TenantRecord, error)
}

var ErrTenantNotFound = errors.New("tenant not found")

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, name, domain, adminEmail string) (TenantRecord, error) {
	var t TenantRecord
	// adminEmail is stored in the metadata JSON column for auditing; the actual admin user
	// is provisioned separately via the User/Auth service after tenant creation.
	err := r.pool.QueryRow(ctx,
		`INSERT INTO tenants (name, domain, metadata) VALUES ($1, $2, jsonb_build_object('admin_email', $3::text))
		 RETURNING id, name, domain, status, theme, logo_url, created_at, updated_at`,
		name, domain, adminEmail,
	).Scan(&t.ID, &t.Name, &t.Domain, &t.Status, &t.Theme, &t.LogoURL, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}

func (r *PostgresRepository) GetByID(ctx context.Context, id string) (TenantRecord, error) {
	var t TenantRecord
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, domain, status, theme, logo_url, created_at, updated_at FROM tenants WHERE id = $1`, id,
	).Scan(&t.ID, &t.Name, &t.Domain, &t.Status, &t.Theme, &t.LogoURL, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return t, ErrTenantNotFound
	}
	return t, err
}

func (r *PostgresRepository) UpdateSettings(ctx context.Context, id, theme, logoURL string) (TenantRecord, error) {
	var t TenantRecord
	err := r.pool.QueryRow(ctx,
		`UPDATE tenants SET theme = $2, logo_url = $3 WHERE id = $1
		 RETURNING id, name, domain, status, theme, logo_url, created_at, updated_at`,
		id, theme, logoURL,
	).Scan(&t.ID, &t.Name, &t.Domain, &t.Status, &t.Theme, &t.LogoURL, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return t, ErrTenantNotFound
	}
	return t, err
}
