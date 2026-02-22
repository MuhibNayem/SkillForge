package user

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRecord struct {
	ID        string
	TenantID  string
	Email     string
	FirstName string
	LastName  string
	Role      string
	AvatarURL string
	Bio       string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Repository interface {
	Create(ctx context.Context, tenantID, email, firstName, lastName, role string) (UserRecord, error)
	GetByID(ctx context.Context, id string) (UserRecord, error)
	Update(ctx context.Context, id, firstName, lastName, avatarURL, bio string) (UserRecord, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, tenantID string, page, pageSize int) ([]UserRecord, int, error)
}

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

var ErrUserNotFound = errors.New("user not found")

func (r *PostgresRepository) Create(ctx context.Context, tenantID, email, firstName, lastName, role string) (UserRecord, error) {
	var u UserRecord
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (tenant_id, email, password_hash, first_name, last_name, role)
		 VALUES ($1, $2, '', $3, $4, $5)
		 RETURNING id, tenant_id, email, first_name, last_name, role, avatar_url, bio, is_active, created_at, updated_at`,
		tenantID, email, firstName, lastName, role,
	).Scan(&u.ID, &u.TenantID, &u.Email, &u.FirstName, &u.LastName, &u.Role, &u.AvatarURL, &u.Bio, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	return u, err
}

func (r *PostgresRepository) GetByID(ctx context.Context, id string) (UserRecord, error) {
	var u UserRecord
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, email, first_name, last_name, role, avatar_url, bio, is_active, created_at, updated_at
		 FROM users WHERE id = $1 AND is_active = true`, id,
	).Scan(&u.ID, &u.TenantID, &u.Email, &u.FirstName, &u.LastName, &u.Role, &u.AvatarURL, &u.Bio, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, ErrUserNotFound
	}
	return u, err
}

func (r *PostgresRepository) Update(ctx context.Context, id, firstName, lastName, avatarURL, bio string) (UserRecord, error) {
	var u UserRecord
	err := r.pool.QueryRow(ctx,
		`UPDATE users SET first_name = $2, last_name = $3, avatar_url = $4, bio = $5
		 WHERE id = $1 AND is_active = true
		 RETURNING id, tenant_id, email, first_name, last_name, role, avatar_url, bio, is_active, created_at, updated_at`,
		id, firstName, lastName, avatarURL, bio,
	).Scan(&u.ID, &u.TenantID, &u.Email, &u.FirstName, &u.LastName, &u.Role, &u.AvatarURL, &u.Bio, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, ErrUserNotFound
	}
	return u, err
}

func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `UPDATE users SET is_active = false WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *PostgresRepository) List(ctx context.Context, tenantID string, page, pageSize int) ([]UserRecord, int, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * pageSize

	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE tenant_id = $1 AND is_active = true`, tenantID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, email, first_name, last_name, role, avatar_url, bio, is_active, created_at, updated_at
		 FROM users WHERE tenant_id = $1 AND is_active = true ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		tenantID, pageSize, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []UserRecord
	for rows.Next() {
		var u UserRecord
		if err := rows.Scan(&u.ID, &u.TenantID, &u.Email, &u.FirstName, &u.LastName, &u.Role, &u.AvatarURL, &u.Bio, &u.IsActive, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	return users, total, rows.Err()
}
