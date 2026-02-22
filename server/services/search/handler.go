package search

import (
	"context"

	searchpb "github.com/amnayem/skillforge/shared/pb/search"
	"github.com/amnayem/skillforge/shared/pkg/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Handler implements searchpb.SearchServiceServer.
type Handler struct {
	searchpb.UnimplementedSearchServiceServer
	repo Repository
}

func NewHandler(repo Repository) *Handler {
	return &Handler{repo: repo}
}

// GlobalSearch performs a fuzzy full-text search across courses, lessons, and users
// scoped to the caller's tenant.
func (h *Handler) GlobalSearch(ctx context.Context, req *searchpb.GlobalSearchRequest) (*searchpb.GlobalSearchResponse, error) {
	if req.Query == "" {
		return nil, status.Errorf(codes.InvalidArgument, "query is required")
	}

	// Authenticated users only; tenantID comes from JWT if not supplied in request.
	tenantID := req.TenantId
	if tenantID == "" {
		var err error
		tenantID, err = auth.GetTenantID(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
		}
	}

	limit := int(req.Limit)
	if limit == 0 {
		limit = 10
	}

	results, totalHits, err := h.repo.Search(ctx, req.Query, tenantID, req.Filters, limit, int(req.Offset))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "search: %v", err)
	}

	pbResults := make([]*searchpb.SearchResult, 0, len(results))
	for _, r := range results {
		pbResults = append(pbResults, &searchpb.SearchResult{
			Id:             r.ID,
			Type:           r.Type,
			Title:          r.Title,
			Description:    r.Description,
			Url:            r.URL,
			RelevanceScore: r.RelevanceScore,
		})
	}

	return &searchpb.GlobalSearchResponse{
		Results:   pbResults,
		TotalHits: int32(totalHits),
	}, nil
}
