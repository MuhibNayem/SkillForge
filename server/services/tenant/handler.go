package tenant

import (
	"context"
	"errors"

	"github.com/amnayem/skillforge/shared/pb/tenantpb"
	"github.com/amnayem/skillforge/shared/pkg/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	tenantpb.UnimplementedTenantServiceServer
	repo Repository
}

func NewHandler(repo Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) CreateTenant(ctx context.Context, req *tenantpb.CreateTenantRequest) (*tenantpb.TenantResponse, error) {
	if err := auth.RequireRole(ctx, "super_admin"); err != nil {
		return nil, err
	}
	if req.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument, "name is required")
	}
	t, err := h.repo.Create(ctx, req.Name, req.Domain, req.AdminEmail)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create tenant: %v", err)
	}
	return tenantToProto(t), nil
}

func (h *Handler) GetTenant(ctx context.Context, req *tenantpb.GetTenantRequest) (*tenantpb.TenantResponse, error) {
	if req.TenantId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "tenant_id is required")
	}
	t, err := h.repo.GetByID(ctx, req.TenantId)
	if err != nil {
		if errors.Is(err, ErrTenantNotFound) {
			return nil, status.Errorf(codes.NotFound, "tenant not found")
		}
		return nil, status.Errorf(codes.Internal, "get tenant: %v", err)
	}
	return tenantToProto(t), nil
}

func (h *Handler) UpdateTenantSettings(ctx context.Context, req *tenantpb.UpdateTenantSettingsRequest) (*tenantpb.TenantResponse, error) {
	if err := auth.RequireRole(ctx, "tenant_admin", "super_admin"); err != nil {
		return nil, err
	}
	if req.TenantId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "tenant_id is required")
	}
	t, err := h.repo.UpdateSettings(ctx, req.TenantId, req.Theme, req.LogoUrl)
	if err != nil {
		if errors.Is(err, ErrTenantNotFound) {
			return nil, status.Errorf(codes.NotFound, "tenant not found")
		}
		return nil, status.Errorf(codes.Internal, "update tenant: %v", err)
	}
	return tenantToProto(t), nil
}

func tenantToProto(t TenantRecord) *tenantpb.TenantResponse {
	return &tenantpb.TenantResponse{
		TenantId:  t.ID,
		Name:      t.Name,
		Domain:    t.Domain,
		Status:    t.Status,
		Theme:     t.Theme,
		LogoUrl:   t.LogoURL,
		CreatedAt: t.CreatedAt.Unix(),
	}
}
