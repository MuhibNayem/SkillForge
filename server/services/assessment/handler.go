package assessment

import (
	"context"
	"errors"

	assessmentpb "github.com/amnayem/skillforge/shared/pb/assessment"
	"github.com/amnayem/skillforge/shared/pkg/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	assessmentpb.UnimplementedAssessmentServiceServer
	repo Repository
}

func NewHandler(repo Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) CreateAssessment(ctx context.Context, req *assessmentpb.CreateAssessmentRequest) (*assessmentpb.Assessment, error) {
	tenantID, err := auth.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "unauthenticated")
	}

	if req.LessonId == "" || req.Title == "" {
		return nil, status.Error(codes.InvalidArgument, "lesson_id and title are required")
	}

	a := &assessmentpb.Assessment{
		TenantId:               tenantID,
		LessonId:               req.LessonId,
		Title:                  req.Title,
		Description:            req.Description,
		PassingScorePercentage: req.PassingScorePercentage,
		Questions:              req.Questions,
	}

	created, err := h.repo.CreateAssessment(ctx, a)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create assessment: %v", err)
	}

	return created, nil
}

func (h *Handler) GetAssessment(ctx context.Context, req *assessmentpb.GetAssessmentRequest) (*assessmentpb.Assessment, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "assessment id is required")
	}

	a, err := h.repo.GetAssessment(ctx, req.Id)
	if err != nil {
		if errors.Is(err, ErrAssessmentNotFound) {
			return nil, status.Error(codes.NotFound, "assessment not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get assessment: %v", err)
	}

	return a, nil
}

func (h *Handler) SubmitAssessment(ctx context.Context, req *assessmentpb.SubmitAssessmentRequest) (*assessmentpb.SubmissionResponse, error) {
	studentID, err := auth.GetUserID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "unauthenticated")
	}

	if req.AssessmentId == "" {
		return nil, status.Error(codes.InvalidArgument, "assessment_id is required")
	}

	res, err := h.repo.SubmitAssessment(ctx, studentID, req)
	if err != nil {
		if errors.Is(err, ErrAssessmentNotFound) {
			return nil, status.Error(codes.NotFound, "assessment not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to submit assessment: %v", err)
	}

	return res, nil
}

func (h *Handler) GetGrade(ctx context.Context, req *assessmentpb.GetGradeRequest) (*assessmentpb.GradeResponse, error) {
	if req.AssessmentId == "" || req.SubmissionId == "" {
		return nil, status.Error(codes.InvalidArgument, "assessment_id and submission_id are required")
	}

	res, err := h.repo.GetGrade(ctx, req.AssessmentId, req.SubmissionId)
	if err != nil {
		if errors.Is(err, ErrSubmissionNotFound) {
			return nil, status.Error(codes.NotFound, "submission not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get grade: %v", err)
	}

	return res, nil
}
