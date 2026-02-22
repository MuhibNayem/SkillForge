package certification

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	certificationpb "github.com/amnayem/skillforge/shared/pb/certification"
	"github.com/amnayem/skillforge/shared/pkg/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Handler implements the CertificationService gRPC server.
type Handler struct {
	certificationpb.UnimplementedCertificationServiceServer
	repo     Repository
	minioURL string // base URL for generating PDF download links
}

func NewHandler(repo Repository, minioURL string) *Handler {
	return &Handler{repo: repo, minioURL: minioURL}
}

// CreateTemplate creates a new certificate template for a tenant.
func (h *Handler) CreateTemplate(ctx context.Context, req *certificationpb.CreateTemplateRequest) (*certificationpb.CertificateTemplate, error) {
	tenantID, err := auth.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "unauthenticated")
	}

	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	t, err := h.repo.CreateTemplate(ctx, tenantID, req.Name, req.BackgroundUrl)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create template: %v", err)
	}

	return &certificationpb.CertificateTemplate{
		Id:        t.ID,
		TenantId:  t.TenantID,
		Name:      t.Name,
		CreatedAt: timestamppb.New(t.CreatedAt),
	}, nil
}

// GetUserCertificates lists all certificates for a user within a tenant.
func (h *Handler) GetUserCertificates(ctx context.Context, req *certificationpb.GetUserCertificatesRequest) (*certificationpb.GetUserCertificatesResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	tenantID := req.TenantId
	if tenantID == "" {
		// Fall back to JWT claim
		var ctxErr error
		tenantID, ctxErr = auth.GetTenantID(ctx)
		if ctxErr != nil {
			return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
		}
	}

	certs, err := h.repo.ListUserCertificates(ctx, req.UserId, tenantID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list certificates: %v", err)
	}

	resp := &certificationpb.GetUserCertificatesResponse{}
	for _, c := range certs {
		resp.Certificates = append(resp.Certificates, certToProto(c))
	}
	return resp, nil
}

// VerifyCertificate is a public endpoint that validates a certificate by its verification code.
func (h *Handler) VerifyCertificate(ctx context.Context, req *certificationpb.VerifyCertificateRequest) (*certificationpb.VerifyCertificateResponse, error) {
	if req.VerificationCode == "" {
		return nil, status.Error(codes.InvalidArgument, "verification_code is required")
	}

	c, err := h.repo.GetCertificateByVerificationCode(ctx, req.VerificationCode)
	if err != nil {
		if errors.Is(err, ErrCertificateNotFound) {
			return &certificationpb.VerifyCertificateResponse{IsValid: false}, nil
		}
		return nil, status.Errorf(codes.Internal, "verify certificate: %v", err)
	}

	return &certificationpb.VerifyCertificateResponse{
		IsValid:     true,
		Certificate: certToProto(c),
	}, nil
}

// IssueCertificate is an internal method triggered by course.completed events (called from Kafka consumer in Phase 2).
// It is exposed here as a public method so the service can be driven by other services.
func (h *Handler) IssueCertificate(ctx context.Context, tenantID, studentID, courseID, templateID string) (*certificationpb.Certificate, error) {
	// Generate a cryptographically random verification code
	codeBytes := make([]byte, 16)
	if _, err := rand.Read(codeBytes); err != nil {
		return nil, fmt.Errorf("generate verification code: %w", err)
	}
	verificationCode := hex.EncodeToString(codeBytes)

	pdfURL := fmt.Sprintf("%s/certificates/%s/%s/%s.pdf", h.minioURL, tenantID, courseID, verificationCode)

	c, err := h.repo.IssueCertificate(ctx, tenantID, studentID, courseID, templateID, verificationCode, pdfURL)
	if err != nil {
		return nil, fmt.Errorf("issue certificate: %w", err)
	}
	return certToProto(c), nil
}

// certToProto converts a CertificateRecord to its protobuf representation.
func certToProto(c CertificateRecord) *certificationpb.Certificate {
	return &certificationpb.Certificate{
		Id:               c.ID,
		TenantId:         c.TenantID,
		UserId:           c.StudentID,
		CourseId:         c.CourseID,
		VerificationCode: c.VerificationCode,
		PdfUrl:           c.PDFURL,
		IssuedAt:         timestamppb.New(c.IssuedAt),
	}
}
