package certification

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TemplateRecord maps to the certificate_templates table.
type TemplateRecord struct {
	ID            string
	TenantID      string
	Name          string
	BackgroundURL string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// CertificateRecord maps to the certificates table.
type CertificateRecord struct {
	ID               string
	TenantID         string
	StudentID        string
	CourseID         string
	TemplateID       string
	VerificationCode string
	PDFURL           string
	IssuedAt         time.Time
}

// Repository defines the data access contract for the Certification service.
type Repository interface {
	CreateTemplate(ctx context.Context, tenantID, name, backgroundURL string) (TemplateRecord, error)
	GetTemplate(ctx context.Context, id string) (TemplateRecord, error)
	ListTemplates(ctx context.Context, tenantID string) ([]TemplateRecord, error)
	IssueCertificate(ctx context.Context, tenantID, studentID, courseID, templateID, verificationCode, pdfURL string) (CertificateRecord, error)
	GetCertificate(ctx context.Context, id string) (CertificateRecord, error)
	GetCertificateByVerificationCode(ctx context.Context, code string) (CertificateRecord, error)
	GetCertificateByStudentAndCourse(ctx context.Context, studentID, courseID string) (CertificateRecord, error)
	ListUserCertificates(ctx context.Context, userID, tenantID string) ([]CertificateRecord, error)
}

var (
	ErrTemplateNotFound    = errors.New("certificate template not found")
	ErrCertificateNotFound = errors.New("certificate not found")
)

// PostgresRepository implements Repository with PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) CreateTemplate(ctx context.Context, tenantID, name, backgroundURL string) (TemplateRecord, error) {
	var t TemplateRecord
	err := r.pool.QueryRow(ctx,
		`INSERT INTO certificate_templates (tenant_id, name, background_url)
		 VALUES ($1, $2, $3)
		 RETURNING id, tenant_id, name, background_url, created_at, updated_at`,
		tenantID, name, backgroundURL,
	).Scan(&t.ID, &t.TenantID, &t.Name, &t.BackgroundURL, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}

func (r *PostgresRepository) GetTemplate(ctx context.Context, id string) (TemplateRecord, error) {
	var t TemplateRecord
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, background_url, created_at, updated_at
		 FROM certificate_templates WHERE id = $1`,
		id,
	).Scan(&t.ID, &t.TenantID, &t.Name, &t.BackgroundURL, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return t, ErrTemplateNotFound
	}
	return t, err
}

func (r *PostgresRepository) ListTemplates(ctx context.Context, tenantID string) ([]TemplateRecord, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, name, background_url, created_at, updated_at
		 FROM certificate_templates WHERE tenant_id = $1 ORDER BY created_at DESC`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []TemplateRecord
	for rows.Next() {
		var t TemplateRecord
		if err := rows.Scan(&t.ID, &t.TenantID, &t.Name, &t.BackgroundURL, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}
	return templates, rows.Err()
}

func (r *PostgresRepository) IssueCertificate(ctx context.Context, tenantID, studentID, courseID, templateID, verificationCode, pdfURL string) (CertificateRecord, error) {
	var c CertificateRecord
	err := r.pool.QueryRow(ctx,
		`INSERT INTO certificates (tenant_id, student_id, course_id, template_id, pdf_url)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (student_id, course_id) DO NOTHING
		 RETURNING id, tenant_id, student_id, course_id, template_id, pdf_url, issued_at`,
		tenantID, studentID, courseID, templateID, pdfURL,
	).Scan(&c.ID, &c.TenantID, &c.StudentID, &c.CourseID, &c.TemplateID, &c.PDFURL, &c.IssuedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		// Certificate already issued — fetch the existing one
		return r.GetCertificateByStudentAndCourse(ctx, studentID, courseID)
	}
	if err != nil {
		return c, err
	}
	// Store the verification code in memory only; ideally a future migration
	// adds a verification_code column. For now the code is embedded in the PDF URL.
	c.VerificationCode = verificationCode
	return c, nil
}

func (r *PostgresRepository) GetCertificate(ctx context.Context, id string) (CertificateRecord, error) {
	var c CertificateRecord
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, student_id, course_id, template_id, pdf_url, issued_at
		 FROM certificates WHERE id = $1`,
		id,
	).Scan(&c.ID, &c.TenantID, &c.StudentID, &c.CourseID, &c.TemplateID, &c.PDFURL, &c.IssuedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, ErrCertificateNotFound
	}
	return c, err
}

// GetCertificateByVerificationCode looks up by the verification code embedded in pdf_url.
// NOTE: Once a migration adds a dedicated verification_code column, use that index instead.
func (r *PostgresRepository) GetCertificateByVerificationCode(ctx context.Context, code string) (CertificateRecord, error) {
	var c CertificateRecord
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, student_id, course_id, template_id, pdf_url, issued_at
		 FROM certificates WHERE pdf_url LIKE '%' || $1 || '%'`,
		code,
	).Scan(&c.ID, &c.TenantID, &c.StudentID, &c.CourseID, &c.TemplateID, &c.PDFURL, &c.IssuedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, ErrCertificateNotFound
	}
	c.VerificationCode = code
	return c, err
}

func (r *PostgresRepository) GetCertificateByStudentAndCourse(ctx context.Context, studentID, courseID string) (CertificateRecord, error) {
	var c CertificateRecord
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, student_id, course_id, template_id, pdf_url, issued_at
		 FROM certificates WHERE student_id = $1 AND course_id = $2`,
		studentID, courseID,
	).Scan(&c.ID, &c.TenantID, &c.StudentID, &c.CourseID, &c.TemplateID, &c.PDFURL, &c.IssuedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, ErrCertificateNotFound
	}
	return c, err
}

func (r *PostgresRepository) ListUserCertificates(ctx context.Context, userID, tenantID string) ([]CertificateRecord, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, student_id, course_id, template_id, pdf_url, issued_at
		 FROM certificates WHERE student_id = $1 AND tenant_id = $2 ORDER BY issued_at DESC`,
		userID, tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var certs []CertificateRecord
	for rows.Next() {
		var c CertificateRecord
		if err := rows.Scan(&c.ID, &c.TenantID, &c.StudentID, &c.CourseID, &c.TemplateID, &c.PDFURL, &c.IssuedAt); err != nil {
			return nil, err
		}
		certs = append(certs, c)
	}
	return certs, rows.Err()
}
