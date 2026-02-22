package analytics

import (
	"context"

	analyticspb "github.com/amnayem/skillforge/shared/pb/analytics"
	"github.com/amnayem/skillforge/shared/pkg/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Handler implements analyticspb.AnalyticsServiceServer.
type Handler struct {
	analyticspb.UnimplementedAnalyticsServiceServer
	repo Repository
}

func NewHandler(repo Repository) *Handler {
	return &Handler{repo: repo}
}

// GetCourseAnalytics returns enrollment/completion metrics for a course.
// Accessible by instructors and admins.
func (h *Handler) GetCourseAnalytics(ctx context.Context, req *analyticspb.CourseAnalyticsRequest) (*analyticspb.CourseAnalyticsResponse, error) {
	if req.CourseId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "course_id is required")
	}
	if err := auth.RequireRole(ctx, "instructor", "tenant_admin", "super_admin"); err != nil {
		return nil, err
	}

	stats, err := h.repo.GetCourseAnalytics(ctx, req.CourseId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get course analytics: %v", err)
	}
	return &analyticspb.CourseAnalyticsResponse{
		TotalEnrollments: stats.TotalEnrollments,
		ActiveStudents:   stats.ActiveStudents,
		Completions:      stats.Completions,
		AverageProgress:  stats.AverageProgress,
	}, nil
}

// GetUserAnalytics returns learning stats for a specific user.
// Users can view their own stats; admins can view any user's stats.
func (h *Handler) GetUserAnalytics(ctx context.Context, req *analyticspb.UserAnalyticsRequest) (*analyticspb.UserAnalyticsResponse, error) {
	if req.UserId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "user_id is required")
	}
	callerID, err := auth.GetUserID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}
	if req.UserId != callerID {
		if err := auth.RequireRole(ctx, "tenant_admin", "super_admin"); err != nil {
			return nil, err
		}
	}

	stats, err := h.repo.GetUserAnalytics(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get user analytics: %v", err)
	}
	return &analyticspb.UserAnalyticsResponse{
		CoursesEnrolled:    stats.CoursesEnrolled,
		CoursesCompleted:   stats.CoursesCompleted,
		LearningStreakDays: stats.LearningStreakDays,
		TotalLearningHours: stats.TotalLearningHours,
	}, nil
}

// GetTenantDashboard returns tenant-level aggregate metrics.
// Restricted to tenant_admin and super_admin.
func (h *Handler) GetTenantDashboard(ctx context.Context, req *analyticspb.TenantDashboardRequest) (*analyticspb.TenantDashboardResponse, error) {
	if req.TenantId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "tenant_id is required")
	}
	if err := auth.RequireRole(ctx, "tenant_admin", "super_admin"); err != nil {
		return nil, err
	}

	stats, err := h.repo.GetTenantDashboard(ctx, req.TenantId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get tenant dashboard: %v", err)
	}
	return &analyticspb.TenantDashboardResponse{
		TotalUsers:             stats.TotalUsers,
		TotalCourses:           stats.TotalCourses,
		TotalEnrollments:       stats.TotalEnrollments,
		ActiveUsersLast_30Days: stats.ActiveUsersLast30,
	}, nil
}
