package tenant_test

import (
	"context"
	"testing"
	"time"

	tenant "github.com/amnayem/skillforge/services/tenant"
	"github.com/amnayem/skillforge/shared/pb/tenantpb"
	"github.com/amnayem/skillforge/shared/pkg/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func newAuthCtx(role, userID, tenantID string) context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, auth.RoleKey, role)
	ctx = context.WithValue(ctx, auth.UserIDKey, userID)
	ctx = context.WithValue(ctx, auth.TenantIDKey, tenantID)
	return ctx
}

type mockTenantRepo struct {
	records map[string]tenant.TenantRecord
	nextID  int
}

func newMockTenantRepo() *mockTenantRepo {
	return &mockTenantRepo{records: make(map[string]tenant.TenantRecord)}
}

func (m *mockTenantRepo) Create(_ context.Context, name, domain, adminEmail string) (tenant.TenantRecord, error) {
	m.nextID++
	t := tenant.TenantRecord{
		ID:        "tenant-id-" + string(rune('0'+m.nextID)),
		Name:      name,
		Domain:    domain,
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_ = adminEmail
	m.records[t.ID] = t
	return t, nil
}

func (m *mockTenantRepo) GetByID(_ context.Context, id string) (tenant.TenantRecord, error) {
	t, ok := m.records[id]
	if !ok {
		return t, tenant.ErrTenantNotFound
	}
	return t, nil
}

func (m *mockTenantRepo) UpdateSettings(_ context.Context, id, theme, logoURL string) (tenant.TenantRecord, error) {
	t, ok := m.records[id]
	if !ok {
		return t, tenant.ErrTenantNotFound
	}
	t.Theme = theme
	t.LogoURL = logoURL
	t.UpdatedAt = time.Now()
	m.records[id] = t
	return t, nil
}

func TestCreateTenant(t *testing.T) {
	h := tenant.NewHandler(newMockTenantRepo())
	ctx := newAuthCtx("super_admin", "admin-1", "")
	resp, err := h.CreateTenant(ctx, &tenantpb.CreateTenantRequest{
		Name: "Acme Corp", Domain: "acme.skillforge.io", AdminEmail: "admin@acme.com",
	})
	if err != nil {
		t.Fatalf("CreateTenant failed: %v", err)
	}
	if resp.TenantId == "" {
		t.Error("expected non-empty tenant ID")
	}
	if resp.Name != "Acme Corp" {
		t.Errorf("expected 'Acme Corp', got %q", resp.Name)
	}
	if resp.Status != "active" {
		t.Errorf("expected status 'active', got %q", resp.Status)
	}
}

func TestCreateTenant_MissingName(t *testing.T) {
	h := tenant.NewHandler(newMockTenantRepo())
	_, err := h.CreateTenant(newAuthCtx("super_admin", "admin-1", ""), &tenantpb.CreateTenantRequest{Domain: "x.io"})
	if err == nil {
		t.Fatal("expected error for missing name")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", status.Code(err))
	}
}

func TestGetTenant(t *testing.T) {
	h := tenant.NewHandler(newMockTenantRepo())
	created, _ := h.CreateTenant(newAuthCtx("super_admin", "admin-1", ""), &tenantpb.CreateTenantRequest{
		Name: "Beta Inc", Domain: "beta.io",
	})
	resp, err := h.GetTenant(context.Background(), &tenantpb.GetTenantRequest{TenantId: created.TenantId})
	if err != nil {
		t.Fatalf("GetTenant failed: %v", err)
	}
	if resp.Name != "Beta Inc" {
		t.Errorf("expected 'Beta Inc', got %q", resp.Name)
	}
}

func TestGetTenant_NotFound(t *testing.T) {
	h := tenant.NewHandler(newMockTenantRepo())
	_, err := h.GetTenant(context.Background(), &tenantpb.GetTenantRequest{TenantId: "nope"})
	if err == nil {
		t.Fatal("expected error")
	}
	if status.Code(err) != codes.NotFound {
		t.Errorf("expected NotFound, got %v", status.Code(err))
	}
}

func TestGetTenant_MissingID(t *testing.T) {
	h := tenant.NewHandler(newMockTenantRepo())
	_, err := h.GetTenant(context.Background(), &tenantpb.GetTenantRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", status.Code(err))
	}
}

func TestUpdateTenantSettings(t *testing.T) {
	h := tenant.NewHandler(newMockTenantRepo())
	created, _ := h.CreateTenant(newAuthCtx("super_admin", "admin-1", ""), &tenantpb.CreateTenantRequest{
		Name: "Gamma LLC", Domain: "gamma.io",
	})
	resp, err := h.UpdateTenantSettings(newAuthCtx("tenant_admin", "admin-1", created.TenantId), &tenantpb.UpdateTenantSettingsRequest{
		TenantId: created.TenantId, Theme: "dark", LogoUrl: "http://minio:9000/logo.png",
	})
	if err != nil {
		t.Fatalf("UpdateTenantSettings failed: %v", err)
	}
	if resp.Theme != "dark" {
		t.Errorf("expected 'dark', got %q", resp.Theme)
	}
}

func TestUpdateTenantSettings_NotFound(t *testing.T) {
	h := tenant.NewHandler(newMockTenantRepo())
	_, err := h.UpdateTenantSettings(newAuthCtx("tenant_admin", "admin-1", "nope"), &tenantpb.UpdateTenantSettingsRequest{
		TenantId: "nope", Theme: "light",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if status.Code(err) != codes.NotFound {
		t.Errorf("expected NotFound, got %v", status.Code(err))
	}
}

func TestUpdateTenantSettings_MissingID(t *testing.T) {
	h := tenant.NewHandler(newMockTenantRepo())
	_, err := h.UpdateTenantSettings(newAuthCtx("tenant_admin", "admin-1", ""), &tenantpb.UpdateTenantSettingsRequest{Theme: "light"})
	if err == nil {
		t.Fatal("expected error")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", status.Code(err))
	}
}
