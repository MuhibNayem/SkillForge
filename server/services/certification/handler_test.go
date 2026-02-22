package certification_test

import (
	"context"
	"testing"
	"time"

	certification "github.com/amnayem/skillforge/services/certification"
	certificationpb "github.com/amnayem/skillforge/shared/pb/certification"
	"github.com/amnayem/skillforge/shared/pkg/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockCertRepo struct {
	templates    map[string]certification.TemplateRecord
	certificates map[string]certification.CertificateRecord
	byCourse     map[string]string
	byCode       map[string]string
}

func newMockCertRepo() *mockCertRepo {
	return &mockCertRepo{
		templates:    make(map[string]certification.TemplateRecord),
		certificates: make(map[string]certification.CertificateRecord),
		byCourse:     make(map[string]string),
		byCode:       make(map[string]string),
	}
}

func (m *mockCertRepo) CreateTemplate(_ context.Context, tenantID, name, backgroundURL string) (certification.TemplateRecord, error) {
	t := certification.TemplateRecord{
		ID: "tpl-" + name, TenantID: tenantID, Name: name, BackgroundURL: backgroundURL,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	m.templates[t.ID] = t
	return t, nil
}

func (m *mockCertRepo) GetTemplate(_ context.Context, id string) (certification.TemplateRecord, error) {
	t, ok := m.templates[id]
	if !ok {
		return t, certification.ErrTemplateNotFound
	}
	return t, nil
}

func (m *mockCertRepo) ListTemplates(_ context.Context, tenantID string) ([]certification.TemplateRecord, error) {
	var result []certification.TemplateRecord
	for _, t := range m.templates {
		if t.TenantID == tenantID {
			result = append(result, t)
		}
	}
	return result, nil
}

func (m *mockCertRepo) IssueCertificate(_ context.Context, tenantID, studentID, courseID, templateID, verificationCode, pdfURL string) (certification.CertificateRecord, error) {
	key := studentID + ":" + courseID
	if id, exists := m.byCourse[key]; exists {
		return m.certificates[id], nil
	}
	c := certification.CertificateRecord{
		ID: "cert-" + courseID, TenantID: tenantID, StudentID: studentID, CourseID: courseID,
		TemplateID: templateID, VerificationCode: verificationCode, PDFURL: pdfURL, IssuedAt: time.Now(),
	}
	m.certificates[c.ID] = c
	m.byCourse[key] = c.ID
	m.byCode[verificationCode] = c.ID
	return c, nil
}

func (m *mockCertRepo) GetCertificate(_ context.Context, id string) (certification.CertificateRecord, error) {
	c, ok := m.certificates[id]
	if !ok {
		return c, certification.ErrCertificateNotFound
	}
	return c, nil
}

func (m *mockCertRepo) GetCertificateByVerificationCode(_ context.Context, code string) (certification.CertificateRecord, error) {
	id, ok := m.byCode[code]
	if !ok {
		return certification.CertificateRecord{}, certification.ErrCertificateNotFound
	}
	c := m.certificates[id]
	c.VerificationCode = code
	return c, nil
}

func (m *mockCertRepo) GetCertificateByStudentAndCourse(_ context.Context, studentID, courseID string) (certification.CertificateRecord, error) {
	key := studentID + ":" + courseID
	id, ok := m.byCourse[key]
	if !ok {
		return certification.CertificateRecord{}, certification.ErrCertificateNotFound
	}
	return m.certificates[id], nil
}

func (m *mockCertRepo) ListUserCertificates(_ context.Context, userID, tenantID string) ([]certification.CertificateRecord, error) {
	var result []certification.CertificateRecord
	for _, c := range m.certificates {
		if c.StudentID == userID && c.TenantID == tenantID {
			result = append(result, c)
		}
	}
	return result, nil
}

func TestCreateTemplate(t *testing.T) {
	repo := newMockCertRepo()
	h := certification.NewHandler(repo, "http://minio:9000")
	ctx := context.WithValue(context.Background(), auth.TenantIDKey, "tenant-1")

	tpl, err := h.CreateTemplate(ctx, &certificationpb.CreateTemplateRequest{
		Name: "Completion Certificate", BackgroundUrl: "http://minio:9000/bg.png",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if tpl.Name != "Completion Certificate" {
		t.Errorf("expected correct name, got %q", tpl.Name)
	}
	if tpl.TenantId != "tenant-1" {
		t.Errorf("expected tenant-1, got %q", tpl.TenantId)
	}
}

func TestCreateTemplate_MissingName(t *testing.T) {
	repo := newMockCertRepo()
	h := certification.NewHandler(repo, "http://minio:9000")
	ctx := context.WithValue(context.Background(), auth.TenantIDKey, "tenant-1")

	_, err := h.CreateTemplate(ctx, &certificationpb.CreateTemplateRequest{Name: ""})
	if err == nil {
		t.Fatal("expected error for missing name")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", status.Code(err))
	}
}

func TestCreateTemplate_Unauthenticated(t *testing.T) {
	repo := newMockCertRepo()
	h := certification.NewHandler(repo, "http://minio:9000")

	_, err := h.CreateTemplate(context.Background(), &certificationpb.CreateTemplateRequest{Name: "Test"})
	if err == nil {
		t.Fatal("expected error for missing auth")
	}
	if status.Code(err) != codes.Unauthenticated {
		t.Errorf("expected Unauthenticated, got %v", status.Code(err))
	}
}

func TestGetUserCertificates(t *testing.T) {
	repo := newMockCertRepo()
	h := certification.NewHandler(repo, "http://minio:9000")
	ctx := context.WithValue(context.Background(), auth.TenantIDKey, "tenant-1")

	_, err := h.IssueCertificate(ctx, "tenant-1", "user-1", "course-1", "tpl-1")
	if err != nil {
		t.Fatalf("IssueCertificate failed: %v", err)
	}
	_, err = h.IssueCertificate(ctx, "tenant-1", "user-1", "course-2", "tpl-1")
	if err != nil {
		t.Fatalf("IssueCertificate failed: %v", err)
	}

	resp, err := h.GetUserCertificates(ctx, &certificationpb.GetUserCertificatesRequest{
		UserId: "user-1", TenantId: "tenant-1",
	})
	if err != nil {
		t.Fatalf("GetUserCertificates failed: %v", err)
	}
	if len(resp.Certificates) != 2 {
		t.Errorf("expected 2 certificates, got %d", len(resp.Certificates))
	}
}

func TestGetUserCertificates_MissingUserID(t *testing.T) {
	repo := newMockCertRepo()
	h := certification.NewHandler(repo, "http://minio:9000")
	ctx := context.WithValue(context.Background(), auth.TenantIDKey, "tenant-1")

	_, err := h.GetUserCertificates(ctx, &certificationpb.GetUserCertificatesRequest{})
	if err == nil {
		t.Fatal("expected error for missing user_id")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", status.Code(err))
	}
}

func TestVerifyCertificate_Valid(t *testing.T) {
	repo := newMockCertRepo()
	h := certification.NewHandler(repo, "http://minio:9000")
	ctx := context.WithValue(context.Background(), auth.TenantIDKey, "tenant-1")

	cert, err := h.IssueCertificate(ctx, "tenant-1", "user-1", "course-1", "tpl-1")
	if err != nil {
		t.Fatalf("IssueCertificate failed: %v", err)
	}

	resp, err := h.VerifyCertificate(ctx, &certificationpb.VerifyCertificateRequest{
		VerificationCode: cert.VerificationCode,
	})
	if err != nil {
		t.Fatalf("VerifyCertificate failed: %v", err)
	}
	if !resp.IsValid {
		t.Error("expected certificate to be valid")
	}
	if resp.Certificate == nil {
		t.Error("expected certificate in response")
	}
}

func TestVerifyCertificate_Invalid(t *testing.T) {
	repo := newMockCertRepo()
	h := certification.NewHandler(repo, "http://minio:9000")

	resp, err := h.VerifyCertificate(context.Background(), &certificationpb.VerifyCertificateRequest{
		VerificationCode: "nonexistent-code",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.IsValid {
		t.Error("expected invalid certificate")
	}
}

func TestVerifyCertificate_MissingCode(t *testing.T) {
	repo := newMockCertRepo()
	h := certification.NewHandler(repo, "http://minio:9000")

	_, err := h.VerifyCertificate(context.Background(), &certificationpb.VerifyCertificateRequest{})
	if err == nil {
		t.Fatal("expected error for missing code")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", status.Code(err))
	}
}

func TestIssueCertificate_Idempotent(t *testing.T) {
	repo := newMockCertRepo()
	h := certification.NewHandler(repo, "http://minio:9000")
	ctx := context.WithValue(context.Background(), auth.TenantIDKey, "tenant-1")

	c1, err := h.IssueCertificate(ctx, "tenant-1", "user-1", "course-1", "tpl-1")
	if err != nil {
		t.Fatalf("first IssueCertificate failed: %v", err)
	}
	c2, err := h.IssueCertificate(ctx, "tenant-1", "user-1", "course-1", "tpl-1")
	if err != nil {
		t.Fatalf("second IssueCertificate failed: %v", err)
	}
	if c1.Id != c2.Id {
		t.Errorf("expected idempotent result, got different IDs: %s vs %s", c1.Id, c2.Id)
	}
}
