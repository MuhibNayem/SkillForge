package assessment

import (
	"context"
	"testing"

	assessmentpb "github.com/amnayem/skillforge/shared/pb/assessment"
	"github.com/amnayem/skillforge/shared/pkg/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCreateAssessment(t *testing.T) {
	repo := NewMockRepository()
	handler := NewHandler(repo)
	ctx := context.WithValue(context.Background(), auth.TenantIDKey, "tenant-123")

	req := &assessmentpb.CreateAssessmentRequest{
		LessonId:               "l1",
		Title:                  "Test Quiz",
		Description:            "A quiz",
		PassingScorePercentage: 80,
		Questions: []*assessmentpb.Question{
			{Type: "true_false", Text: "Is Go fast?", CorrectAnswer: "true", Points: 10},
		},
	}

	a, err := handler.CreateAssessment(ctx, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if a.Id == "" {
		t.Errorf("expected non-empty ID")
	}
	if a.Title != "Test Quiz" {
		t.Errorf("expected Test Quiz, got %s", a.Title)
	}
}

func TestCreateAssessment_MissingFields(t *testing.T) {
	repo := NewMockRepository()
	handler := NewHandler(repo)
	ctx := context.WithValue(context.Background(), auth.TenantIDKey, "tenant-123")

	req := &assessmentpb.CreateAssessmentRequest{
		// Missing LessonId and Title
	}

	_, err := handler.CreateAssessment(ctx, req)
	if err == nil {
		t.Fatal("expected error for missing fields")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", status.Code(err))
	}
}
