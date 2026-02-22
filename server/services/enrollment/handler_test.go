package enrollment_test

import (
	"context"
	"testing"
	"time"

	enrollment "github.com/amnayem/skillforge/services/enrollment"
	"github.com/amnayem/skillforge/shared/pb/enrollmentpb"
)

type mockRepo struct {
	enrollments map[string]enrollment.EnrollmentRecord
	progress    map[string][]enrollment.LessonProgressRecord
	nextID      int
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		enrollments: make(map[string]enrollment.EnrollmentRecord),
		progress:    make(map[string][]enrollment.LessonProgressRecord),
	}
}

func (m *mockRepo) Enroll(ctx context.Context, tenantID, userID, courseID string) (enrollment.EnrollmentRecord, error) {
	for _, e := range m.enrollments {
		if e.UserID == userID && e.CourseID == courseID {
			return enrollment.EnrollmentRecord{}, enrollment.ErrAlreadyEnrolled
		}
	}
	m.nextID++
	e := enrollment.EnrollmentRecord{
		ID: "e-" + courseID, TenantID: tenantID, UserID: userID, CourseID: courseID,
		Status: "active", OverallProgress: 0, EnrolledAt: time.Now(),
	}
	m.enrollments[e.ID] = e
	return e, nil
}

func (m *mockRepo) GetByID(ctx context.Context, id string) (enrollment.EnrollmentRecord, error) {
	e, ok := m.enrollments[id]
	if !ok {
		return e, enrollment.ErrEnrollmentNotFound
	}
	return e, nil
}

func (m *mockRepo) List(ctx context.Context, userID, tenantID string, page, pageSize int) ([]enrollment.EnrollmentRecord, int, error) {
	var results []enrollment.EnrollmentRecord
	for _, e := range m.enrollments {
		if e.UserID == userID {
			results = append(results, e)
		}
	}
	return results, len(results), nil
}

func (m *mockRepo) UpdateProgress(ctx context.Context, enrollmentID, lessonID string, percent int) error {
	m.progress[enrollmentID] = append(m.progress[enrollmentID], enrollment.LessonProgressRecord{
		LessonID: lessonID, PercentComplete: percent, LastAccessed: time.Now(),
	})
	return nil
}

func (m *mockRepo) GetProgress(ctx context.Context, enrollmentID string) ([]enrollment.LessonProgressRecord, error) {
	return m.progress[enrollmentID], nil
}

func (m *mockRepo) RecalculateOverallProgress(ctx context.Context, enrollmentID string) (int, error) {
	progress := m.progress[enrollmentID]
	if len(progress) == 0 {
		return 0, nil
	}
	total := 0
	for _, p := range progress {
		total += p.PercentComplete
	}
	return total / len(progress), nil
}

func TestEnroll(t *testing.T) {
	h := enrollment.NewHandler(newMockRepo())
	resp, err := h.Enroll(context.Background(), &enrollmentpb.EnrollRequest{
		UserId: "u-1", CourseId: "c-1", TenantId: "t-1",
	})
	if err != nil {
		t.Fatalf("Enroll failed: %v", err)
	}
	if resp.Status != "active" {
		t.Errorf("expected active, got %s", resp.Status)
	}
}

func TestEnroll_Duplicate(t *testing.T) {
	repo := newMockRepo()
	h := enrollment.NewHandler(repo)
	_, _ = h.Enroll(context.Background(), &enrollmentpb.EnrollRequest{
		UserId: "u-1", CourseId: "c-1", TenantId: "t-1",
	})
	_, err := h.Enroll(context.Background(), &enrollmentpb.EnrollRequest{
		UserId: "u-1", CourseId: "c-1", TenantId: "t-1",
	})
	if err == nil {
		t.Error("expected error for duplicate enrollment")
	}
}

func TestEnroll_MissingFields(t *testing.T) {
	h := enrollment.NewHandler(newMockRepo())
	_, err := h.Enroll(context.Background(), &enrollmentpb.EnrollRequest{})
	if err == nil {
		t.Error("expected error for missing fields")
	}
}

func TestUpdateProgress(t *testing.T) {
	repo := newMockRepo()
	h := enrollment.NewHandler(repo)
	enrolled, _ := h.Enroll(context.Background(), &enrollmentpb.EnrollRequest{
		UserId: "u-1", CourseId: "c-1", TenantId: "t-1",
	})
	resp, err := h.UpdateProgress(context.Background(), &enrollmentpb.UpdateProgressRequest{
		EnrollmentId: enrolled.EnrollmentId, LessonId: "l-1", PercentComplete: 75,
	})
	if err != nil {
		t.Fatalf("UpdateProgress failed: %v", err)
	}
	if resp.OverallProgress != 75 {
		t.Errorf("expected 75, got %d", resp.OverallProgress)
	}
}
