package content

import (
	"context"
	"errors"
	"fmt"

	"github.com/amnayem/skillforge/shared/pb/contentpb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	contentpb.UnimplementedContentServiceServer
	repo     Repository
	minioURL string
}

func NewHandler(repo Repository, minioURL string) *Handler {
	return &Handler{repo: repo, minioURL: minioURL}
}

func (h *Handler) UploadContent(ctx context.Context, req *contentpb.UploadContentRequest) (*contentpb.ContentResponse, error) {
	if req.Filename == "" {
		return nil, status.Errorf(codes.InvalidArgument, "filename is required")
	}
	rec, err := h.repo.Store(ctx, ContentRecord{
		Filename:    req.Filename,
		ContentType: req.ContentType,
		SizeBytes:   req.SizeBytes,
		TenantID:    req.TenantId,
		CourseID:    req.CourseId,
		UploadedBy:  req.UploadedBy,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "store content: %v", err)
	}
	return contentToProto(rec, h.minioURL), nil
}

func (h *Handler) GetContent(ctx context.Context, req *contentpb.GetContentRequest) (*contentpb.ContentResponse, error) {
	if req.ContentId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "content_id is required")
	}
	rec, err := h.repo.GetByID(ctx, req.ContentId)
	if err != nil {
		if errors.Is(err, ErrContentNotFound) {
			return nil, status.Errorf(codes.NotFound, "content not found")
		}
		return nil, status.Errorf(codes.Internal, "get content: %v", err)
	}
	return contentToProto(rec, h.minioURL), nil
}

func (h *Handler) DeleteContent(ctx context.Context, req *contentpb.DeleteContentRequest) (*contentpb.DeleteContentResponse, error) {
	if req.ContentId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "content_id is required")
	}
	if err := h.repo.Delete(ctx, req.ContentId); err != nil {
		if errors.Is(err, ErrContentNotFound) {
			return nil, status.Errorf(codes.NotFound, "content not found")
		}
		return nil, status.Errorf(codes.Internal, "delete content: %v", err)
	}
	return &contentpb.DeleteContentResponse{Success: true}, nil
}

func (h *Handler) ListContent(ctx context.Context, req *contentpb.ListContentRequest) (*contentpb.ListContentResponse, error) {
	records, total, err := h.repo.List(ctx, req.TenantId, req.CourseId, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list content: %v", err)
	}
	resp := &contentpb.ListContentResponse{Total: int32(total)}
	for _, r := range records {
		resp.Items = append(resp.Items, contentToProto(r, h.minioURL))
	}
	return resp, nil
}

func contentToProto(rec ContentRecord, minioURL string) *contentpb.ContentResponse {
	return &contentpb.ContentResponse{
		ContentId:   rec.ID,
		Filename:    rec.Filename,
		ContentType: rec.ContentType,
		SizeBytes:   rec.SizeBytes,
		DownloadUrl: fmt.Sprintf("%s/%s/%s/%s", minioURL, rec.TenantID, rec.CourseID, rec.Filename),
		TenantId:    rec.TenantID,
		CourseId:    rec.CourseID,
		UploadedBy:  rec.UploadedBy,
		Status:      rec.Status,
		CreatedAt:   rec.CreatedAt.Unix(),
	}
}
