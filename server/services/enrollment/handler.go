package enrollment

import (
	"context"
	"errors"

	"github.com/amnayem/skillforge/shared/pb/enrollmentpb"
	"github.com/amnayem/skillforge/shared/pkg/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	enrollmentpb.UnimplementedEnrollmentServiceServer
	repo Repository
}

func NewHandler(repo Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) Enroll(ctx context.Context, req *enrollmentpb.EnrollRequest) (*enrollmentpb.EnrollmentResponse, error) {
	if req.CourseId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "course_id is required")
	}
	// Always use the authenticated caller's identity — never trust the request body for user/tenant ID
	userID, err := auth.GetUserID(ctx)
	if err != nil {
		return nil, err
	}
	tenantID, err := auth.GetTenantID(ctx)
	if err != nil {
		return nil, err
	}
	e, err := h.repo.Enroll(ctx, tenantID, userID, req.CourseId)
	if err != nil {
		if errors.Is(err, ErrAlreadyEnrolled) {
			return nil, status.Errorf(codes.AlreadyExists, "already enrolled")
		}
		return nil, status.Errorf(codes.Internal, "enroll: %v", err)
	}
	return enrollmentToProto(e), nil
}

func (h *Handler) GetEnrollment(ctx context.Context, req *enrollmentpb.GetEnrollmentRequest) (*enrollmentpb.EnrollmentResponse, error) {
	if req.EnrollmentId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "enrollment_id is required")
	}
	e, err := h.repo.GetByID(ctx, req.EnrollmentId)
	if err != nil {
		if errors.Is(err, ErrEnrollmentNotFound) {
			return nil, status.Errorf(codes.NotFound, "enrollment not found")
		}
		return nil, status.Errorf(codes.Internal, "get enrollment: %v", err)
	}
	return enrollmentToProto(e), nil
}

func (h *Handler) ListEnrollments(ctx context.Context, req *enrollmentpb.ListEnrollmentsRequest) (*enrollmentpb.ListEnrollmentsResponse, error) {
	// Students always see only their own enrollments; admins may pass an explicit user_id to view others'
	callerID, _ := auth.GetUserID(ctx)
	callerRole, _ := auth.GetRole(ctx)
	callerTenantID, _ := auth.GetTenantID(ctx)

	userID := callerID
	if req.UserId != "" && req.UserId != callerID {
		// Only admins may query another user's enrollments
		if callerRole != "tenant_admin" && callerRole != "super_admin" {
			return nil, status.Error(codes.PermissionDenied, "cannot list another user's enrollments")
		}
		userID = req.UserId
	}
	tenantID := callerTenantID
	if req.TenantId != "" {
		tenantID = req.TenantId
	}
	enrollments, total, err := h.repo.List(ctx, userID, tenantID, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list enrollments: %v", err)
	}
	resp := &enrollmentpb.ListEnrollmentsResponse{Total: int32(total)}
	for _, e := range enrollments {
		resp.Enrollments = append(resp.Enrollments, enrollmentToProto(e))
	}
	return resp, nil
}

func (h *Handler) UpdateProgress(ctx context.Context, req *enrollmentpb.UpdateProgressRequest) (*enrollmentpb.ProgressResponse, error) {
	if req.EnrollmentId == "" || req.LessonId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "enrollment_id and lesson_id are required")
	}
	if err := h.repo.UpdateProgress(ctx, req.EnrollmentId, req.LessonId, int(req.PercentComplete)); err != nil {
		return nil, status.Errorf(codes.Internal, "update progress: %v", err)
	}

	// Recalculate overall
	overall, err := h.repo.RecalculateOverallProgress(ctx, req.EnrollmentId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "recalculate progress: %v", err)
	}

	progress, _ := h.repo.GetProgress(ctx, req.EnrollmentId)
	return &enrollmentpb.ProgressResponse{
		EnrollmentId:    req.EnrollmentId,
		OverallProgress: int32(overall),
		Lessons:         lessonProgressToProto(progress),
	}, nil
}

func (h *Handler) GetProgress(ctx context.Context, req *enrollmentpb.GetProgressRequest) (*enrollmentpb.ProgressResponse, error) {
	if req.EnrollmentId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "enrollment_id is required")
	}
	e, err := h.repo.GetByID(ctx, req.EnrollmentId)
	if err != nil {
		if errors.Is(err, ErrEnrollmentNotFound) {
			return nil, status.Errorf(codes.NotFound, "enrollment not found")
		}
		return nil, status.Errorf(codes.Internal, "get enrollment: %v", err)
	}
	progress, _ := h.repo.GetProgress(ctx, req.EnrollmentId)
	return &enrollmentpb.ProgressResponse{
		EnrollmentId:    e.ID,
		OverallProgress: int32(e.OverallProgress),
		Lessons:         lessonProgressToProto(progress),
	}, nil
}

func enrollmentToProto(e EnrollmentRecord) *enrollmentpb.EnrollmentResponse {
	resp := &enrollmentpb.EnrollmentResponse{
		EnrollmentId:    e.ID,
		UserId:          e.UserID,
		CourseId:        e.CourseID,
		Status:          e.Status,
		OverallProgress: int32(e.OverallProgress),
		EnrolledAt:      e.EnrolledAt.Unix(),
	}
	if e.CompletedAt != nil {
		resp.CompletedAt = e.CompletedAt.Unix()
	}
	return resp
}

func lessonProgressToProto(progress []LessonProgressRecord) []*enrollmentpb.LessonProgress {
	var result []*enrollmentpb.LessonProgress
	for _, p := range progress {
		result = append(result, &enrollmentpb.LessonProgress{
			LessonId:        p.LessonID,
			PercentComplete: int32(p.PercentComplete),
			LastAccessed:    p.LastAccessed.Unix(),
		})
	}
	return result
}
