package assessment

import (
	"context"

	assessmentpb "github.com/amnayem/skillforge/shared/pb/assessment"
)

type MockRepository struct {
	Assessments map[string]*assessmentpb.Assessment
	Submissions map[string]*assessmentpb.SubmissionResponse
	Grades      map[string]*assessmentpb.GradeResponse
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		Assessments: make(map[string]*assessmentpb.Assessment),
		Submissions: make(map[string]*assessmentpb.SubmissionResponse),
		Grades:      make(map[string]*assessmentpb.GradeResponse),
	}
}

func (m *MockRepository) CreateAssessment(ctx context.Context, a *assessmentpb.Assessment) (*assessmentpb.Assessment, error) {
	a.Id = "mock-id-1"
	m.Assessments[a.Id] = a
	return a, nil
}

func (m *MockRepository) GetAssessment(ctx context.Context, id string) (*assessmentpb.Assessment, error) {
	if a, ok := m.Assessments[id]; ok {
		return a, nil
	}
	return nil, ErrAssessmentNotFound
}

func (m *MockRepository) SubmitAssessment(ctx context.Context, studentID string, req *assessmentpb.SubmitAssessmentRequest) (*assessmentpb.SubmissionResponse, error) {
	res := &assessmentpb.SubmissionResponse{
		SubmissionId: "submission-1",
		Status:       "graded",
	}
	m.Submissions["submission-1"] = res

	m.Grades["submission-1"] = &assessmentpb.GradeResponse{
		SubmissionId: "submission-1",
		TotalPoints:  10,
		EarnedPoints: 10,
		Passed:       true,
		Feedback:     "Great job",
	}

	return res, nil
}

func (m *MockRepository) GetGrade(ctx context.Context, assessmentID, submissionID string) (*assessmentpb.GradeResponse, error) {
	if g, ok := m.Grades[submissionID]; ok {
		return g, nil
	}
	return nil, ErrSubmissionNotFound
}
