package notification

import (
	"context"
	"errors"
	"testing"
	"time"

	notificationpb "github.com/amnayem/skillforge/shared/pb/notification"
	"github.com/amnayem/skillforge/shared/pkg/auth"
)

type mockRepo struct {
	prefs      map[string]PreferencesRecord
	deliveries []DeliveryRecord
}

func newMockRepo() *mockRepo {
	return &mockRepo{prefs: make(map[string]PreferencesRecord)}
}

func (m *mockRepo) UpsertPreferences(_ context.Context, p PreferencesRecord) (PreferencesRecord, error) {
	p.UpdatedAt = time.Now()
	m.prefs[p.UserID] = p
	return p, nil
}

func (m *mockRepo) GetPreferences(_ context.Context, userID string) (PreferencesRecord, error) {
	p, ok := m.prefs[userID]
	if !ok {
		return p, ErrPreferencesNotFound
	}
	return p, nil
}

func (m *mockRepo) RecordDelivery(_ context.Context, d DeliveryRecord) error {
	m.deliveries = append(m.deliveries, d)
	return nil
}

type errRepo struct{}

func (e *errRepo) UpsertPreferences(_ context.Context, _ PreferencesRecord) (PreferencesRecord, error) {
	return PreferencesRecord{}, errors.New("db error")
}
func (e *errRepo) GetPreferences(_ context.Context, _ string) (PreferencesRecord, error) {
	return PreferencesRecord{}, errors.New("db error")
}
func (e *errRepo) RecordDelivery(_ context.Context, _ DeliveryRecord) error {
	return errors.New("db error")
}

func newAuthCtx(role, userID, tenantID string) context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, auth.RoleKey, role)
	ctx = context.WithValue(ctx, auth.UserIDKey, userID)
	ctx = context.WithValue(ctx, auth.TenantIDKey, tenantID)
	return ctx
}

func TestUpdatePreferences_OwnUser(t *testing.T) {
	h := NewHandler(newMockRepo())
	ctx := newAuthCtx("student", "user-1", "tenant-1")
	resp, err := h.UpdatePreferences(ctx, &notificationpb.UpdatePreferencesRequest{
		UserId:             "user-1",
		EmailCourseUpdates: true,
		EmailMarketing:     false,
		InAppMentions:      true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Preferences.EmailCourseUpdates {
		t.Error("expected EmailCourseUpdates=true")
	}
	if resp.Preferences.EmailMarketing {
		t.Error("expected EmailMarketing=false")
	}
}

func TestUpdatePreferences_AdminCanUpdateOthers(t *testing.T) {
	h := NewHandler(newMockRepo())
	ctx := newAuthCtx("tenant_admin", "admin-1", "tenant-1")
	resp, err := h.UpdatePreferences(ctx, &notificationpb.UpdatePreferencesRequest{
		UserId:         "other-user",
		EmailMarketing: true,
	})
	if err != nil {
		t.Fatalf("admin should be able to update other user: %v", err)
	}
	if !resp.Preferences.EmailMarketing {
		t.Error("expected EmailMarketing=true")
	}
}

func TestUpdatePreferences_StudentCannotUpdateOthers(t *testing.T) {
	h := NewHandler(newMockRepo())
	ctx := newAuthCtx("student", "user-1", "tenant-1")
	_, err := h.UpdatePreferences(ctx, &notificationpb.UpdatePreferencesRequest{
		UserId: "other-user",
	})
	if err == nil {
		t.Fatal("expected permission denied error")
	}
}

func TestGetPreferences_NotFound_ReturnsDefaults(t *testing.T) {
	h := NewHandler(newMockRepo())
	ctx := newAuthCtx("student", "user-1", "tenant-1")
	resp, err := h.GetPreferences(ctx, &notificationpb.GetPreferencesRequest{UserId: "user-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Preferences.EmailCourseUpdates {
		t.Error("expected default EmailCourseUpdates=true")
	}
}

func TestGetPreferences_Unauthenticated(t *testing.T) {
	h := NewHandler(newMockRepo())
	_, err := h.GetPreferences(context.Background(), &notificationpb.GetPreferencesRequest{UserId: "user-1"})
	if err == nil {
		t.Fatal("expected unauthenticated error")
	}
}

func TestGetPreferences_DBError(t *testing.T) {
	h := NewHandler(&errRepo{})
	ctx := newAuthCtx("student", "user-1", "tenant-1")
	_, err := h.GetPreferences(ctx, &notificationpb.GetPreferencesRequest{UserId: "user-1"})
	if err == nil {
		t.Fatal("expected internal error")
	}
}
