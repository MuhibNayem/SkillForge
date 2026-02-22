package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

// UserRecord represents a user row in the database.
type UserRecord struct {
	ID           string
	TenantID     string
	Email        string
	PasswordHash string
	FirstName    string
	LastName     string
	Role         string
	IsActive     bool
	CreatedAt    time.Time
}

// Repository defines the data access contract for the Auth service.
type Repository interface {
	CreateUser(ctx context.Context, tenantID, email, passwordHash, firstName, lastName string) (UserRecord, error)
	GetUserByEmail(ctx context.Context, email string) (UserRecord, error)
	GetUserByID(ctx context.Context, id string) (UserRecord, error)
	StoreRefreshToken(ctx context.Context, userID, token string, ttl time.Duration) error
	ValidateRefreshToken(ctx context.Context, userID, token string) (bool, error)
	DeleteRefreshToken(ctx context.Context, userID string) error
}

// PostgresRepository implements Repository with PostgreSQL + Redis.
type PostgresRepository struct {
	pool  *pgxpool.Pool
	redis *redis.Client
}

func NewPostgresRepository(pool *pgxpool.Pool, rdb *redis.Client) *PostgresRepository {
	return &PostgresRepository{pool: pool, redis: rdb}
}

func (r *PostgresRepository) CreateUser(ctx context.Context, tenantID, email, passwordHash, firstName, lastName string) (UserRecord, error) {
	var u UserRecord
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (tenant_id, email, password_hash, first_name, last_name)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, tenant_id, email, first_name, last_name, role, is_active, created_at`,
		tenantID, email, passwordHash, firstName, lastName,
	).Scan(&u.ID, &u.TenantID, &u.Email, &u.FirstName, &u.LastName, &u.Role, &u.IsActive, &u.CreatedAt)
	return u, err
}

func (r *PostgresRepository) GetUserByEmail(ctx context.Context, email string) (UserRecord, error) {
	var u UserRecord
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, email, password_hash, first_name, last_name, role, is_active, created_at
		 FROM users WHERE email = $1 AND is_active = true`, email,
	).Scan(&u.ID, &u.TenantID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName, &u.Role, &u.IsActive, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, ErrUserNotFound
	}
	return u, err
}

func (r *PostgresRepository) GetUserByID(ctx context.Context, id string) (UserRecord, error) {
	var u UserRecord
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, email, password_hash, first_name, last_name, role, is_active, created_at
		 FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.TenantID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName, &u.Role, &u.IsActive, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, ErrUserNotFound
	}
	return u, err
}

func (r *PostgresRepository) StoreRefreshToken(ctx context.Context, userID, token string, ttl time.Duration) error {
	return r.redis.Set(ctx, "refresh:"+userID, token, ttl).Err()
}

func (r *PostgresRepository) ValidateRefreshToken(ctx context.Context, userID, token string) (bool, error) {
	stored, err := r.redis.Get(ctx, "refresh:"+userID).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return stored == token, nil
}

func (r *PostgresRepository) DeleteRefreshToken(ctx context.Context, userID string) error {
	return r.redis.Del(ctx, "refresh:"+userID).Err()
}

// HashPassword creates a bcrypt hash.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

// CheckPassword compares a plaintext password with a hash.
func CheckPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// Sentinel errors
var (
	ErrUserNotFound      = errors.New("user not found")
	ErrInvalidPassword   = errors.New("invalid password")
	ErrUserAlreadyExists = errors.New("user already exists")
)
