package notification

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PreferencesRecord holds per-user notification opt-in settings.
type PreferencesRecord struct {
	UserID             string
	EmailCourseUpdates bool
	EmailMarketing     bool
	InAppMentions      bool
	UpdatedAt          time.Time
}

// DeliveryRecord stores the outcome of a notification dispatch attempt.
type DeliveryRecord struct {
	ID        string
	UserID    string
	Channel   string // "email" | "in_app"
	EventType string
	Payload   string
	Status    string // "pending" | "sent" | "failed"
	CreatedAt time.Time
}

// Repository is the storage interface for the Notification Service.
type Repository interface {
	UpsertPreferences(ctx context.Context, p PreferencesRecord) (PreferencesRecord, error)
	GetPreferences(ctx context.Context, userID string) (PreferencesRecord, error)
	RecordDelivery(ctx context.Context, d DeliveryRecord) error
}

var ErrPreferencesNotFound = errors.New("preferences not found")

// PostgresRepository implements Repository backed by PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) UpsertPreferences(ctx context.Context, p PreferencesRecord) (PreferencesRecord, error) {
	var out PreferencesRecord
	err := r.pool.QueryRow(ctx,
		`INSERT INTO notification_preferences
		   (user_id, email_course_updates, email_marketing, in_app_mentions, updated_at)
		 VALUES ($1, $2, $3, $4, NOW())
		 ON CONFLICT (user_id) DO UPDATE SET
		   email_course_updates = EXCLUDED.email_course_updates,
		   email_marketing      = EXCLUDED.email_marketing,
		   in_app_mentions      = EXCLUDED.in_app_mentions,
		   updated_at           = NOW()
		 RETURNING user_id, email_course_updates, email_marketing, in_app_mentions, updated_at`,
		p.UserID, p.EmailCourseUpdates, p.EmailMarketing, p.InAppMentions,
	).Scan(&out.UserID, &out.EmailCourseUpdates, &out.EmailMarketing, &out.InAppMentions, &out.UpdatedAt)
	return out, err
}

func (r *PostgresRepository) GetPreferences(ctx context.Context, userID string) (PreferencesRecord, error) {
	var p PreferencesRecord
	err := r.pool.QueryRow(ctx,
		`SELECT user_id, email_course_updates, email_marketing, in_app_mentions, updated_at
		 FROM notification_preferences WHERE user_id = $1`, userID,
	).Scan(&p.UserID, &p.EmailCourseUpdates, &p.EmailMarketing, &p.InAppMentions, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, ErrPreferencesNotFound
	}
	return p, err
}

func (r *PostgresRepository) RecordDelivery(ctx context.Context, d DeliveryRecord) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO notification_deliveries (user_id, channel, event_type, payload, status)
		 VALUES ($1, $2, $3, $4, $5)`,
		d.UserID, d.Channel, d.EventType, d.Payload, d.Status,
	)
	return err
}
