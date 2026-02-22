package assessment

import (
	"context"

	assessmentpb "github.com/amnayem/skillforge/shared/pb/assessment"
)

type Repository interface {
	CreateAssessment(ctx context.Context, assessment *assessmentpb.Assessment) (*assessmentpb.Assessment, error)
	GetAssessment(ctx context.Context, id string) (*assessmentpb.Assessment, error)
	SubmitAssessment(ctx context.Context, studentID string, req *assessmentpb.SubmitAssessmentRequest) (*assessmentpb.SubmissionResponse, error)
	GetGrade(ctx context.Context, assessmentID, submissionID string) (*assessmentpb.GradeResponse, error)
}
