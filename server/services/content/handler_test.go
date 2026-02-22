package content_test

import (
	"context"
	"testing"
	"time"

	content "github.com/amnayem/skillforge/services/content"
	"github.com/amnayem/skillforge/shared/pb/contentpb"
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

// mockContentRepo is an in-memory Repository implementation.
type mockContentRepo struct {
	records map[string]content.ContentRecord
	nextID  int
}

func newMockContentRepo() *mockContentRepo {
	return &mockContentRepo{records: make(map[string]content.ContentRecord)}
}

func (m *mockContentRepo) Store(_ context.Context, rec content.ContentRecord) (content.ContentRecord, error) {
	m.nextID++
	rec.ID = "content-id-" + string(rune('0'+m.nextID))
	rec.Status = "uploaded"
	rec.CreatedAt = time.Now()
	m.records[rec.ID] = rec
	return rec, nil
}

func (m *mockContentRepo) GetByID(_ context.Context, id string) (content.ContentRecord, error) {
	rec, ok := m.records[id]
	if !ok {
		return rec, content.ErrContentNotFound
	}
	return rec, nil
}

func (m *mockContentRepo) Delete(_ context.Context, id string) error {
	if _, ok := m.records[id]; !ok {
		return content.ErrContentNotFound
	}
	delete(m.records, id)
	return nil
}

func (m *mockContentRepo) List(_ context.Context, tenantID, courseID string, page, pageSize int) ([]content.ContentRecord, int, error) {
	var results []content.ContentRecord
	for _, r := range m.records {
		if r.TenantID == tenantID && (courseID == "" || r.CourseID == courseID) {
			results = append(results, r)
		}
	}
	return results, len(results), nil
}

// ─── Tests ───────────────────────────────────────────────────────────────────

func TestUploadContent(t *testing.T) {
	repo := newMockContentRepo()
	h := content.NewHandler(repo, "http://minio:9000")
	ctx := newAuthCtx("instructor", "user-1", "tenant-1")

	resp, err := h.UploadContent(ctx, &contentpb.UploadContentRequest{
		Filename:    "lecture.mp4",
		ContentType: "video/mp4",
		SizeBytes:   1024 * 1024,
		CourseId:    "course-1",
	})
	if err != nil {
		t.Fatalf("UploadContent failed: %v", err)
	}
	if resp.ContentId == "" {
		t.Error("expected non-empty content ID")
	}
	if resp.Filename != "lecture.mp4" {
		t.Errorf("expected 'lecture.mp4', got %q", resp.Filename)
	}
	if resp.Status != "uploaded" {
		t.Errorf("expected status 'uploaded', got %q", resp.Status)
	}
}

func TestUploadContent_MissingFilename(t *testing.T) {
	repo := newMockContentRepo()
	h := content.NewHandler(repo, "http://minio:9000")

	_, err := h.UploadContent(newAuthCtx("instructor", "u1", "t1"), &contentpb.UploadContentRequest{
		CourseId: "course-1",
	})
	if err == nil {
		t.Fatal("expected error for missing filename")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", status.Code(err))
	}
}

func TestGetContent(t *testing.T) {
	repo := newMockContentRepo()
	h := content.NewHandler(repo, "http://minio:9000")
	ctx := newAuthCtx("instructor", "u1", "t1")

	uploaded, _ := h.UploadContent(ctx, &contentpb.UploadContentRequest{
		Filename: "slides.pdf", ContentType: "application/pdf", CourseId: "c1",
	})

	resp, err := h.GetContent(context.Background(), &contentpb.GetContentRequest{
		ContentId: uploaded.ContentId,
	})
	if err != nil {
		t.Fatalf("GetContent failed: %v", err)
	}
	if resp.Filename != "slides.pdf" {
		t.Errorf("expected 'slides.pdf', got %q", resp.Filename)
	}
}

func TestGetContent_NotFound(t *testing.T) {
	repo := newMockContentRepo()
	h := content.NewHandler(repo, "http://minio:9000")

	_, err := h.GetContent(context.Background(), &contentpb.GetContentRequest{
		ContentId: "nonexistent-id",
	})
	if err == nil {
		t.Fatal("expected error for missing content")
	}
	if status.Code(err) != codes.NotFound {
		t.Errorf("expected NotFound, got %v", status.Code(err))
	}
}

func TestGetContent_MissingID(t *testing.T) {
	repo := newMockContentRepo()
	h := content.NewHandler(repo, "http://minio:9000")

	_, err := h.GetContent(context.Background(), &contentpb.GetContentRequest{})
	if err == nil {
		t.Fatal("expected error for empty content_id")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", status.Code(err))
	}
}

func TestDeleteContent(t *testing.T) {
	repo := newMockContentRepo()
	h := content.NewHandler(repo, "http://minio:9000")
	ctx := newAuthCtx("instructor", "u1", "t1")

	uploaded, _ := h.UploadContent(ctx, &contentpb.UploadContentRequest{
		Filename: "video.mp4", ContentType: "video/mp4", CourseId: "c1",
	})

	resp, err := h.DeleteContent(ctx, &contentpb.DeleteContentRequest{
		ContentId: uploaded.ContentId,
	})
	if err != nil {
		t.Fatalf("DeleteContent failed: %v", err)
	}
	if !resp.Success {
		t.Error("expected success=true")
	}

	// Second delete should return NotFound
	_, err = h.DeleteContent(ctx, &contentpb.DeleteContentRequest{
		ContentId: uploaded.ContentId,
	})
	if status.Code(err) != codes.NotFound {
		t.Errorf("expected NotFound on re-delete, got %v", status.Code(err))
	}
}

func TestDeleteContent_MissingID(t *testing.T) {
	repo := newMockContentRepo()
	h := content.NewHandler(repo, "http://minio:9000")

	_, err := h.DeleteContent(newAuthCtx("instructor", "u1", "t1"), &contentpb.DeleteContentRequest{})
	if err == nil {
		t.Fatal("expected error for empty content_id")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", status.Code(err))
	}
}

func TestListContent(t *testing.T) {
	repo := newMockContentRepo()
	h := content.NewHandler(repo, "http://minio:9000")
	ctx := newAuthCtx("instructor", "u1", "t1")

	for i := 0; i < 3; i++ {
		_, _ = h.UploadContent(ctx, &contentpb.UploadContentRequest{
			Filename: "file.mp4", ContentType: "video/mp4", CourseId: "c1",
		})
	}

	resp, err := h.ListContent(context.Background(), &contentpb.ListContentRequest{
		TenantId: "t1",
		CourseId: "c1",
	})
	if err != nil {
		t.Fatalf("ListContent failed: %v", err)
	}
	if resp.Total != 3 {
		t.Errorf("expected 3 items, got %d", resp.Total)
	}
}
