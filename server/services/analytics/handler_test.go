package analytics

import (
	"context"
	"errors"
	"testing"

	analyticspb "github.com/amnayem/skillforge/shared/pb/analytics"
	"github.com/amnayem/skillforge/shared/pkg/auth"
)

// mockRepo implements Repository with in-memory data.
type mockRepo struct {
	courseStats CourseStats
	userStats   UserStats
	tenantStats TenantStats
	indexedDocs []map[string]interface{}
	err         error
}

func (m *mockRepo) GetCourseAnalytics(_ context.Context, _ string) (CourseStats, error) {
	return m.courseStats, m.err
}
func (m *mockRepo) GetUserAnalytics(_ context.Context, _ string) (UserStats, error) {
	return m.userStats, m.err
}
func (m *mockRepo) GetTenantDashboard(_ context.Context, _ string) (TenantStats, error) {
	return m.tenantStats, m.err
}
func (m *mockRepo) IndexEvent(_ context.Context, _ string, doc map[string]interface{}) error {
	m.indexedDocs = append(m.indexedDocs, doc)
	return m.err
}

func newAuthCtx(role, userID, tenantID string) context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, auth.RoleKey, role)
	ctx = context.WithValue(ctx, auth.UserIDKey, userID)
	ctx = context.WithValue(ctx, auth.TenantIDKey, tenantID)
	return ctx
}

func TestGetCourseAnalytics_Instructor(t *testing.T) {
	repo := &mockRepo{courseStats: CourseStats{TotalEnrollments: 50, Completions: 10}}
	h := NewHandler(repo)
	ctx := newAuthCtx("instructor", "user-1", "tenant-1")

	resp, err := h.GetCourseAnalytics(ctx, &analyticspb.CourseAnalyticsRequest{CourseId: "course-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.TotalEnrollments != 50 {
		t.Errorf("expected 50, got %d", resp.TotalEnrollments)
	}
}

func TestGetCourseAnalytics_StudentDenied(t *testing.T) {
	repo := &mockRepo{}
	h := NewHandler(repo)
	ctx := newAuthCtx("student", "user-1", "tenant-1")

	_, err := h.GetCourseAnalytics(ctx, &analyticspb.CourseAnalyticsRequest{CourseId: "course-1"})
	if err == nil {
		t.Fatal("expected permission denied")
	}
}

func TestGetCourseAnalytics_MissingID(t *testing.T) {
	h := NewHandler(&mockRepo{})
	ctx := newAuthCtx("instructor", "user-1", "tenant-1")
	_, err := h.GetCourseAnalytics(ctx, &analyticspb.CourseAnalyticsRequest{})
	if err == nil {
		t.Fatal("expected invalid argument error")
	}
}

func TestGetUserAnalytics_OwnUser(t *testing.T) {
	repo := &mockRepo{userStats: UserStats{CoursesEnrolled: 5, CoursesCompleted: 2}}
	h := NewHandler(repo)
	ctx := newAuthCtx("student", "user-1", "tenant-1")

	resp, err := h.GetUserAnalytics(ctx, &analyticspb.UserAnalyticsRequest{UserId: "user-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.CoursesEnrolled != 5 {
		t.Errorf("expected 5, got %d", resp.CoursesEnrolled)
	}
}

func TestGetUserAnalytics_OtherUserDenied(t *testing.T) {
	h := NewHandler(&mockRepo{})
	ctx := newAuthCtx("student", "user-1", "tenant-1")
	_, err := h.GetUserAnalytics(ctx, &analyticspb.UserAnalyticsRequest{UserId: "other-user"})
	if err == nil {
		t.Fatal("expected permission denied")
	}
}

func TestGetTenantDashboard_Admin(t *testing.T) {
	repo := &mockRepo{tenantStats: TenantStats{TotalUsers: 100, TotalCourses: 20}}
	h := NewHandler(repo)
	ctx := newAuthCtx("tenant_admin", "admin-1", "tenant-1")

	resp, err := h.GetTenantDashboard(ctx, &analyticspb.TenantDashboardRequest{TenantId: "tenant-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.TotalUsers != 100 {
		t.Errorf("expected 100, got %d", resp.TotalUsers)
	}
}

func TestGetTenantDashboard_StudentDenied(t *testing.T) {
	h := NewHandler(&mockRepo{})
	ctx := newAuthCtx("student", "user-1", "tenant-1")
	_, err := h.GetTenantDashboard(ctx, &analyticspb.TenantDashboardRequest{TenantId: "tenant-1"})
	if err == nil {
		t.Fatal("expected permission denied")
	}
}

func TestGetTenantDashboard_RepoError(t *testing.T) {
	h := NewHandler(&mockRepo{err: errors.New("es down")})
	ctx := newAuthCtx("tenant_admin", "admin-1", "tenant-1")
	_, err := h.GetTenantDashboard(ctx, &analyticspb.TenantDashboardRequest{TenantId: "tenant-1"})
	if err == nil {
		t.Fatal("expected internal error")
	}
}
