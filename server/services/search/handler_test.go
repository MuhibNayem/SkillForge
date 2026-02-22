package search

import (
	"context"
	"errors"
	"testing"

	searchpb "github.com/amnayem/skillforge/shared/pb/search"
	"github.com/amnayem/skillforge/shared/pkg/auth"
)

// mockRepo is an in-memory Repository for tests.
type mockRepo struct {
	results []SearchResult
	total   int
	indexed []IndexDocument
	err     error
}

func (m *mockRepo) Search(_ context.Context, _, _ string, _ []string, _, _ int) ([]SearchResult, int, error) {
	return m.results, m.total, m.err
}
func (m *mockRepo) IndexDocument(_ context.Context, doc IndexDocument) error {
	m.indexed = append(m.indexed, doc)
	return m.err
}

func newAuthCtx(role, userID, tenantID string) context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, auth.RoleKey, role)
	ctx = context.WithValue(ctx, auth.UserIDKey, userID)
	ctx = context.WithValue(ctx, auth.TenantIDKey, tenantID)
	return ctx
}

func TestGlobalSearch_Basic(t *testing.T) {
	repo := &mockRepo{
		results: []SearchResult{
			{ID: "c1", Type: "course", Title: "Go Basics", RelevanceScore: 0.9},
		},
		total: 1,
	}
	h := NewHandler(repo)
	ctx := newAuthCtx("student", "user-1", "tenant-1")

	resp, err := h.GlobalSearch(ctx, &searchpb.GlobalSearchRequest{Query: "Go"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(resp.Results))
	}
	if resp.TotalHits != 1 {
		t.Errorf("expected total_hits=1, got %d", resp.TotalHits)
	}
}

func TestGlobalSearch_EmptyQuery(t *testing.T) {
	h := NewHandler(&mockRepo{})
	ctx := newAuthCtx("student", "user-1", "tenant-1")
	_, err := h.GlobalSearch(ctx, &searchpb.GlobalSearchRequest{Query: ""})
	if err == nil {
		t.Fatal("expected invalid argument error")
	}
}

func TestGlobalSearch_UsesJWTTenantID(t *testing.T) {
	repo := &mockRepo{results: []SearchResult{}, total: 0}
	h := NewHandler(repo)
	ctx := newAuthCtx("student", "user-1", "tenant-xyz")

	// TenantId not set in request — should fall back to JWT.
	_, err := h.GlobalSearch(ctx, &searchpb.GlobalSearchRequest{Query: "python"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGlobalSearch_Unauthenticated_NoTenant(t *testing.T) {
	h := NewHandler(&mockRepo{})
	// No auth context, no tenant in request.
	_, err := h.GlobalSearch(context.Background(), &searchpb.GlobalSearchRequest{Query: "test"})
	if err == nil {
		t.Fatal("expected unauthenticated error")
	}
}

func TestGlobalSearch_RepoError(t *testing.T) {
	h := NewHandler(&mockRepo{err: errors.New("es error")})
	ctx := newAuthCtx("student", "user-1", "tenant-1")
	_, err := h.GlobalSearch(ctx, &searchpb.GlobalSearchRequest{Query: "test"})
	if err == nil {
		t.Fatal("expected internal error")
	}
}

func TestGlobalSearch_WithFilters(t *testing.T) {
	repo := &mockRepo{results: []SearchResult{{ID: "c1", Type: "course"}}, total: 1}
	h := NewHandler(repo)
	ctx := newAuthCtx("instructor", "inst-1", "tenant-1")

	resp, err := h.GlobalSearch(ctx, &searchpb.GlobalSearchRequest{
		Query:   "machine learning",
		Filters: []string{"type:course"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Results[0].Type != "course" {
		t.Errorf("expected type=course, got %s", resp.Results[0].Type)
	}
}
