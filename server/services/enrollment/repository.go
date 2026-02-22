package enrollment

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EnrollmentRecord struct {
	ID              string
	TenantID        string
	UserID          string
	CourseID        string
	Status          string
	OverallProgress int
	EnrolledAt      time.Time
	CompletedAt     *time.Time
}

type LessonProgressRecord struct {
	ID              string
	EnrollmentID    string
	LessonID        string
	PercentComplete int
	LastAccessed    time.Time
}

type Repository interface {
	Enroll(ctx context.Context, tenantID, userID, courseID string) (EnrollmentRecord, error)
	GetByID(ctx context.Context, id string) (EnrollmentRecord, error)
	List(ctx context.Context, userID, tenantID string, page, pageSize int) ([]EnrollmentRecord, int, error)
	UpdateProgress(ctx context.Context, enrollmentID, lessonID string, percent int) error
	GetProgress(ctx context.Context, enrollmentID string) ([]LessonProgressRecord, error)
	RecalculateOverallProgress(ctx context.Context, enrollmentID string) (int, error)
}

var (
	ErrEnrollmentNotFound = errors.New("enrollment not found")
	ErrAlreadyEnrolled    = errors.New("already enrolled")
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Enroll(ctx context.Context, tenantID, userID, courseID string) (EnrollmentRecord, error) {
	var e EnrollmentRecord
	err := r.pool.QueryRow(ctx,
		`INSERT INTO enrollments (tenant_id, user_id, course_id) VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, course_id) DO NOTHING
		 RETURNING id, tenant_id, user_id, course_id, status, overall_progress, enrolled_at, completed_at`,
		tenantID, userID, courseID,
	).Scan(&e.ID, &e.TenantID, &e.UserID, &e.CourseID, &e.Status, &e.OverallProgress, &e.EnrolledAt, &e.CompletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		// ON CONFLICT DO NOTHING returns no row — means already enrolled
		return e, ErrAlreadyEnrolled
	}
	if err != nil {
		return e, err
	}
	if e.ID == "" {
		// Shouldn't happen but guard anyway
		return e, ErrAlreadyEnrolled
	}
	return e, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id string) (EnrollmentRecord, error) {
	var e EnrollmentRecord
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, user_id, course_id, status, overall_progress, enrolled_at, completed_at
		 FROM enrollments WHERE id = $1`, id,
	).Scan(&e.ID, &e.TenantID, &e.UserID, &e.CourseID, &e.Status, &e.OverallProgress, &e.EnrolledAt, &e.CompletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return e, ErrEnrollmentNotFound
	}
	return e, err
}

func (r *PostgresRepository) List(ctx context.Context, userID, tenantID string, page, pageSize int) ([]EnrollmentRecord, int, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * pageSize

	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM enrollments WHERE user_id = $1 AND tenant_id = $2`, userID, tenantID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, user_id, course_id, status, overall_progress, enrolled_at, completed_at
		 FROM enrollments WHERE user_id = $1 AND tenant_id = $2 ORDER BY enrolled_at DESC LIMIT $3 OFFSET $4`,
		userID, tenantID, pageSize, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var enrollments []EnrollmentRecord
	for rows.Next() {
		var e EnrollmentRecord
		if err := rows.Scan(&e.ID, &e.TenantID, &e.UserID, &e.CourseID, &e.Status, &e.OverallProgress, &e.EnrolledAt, &e.CompletedAt); err != nil {
			return nil, 0, err
		}
		enrollments = append(enrollments, e)
	}
	return enrollments, total, rows.Err()
}

func (r *PostgresRepository) UpdateProgress(ctx context.Context, enrollmentID, lessonID string, percent int) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO lesson_progress (enrollment_id, lesson_id, percent_complete, last_accessed)
		 VALUES ($1, $2, $3, NOW())
		 ON CONFLICT (enrollment_id, lesson_id) DO UPDATE SET percent_complete = $3, last_accessed = NOW()`,
		enrollmentID, lessonID, percent,
	)
	return err
}

func (r *PostgresRepository) GetProgress(ctx context.Context, enrollmentID string) ([]LessonProgressRecord, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, enrollment_id, lesson_id, percent_complete, last_accessed
		 FROM lesson_progress WHERE enrollment_id = $1 ORDER BY last_accessed DESC`, enrollmentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var progress []LessonProgressRecord
	for rows.Next() {
		var p LessonProgressRecord
		if err := rows.Scan(&p.ID, &p.EnrollmentID, &p.LessonID, &p.PercentComplete, &p.LastAccessed); err != nil {
			return nil, err
		}
		progress = append(progress, p)
	}
	return progress, rows.Err()
}

func (r *PostgresRepository) RecalculateOverallProgress(ctx context.Context, enrollmentID string) (int, error) {
	// Calculate average progress across all lessons for this enrollment's course
	var avg int
	err := r.pool.QueryRow(ctx,
		`WITH course_lessons AS (
			SELECT l.id FROM lessons l
			JOIN modules m ON m.id = l.module_id
			JOIN courses c ON c.id = m.course_id
			JOIN enrollments e ON e.course_id = c.id
			WHERE e.id = $1
		)
		SELECT COALESCE(AVG(COALESCE(lp.percent_complete, 0))::int, 0)
		FROM course_lessons cl
		LEFT JOIN lesson_progress lp ON lp.lesson_id = cl.id AND lp.enrollment_id = $1`,
		enrollmentID,
	).Scan(&avg)
	if err != nil {
		return 0, err
	}

	// Update enrollment and mark complete if 100%
	status := "active"
	if avg >= 100 {
		status = "completed"
	}
	_, err = r.pool.Exec(ctx,
		`UPDATE enrollments SET overall_progress = $2, status = $3,
		 completed_at = CASE WHEN $3 = 'completed' THEN NOW() ELSE completed_at END
		 WHERE id = $1`, enrollmentID, avg, status,
	)
	return avg, err
}
