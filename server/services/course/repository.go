package course

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CourseRecord struct {
	ID           string
	TenantID     string
	InstructorID string
	Title        string
	Description  string
	Category     string
	Difficulty   string
	Status       string
	ThumbnailURL string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type ModuleRecord struct {
	ID       string
	CourseID string
	Title    string
	Position int
}

type LessonRecord struct {
	ID        string
	ModuleID  string
	Title     string
	Type      string
	ContentID string
	Position  int
}

type Repository interface {
	Create(ctx context.Context, tenantID, instructorID, title, desc, category, difficulty string) (CourseRecord, error)
	GetByID(ctx context.Context, id string) (CourseRecord, error)
	Update(ctx context.Context, id, title, desc, category, difficulty, thumbnailURL string) (CourseRecord, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, tenantID, category string, page, pageSize int) ([]CourseRecord, int, error)
	Publish(ctx context.Context, id string) (CourseRecord, error)
	AddModule(ctx context.Context, courseID, title string, position int) (ModuleRecord, error)
	AddLesson(ctx context.Context, courseID, moduleID, title, lessonType, contentID string, position int) (LessonRecord, error)
	GetModules(ctx context.Context, courseID string) ([]ModuleRecord, error)
	GetLessons(ctx context.Context, moduleID string) ([]LessonRecord, error)
}

var (
	ErrCourseNotFound = errors.New("course not found")
	ErrModuleNotFound = errors.New("module not found")
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, tenantID, instructorID, title, desc, category, difficulty string) (CourseRecord, error) {
	var c CourseRecord
	err := r.pool.QueryRow(ctx,
		`INSERT INTO courses (tenant_id, instructor_id, title, description, category, difficulty)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, tenant_id, instructor_id, title, description, category, difficulty, status, thumbnail_url, created_at, updated_at`,
		tenantID, instructorID, title, desc, category, difficulty,
	).Scan(&c.ID, &c.TenantID, &c.InstructorID, &c.Title, &c.Description, &c.Category, &c.Difficulty, &c.Status, &c.ThumbnailURL, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

func (r *PostgresRepository) GetByID(ctx context.Context, id string) (CourseRecord, error) {
	var c CourseRecord
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, instructor_id, title, description, category, difficulty, status, thumbnail_url, created_at, updated_at
		 FROM courses WHERE id = $1`, id,
	).Scan(&c.ID, &c.TenantID, &c.InstructorID, &c.Title, &c.Description, &c.Category, &c.Difficulty, &c.Status, &c.ThumbnailURL, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, ErrCourseNotFound
	}
	return c, err
}

func (r *PostgresRepository) Update(ctx context.Context, id, title, desc, category, difficulty, thumbnailURL string) (CourseRecord, error) {
	var c CourseRecord
	err := r.pool.QueryRow(ctx,
		`UPDATE courses SET title = $2, description = $3, category = $4, difficulty = $5, thumbnail_url = $6
		 WHERE id = $1
		 RETURNING id, tenant_id, instructor_id, title, description, category, difficulty, status, thumbnail_url, created_at, updated_at`,
		id, title, desc, category, difficulty, thumbnailURL,
	).Scan(&c.ID, &c.TenantID, &c.InstructorID, &c.Title, &c.Description, &c.Category, &c.Difficulty, &c.Status, &c.ThumbnailURL, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, ErrCourseNotFound
	}
	return c, err
}

func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM courses WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrCourseNotFound
	}
	return nil
}

func (r *PostgresRepository) List(ctx context.Context, tenantID, category string, page, pageSize int) ([]CourseRecord, int, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * pageSize

	query := `SELECT COUNT(*) FROM courses WHERE tenant_id = $1`
	args := []any{tenantID}
	if category != "" {
		query += ` AND category = $2`
		args = append(args, category)
	}
	var total int
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	dataQuery := `SELECT id, tenant_id, instructor_id, title, description, category, difficulty, status, thumbnail_url, created_at, updated_at
		FROM courses WHERE tenant_id = $1`
	dataArgs := []any{tenantID}
	paramIdx := 2
	if category != "" {
		dataQuery += ` AND category = $` + pgParamStr(paramIdx)
		dataArgs = append(dataArgs, category)
		paramIdx++
	}
	dataQuery += ` ORDER BY created_at DESC LIMIT $` + pgParamStr(paramIdx) + ` OFFSET $` + pgParamStr(paramIdx+1)
	dataArgs = append(dataArgs, pageSize, offset)

	rows, err := r.pool.Query(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var courses []CourseRecord
	for rows.Next() {
		var c CourseRecord
		if err := rows.Scan(&c.ID, &c.TenantID, &c.InstructorID, &c.Title, &c.Description, &c.Category, &c.Difficulty, &c.Status, &c.ThumbnailURL, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, 0, err
		}
		courses = append(courses, c)
	}
	return courses, total, rows.Err()
}

func (r *PostgresRepository) Publish(ctx context.Context, id string) (CourseRecord, error) {
	var c CourseRecord
	err := r.pool.QueryRow(ctx,
		`UPDATE courses SET status = 'published' WHERE id = $1 AND status = 'draft'
		 RETURNING id, tenant_id, instructor_id, title, description, category, difficulty, status, thumbnail_url, created_at, updated_at`, id,
	).Scan(&c.ID, &c.TenantID, &c.InstructorID, &c.Title, &c.Description, &c.Category, &c.Difficulty, &c.Status, &c.ThumbnailURL, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, ErrCourseNotFound
	}
	return c, err
}

func (r *PostgresRepository) AddModule(ctx context.Context, courseID, title string, position int) (ModuleRecord, error) {
	var m ModuleRecord
	err := r.pool.QueryRow(ctx,
		`INSERT INTO modules (course_id, title, position) VALUES ($1, $2, $3)
		 RETURNING id, course_id, title, position`, courseID, title, position,
	).Scan(&m.ID, &m.CourseID, &m.Title, &m.Position)
	return m, err
}

func (r *PostgresRepository) AddLesson(ctx context.Context, courseID, moduleID, title, lessonType, contentID string, position int) (LessonRecord, error) {
	var l LessonRecord
	err := r.pool.QueryRow(ctx,
		`INSERT INTO lessons (module_id, title, type, content_id, position) VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, module_id, title, type, content_id, position`, moduleID, title, lessonType, contentID, position,
	).Scan(&l.ID, &l.ModuleID, &l.Title, &l.Type, &l.ContentID, &l.Position)
	return l, err
}

func (r *PostgresRepository) GetModules(ctx context.Context, courseID string) ([]ModuleRecord, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, course_id, title, position FROM modules WHERE course_id = $1 ORDER BY position`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var mods []ModuleRecord
	for rows.Next() {
		var m ModuleRecord
		if err := rows.Scan(&m.ID, &m.CourseID, &m.Title, &m.Position); err != nil {
			return nil, err
		}
		mods = append(mods, m)
	}
	return mods, rows.Err()
}

func (r *PostgresRepository) GetLessons(ctx context.Context, moduleID string) ([]LessonRecord, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, module_id, title, type, content_id, position FROM lessons WHERE module_id = $1 ORDER BY position`, moduleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var lessons []LessonRecord
	for rows.Next() {
		var l LessonRecord
		if err := rows.Scan(&l.ID, &l.ModuleID, &l.Title, &l.Type, &l.ContentID, &l.Position); err != nil {
			return nil, err
		}
		lessons = append(lessons, l)
	}
	return lessons, rows.Err()
}

func pgParamStr(n int) string {
	return strconv.Itoa(n)
}
