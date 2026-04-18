package handler

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/learnhub/lms/server/services/discussion/internal/model"
	"github.com/learnhub/lms/server/services/discussion/internal/repository"
	"github.com/learnhub/lms/server/shared/pb/discussion"
	"github.com/learnhub/lms/server/shared/pkg/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Handler implements the DiscussionService gRPC server
type Handler struct {
	discussion.UnimplementedDiscussionServiceServer
	repo repository.Repository
}

// NewHandler creates a new discussion handler
func NewHandler(repo repository.Repository) *Handler {
	return &Handler{repo: repo}
}

// CreateThread creates a new discussion thread
func (h *Handler) CreateThread(ctx context.Context, req *discussion.CreateThreadRequest) (*discussion.ThreadResponse, error) {
	// Get user from context
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	userName, _ := auth.GetUserNameFromContext(ctx)
	userAvatar, _ := auth.GetUserAvatarFromContext(ctx)
	tenantID, _ := auth.GetTenantIDFromContext(ctx)

	thread := &model.Thread{
		TenantID:     req.TenantId,
		CourseID:     req.CourseId,
		ModuleID:     req.ModuleId,
		LessonID:     req.LessonId,
		AuthorID:     userID,
		AuthorName:   userName,
		AuthorAvatar: userAvatar,
		Title:        req.Title,
		Content:      req.Content,
		Tags:         req.Tags,
		Category:     req.Category,
		Upvotes:      0,
		Downvotes:    0,
		ReplyCount:   0,
		IsPinned:     false,
		IsResolved:   false,
	}

	// Set default tenant if not provided
	if thread.TenantID == "" {
		thread.TenantID = tenantID
	}

	// Set default category
	if thread.Category == "" {
		thread.Category = "general"
	}

	created, err := h.repo.CreateThread(ctx, thread)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create thread")
	}

	return &discussion.ThreadResponse{
		Thread: h.toProtoThread(created),
	}, nil
}

// GetThread retrieves a specific thread
func (h *Handler) GetThread(ctx context.Context, req *discussion.GetThreadRequest) (*discussion.ThreadResponse, error) {
	thread, err := h.repo.GetThread(ctx, req.ThreadId)
	if err != nil {
		if errors.Is(err, repository.ErrThreadNotFound) {
			return nil, status.Error(codes.NotFound, "thread not found")
		}
		return nil, status.Error(codes.Internal, "failed to get thread")
	}

	// Enrich with user-specific data
	h.enrichThreadWithUserData(ctx, thread)

	return &discussion.ThreadResponse{
		Thread: h.toProtoThread(thread),
	}, nil
}

// ListThreads lists threads with filtering and pagination
func (h *Handler) ListThreads(ctx context.Context, req *discussion.ListThreadsRequest) (*discussion.ListThreadsResponse, error) {
	filter := repository.ThreadFilter{
		TenantID:         req.TenantId,
		CourseID:         req.CourseId,
		ModuleID:         req.ModuleId,
		LessonID:         req.LessonId,
		Category:         req.Category,
		Search:           req.Search,
		Tags:             req.Tags,
		IncludePinnedFirst: req.IncludePinnedFirst,
		ShowResolved:     req.ShowResolved,
		SortBy:           req.SortBy,
		SortOrder:        req.SortOrder,
		Page:             req.Page,
		PageSize:         req.PageSize,
	}

	// Set defaults
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}
	if filter.SortBy == "" {
		filter.SortBy = "updated_at"
	}
	if filter.SortOrder == "" {
		filter.SortOrder = "desc"
	}

	threads, totalCount, err := h.repo.ListThreads(ctx, filter)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list threads")
	}

	// Enrich threads with user data
	for _, thread := range threads {
		h.enrichThreadWithUserData(ctx, thread)
	}

	totalPages := int32((totalCount + int64(filter.PageSize) - 1) / int64(filter.PageSize))

	protoThreads := make([]*discussion.Thread, len(threads))
	for i, t := range threads {
		protoThreads[i] = h.toProtoThread(t)
	}

	return &discussion.ListThreadsResponse{
		Threads:    protoThreads,
		TotalCount: int32(totalCount),
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
	}, nil
}

// UpdateThread updates an existing thread
func (h *Handler) UpdateThread(ctx context.Context, req *discussion.UpdateThreadRequest) (*discussion.ThreadResponse, error) {
	// Verify ownership
	existing, err := h.repo.GetThread(ctx, req.ThreadId)
	if err != nil {
		if errors.Is(err, repository.ErrThreadNotFound) {
			return nil, status.Error(codes.NotFound, "thread not found")
		}
		return nil, status.Error(codes.Internal, "failed to get thread")
	}

	userID, _ := auth.GetUserIDFromContext(ctx)
	if existing.AuthorID != userID {
		// Check if user is instructor or admin
		userRole, _ := auth.GetUserRoleFromContext(ctx)
		if userRole != "instructor" && userRole != "tenant_admin" && userRole != "super_admin" {
			return nil, status.Error(codes.PermissionDenied, "not authorized to update this thread")
		}
	}

	thread := &model.Thread{
		ID:       existing.ID,
		Title:    req.Title,
		Content:  req.Content,
		Tags:     req.Tags,
		Category: req.Category,
	}

	updated, err := h.repo.UpdateThread(ctx, thread)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to update thread")
	}

	return &discussion.ThreadResponse{
		Thread: h.toProtoThread(updated),
	}, nil
}

// DeleteThread soft deletes a thread
func (h *Handler) DeleteThread(ctx context.Context, req *discussion.DeleteThreadRequest) (*discussion.EmptyResponse, error) {
	thread, err := h.repo.GetThread(ctx, req.ThreadId)
	if err != nil {
		if errors.Is(err, repository.ErrThreadNotFound) {
			return nil, status.Error(codes.NotFound, "thread not found")
		}
		return nil, status.Error(codes.Internal, "failed to get thread")
	}

	userID, _ := auth.GetUserIDFromContext(ctx)
	userRole, _ := auth.GetUserRoleFromContext(ctx)

	// Only author, instructor, or admin can delete
	if thread.AuthorID != userID && userRole != "instructor" && userRole != "tenant_admin" && userRole != "super_admin" {
		return nil, status.Error(codes.PermissionDenied, "not authorized to delete this thread")
	}

	if err := h.repo.DeleteThread(ctx, req.ThreadId); err != nil {
		return nil, status.Error(codes.Internal, "failed to delete thread")
	}

	return &discussion.EmptyResponse{}, nil
}

// PinThread pins or unpins a thread (instructor/admin only)
func (h *Handler) PinThread(ctx context.Context, req *discussion.PinThreadRequest) (*discussion.ThreadResponse, error) {
	userRole, _ := auth.GetUserRoleFromContext(ctx)
	if userRole != "instructor" && userRole != "tenant_admin" && userRole != "super_admin" {
		return nil, status.Error(codes.PermissionDenied, "only instructors and admins can pin threads")
	}

	thread, err := h.repo.PinThread(ctx, req.ThreadId, req.Pinned)
	if err != nil {
		if errors.Is(err, repository.ErrThreadNotFound) {
			return nil, status.Error(codes.NotFound, "thread not found")
		}
		return nil, status.Error(codes.Internal, "failed to pin thread")
	}

	return &discussion.ThreadResponse{
		Thread: h.toProtoThread(thread),
	}, nil
}

// ResolveThread marks a thread as resolved
func (h *Handler) ResolveThread(ctx context.Context, req *discussion.ResolveThreadRequest) (*discussion.ThreadResponse, error) {
	userRole, _ := auth.GetUserRoleFromContext(ctx)
	if userRole != "instructor" && userRole != "tenant_admin" && userRole != "super_admin" {
		return nil, status.Error(codes.PermissionDenied, "only instructors and admins can resolve threads")
	}

	thread, err := h.repo.ResolveThread(ctx, req.ThreadId, req.ReplyId)
	if err != nil {
		if errors.Is(err, repository.ErrThreadNotFound) {
			return nil, status.Error(codes.NotFound, "thread not found")
		}
		return nil, status.Error(codes.Internal, "failed to resolve thread")
	}

	return &discussion.ThreadResponse{
		Thread: h.toProtoThread(thread),
	}, nil
}

// CreateReply creates a reply to a thread
func (h *Handler) CreateReply(ctx context.Context, req *discussion.CreateReplyRequest) (*discussion.ReplyResponse, error) {
	userID, _ := auth.GetUserIDFromContext(ctx)
	userName, _ := auth.GetUserNameFromContext(ctx)
	userAvatar, _ := auth.GetUserAvatarFromContext(ctx)
	tenantID, _ := auth.GetTenantIDFromContext(ctx)

	// Get thread to verify it exists and get tenant
	thread, err := h.repo.GetThread(ctx, req.ThreadId)
	if err != nil {
		if errors.Is(err, repository.ErrThreadNotFound) {
			return nil, status.Error(codes.NotFound, "thread not found")
		}
		return nil, status.Error(codes.Internal, "failed to get thread")
	}

	reply := &model.Reply{
		ThreadID:   req.ThreadId,
		TenantID:   thread.TenantID,
		AuthorID:   userID,
		AuthorName: userName,
		AuthorAvatar: userAvatar,
		Content:    req.Content,
		Upvotes:    0,
		Downvotes:  0,
		IsAccepted: false,
	}

	// Handle nested replies
	if req.ParentReplyId != "" {
		// Validate parent reply exists
		// For simplicity, we'll set it directly
		// In production, add proper validation
	}

	created, err := h.repo.CreateReply(ctx, reply)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create reply")
	}

	return &discussion.ReplyResponse{
		Reply: h.toProtoReply(created),
	}, nil
}

// ListReplies lists replies for a thread
func (h *Handler) ListReplies(ctx context.Context, req *discussion.ListRepliesRequest) (*discussion.ListRepliesResponse, error) {
	filter := repository.ReplyFilter{
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
		Page:      req.Page,
		PageSize:  req.PageSize,
	}

	// Set defaults
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 50
	}
	if filter.PageSize > 200 {
		filter.PageSize = 200
	}
	if filter.SortBy == "" {
		filter.SortBy = "created_at"
	}
	if filter.SortOrder == "" {
		filter.SortOrder = "asc"
	}

	replies, totalCount, err := h.repo.ListReplies(ctx, req.ThreadId, filter)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list replies")
	}

	// Enrich with user vote data
	for _, reply := range replies {
		h.enrichReplyWithUserData(ctx, reply)
	}

	totalPages := int32((totalCount + int64(filter.PageSize) - 1) / int64(filter.PageSize))

	protoReplies := make([]*discussion.Reply, len(replies))
	for i, r := range replies {
		protoReplies[i] = h.toProtoReply(r)
	}

	return &discussion.ListRepliesResponse{
		Replies:    protoReplies,
		TotalCount: int32(totalCount),
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
	}, nil
}

// UpdateReply updates an existing reply
func (h *Handler) UpdateReply(ctx context.Context, req *discussion.UpdateReplyRequest) (*discussion.ReplyResponse, error) {
	existing, err := h.repo.GetReply(ctx, req.ReplyId)
	if err != nil {
		if errors.Is(err, repository.ErrReplyNotFound) {
			return nil, status.Error(codes.NotFound, "reply not found")
		}
		return nil, status.Error(codes.Internal, "failed to get reply")
	}

	userID, _ := auth.GetUserIDFromContext(ctx)
	if existing.AuthorID != userID {
		userRole, _ := auth.GetUserRoleFromContext(ctx)
		if userRole != "instructor" && userRole != "tenant_admin" && userRole != "super_admin" {
			return nil, status.Error(codes.PermissionDenied, "not authorized to update this reply")
		}
	}

	reply := &model.Reply{
		ID:      existing.ID,
		ThreadID: existing.ThreadID,
		Content: req.Content,
	}

	updated, err := h.repo.UpdateReply(ctx, reply)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to update reply")
	}

	return &discussion.ReplyResponse{
		Reply: h.toProtoReply(updated),
	}, nil
}

// DeleteReply soft deletes a reply
func (h *Handler) DeleteReply(ctx context.Context, req *discussion.DeleteReplyRequest) (*discussion.EmptyResponse, error) {
	reply, err := h.repo.GetReply(ctx, req.ReplyId)
	if err != nil {
		if errors.Is(err, repository.ErrReplyNotFound) {
			return nil, status.Error(codes.NotFound, "reply not found")
		}
		return nil, status.Error(codes.Internal, "failed to get reply")
	}

	userID, _ := auth.GetUserIDFromContext(ctx)
	userRole, _ := auth.GetUserRoleFromContext(ctx)

	if reply.AuthorID != userID && userRole != "instructor" && userRole != "tenant_admin" && userRole != "super_admin" {
		return nil, status.Error(codes.PermissionDenied, "not authorized to delete this reply")
	}

	if err := h.repo.DeleteReply(ctx, req.ReplyId); err != nil {
		return nil, status.Error(codes.Internal, "failed to delete reply")
	}

	return &discussion.EmptyResponse{}, nil
}

// Vote handles voting on threads and replies
func (h *Handler) Vote(ctx context.Context, req *discussion.VoteRequest) (*discussion.VoteResponse, error) {
	userID, _ := auth.GetUserIDFromContext(ctx)
	tenantID, _ := auth.GetTenantIDFromContext(ctx)

	if req.VoteType < -1 || req.VoteType > 1 {
		return nil, status.Error(codes.InvalidArgument, "vote type must be -1, 0, or 1")
	}

	vote := &model.Vote{
		TenantID:     tenantID,
		UserID:       userID,
		ResourceType: req.Type,
		ResourceID:   req.Id,
		VoteType:     req.VoteType,
	}

	upvotes, downvotes, userVote, err := h.repo.Vote(ctx, vote)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to process vote")
	}

	return &discussion.VoteResponse{
		Upvotes:      upvotes,
		Downvotes:    downvotes,
		UserVoteType: userVote,
	}, nil
}

// Subscribe subscribes a user to a thread
func (h *Handler) Subscribe(ctx context.Context, req *discussion.SubscribeRequest) (*discussion.EmptyResponse, error) {
	userID, _ := auth.GetUserIDFromContext(ctx)
	tenantID, _ := auth.GetTenantIDFromContext(ctx)

	sub := &model.Subscription{
		TenantID: tenantID,
		UserID:   userID,
		ThreadID: req.ThreadId,
	}

	if err := h.repo.Subscribe(ctx, sub); err != nil {
		return nil, status.Error(codes.Internal, "failed to subscribe")
	}

	return &discussion.EmptyResponse{}, nil
}

// Unsubscribe unsubscribes a user from a thread
func (h *Handler) Unsubscribe(ctx context.Context, req *discussion.UnsubscribeRequest) (*discussion.EmptyResponse, error) {
	userID, _ := auth.GetUserIDFromContext(ctx)

	if err := h.repo.Unsubscribe(ctx, userID, req.ThreadId); err != nil {
		return nil, status.Error(codes.Internal, "failed to unsubscribe")
	}

	return &discussion.EmptyResponse{}, nil
}

// SearchDiscussions searches across discussions
func (h *Handler) SearchDiscussions(ctx context.Context, req *discussion.SearchDiscussionsRequest) (*discussion.SearchDiscussionsResponse, error) {
	filter := repository.ThreadFilter{
		TenantID: req.TenantId,
		CourseID: req.CourseId,
		Search:   req.Query,
		Tags:     req.Categories, // Using categories as tags for search
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	threads, totalCount, err := h.repo.ListThreads(ctx, filter)
	if err != nil {
		return nil, status.Error(codes.Internal, "search failed")
	}

	totalPages := int32((totalCount + int64(filter.PageSize) - 1) / int64(filter.PageSize))

	protoThreads := make([]*discussion.Thread, len(threads))
	for i, t := range threads {
		protoThreads[i] = h.toProtoThread(t)
	}

	return &discussion.SearchDiscussionsResponse{
		Threads:    protoThreads,
		TotalCount: int32(totalCount),
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetUserStats gets statistics for a user
func (h *Handler) GetUserStats(ctx context.Context, req *discussion.GetUserStatsRequest) (*discussion.UserStatsResponse, error) {
	tenantID, _ := auth.GetTenantIDFromContext(ctx)

	stats, err := h.repo.GetUserStats(ctx, req.UserId, tenantID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get user stats")
	}

	return &discussion.UserStatsResponse{
		UserId:            stats.UserID,
		ThreadsCreated:    stats.ThreadsCreated,
		RepliesCreated:    stats.RepliesCreated,
		TotalUpvotes:      stats.TotalUpvotes,
		TotalDownvotes:    stats.TotalDownvotes,
		SolutionsProvided: stats.SolutionsProvided,
		BestAnswerCount:   stats.BestAnswerCount,
		ReputationScore:   stats.ReputationScore,
	}, nil
}

// Helper methods

func (h *Handler) toProtoThread(t *model.Thread) *discussion.Thread {
	thread := &discussion.Thread{
		Id:              t.ID.Hex(),
		TenantId:        t.TenantID,
		CourseId:        t.CourseID,
		ModuleId:        t.ModuleID,
		LessonId:        t.LessonID,
		AuthorId:        t.AuthorID,
		AuthorName:      t.AuthorName,
		AuthorAvatar:    t.AuthorAvatar,
		Title:           t.Title,
		Content:         t.Content,
		Tags:            t.Tags,
		Upvotes:         t.Upvotes,
		Downvotes:       t.Downvotes,
		ReplyCount:      t.ReplyCount,
		IsPinned:        t.IsPinned,
		IsResolved:      t.IsResolved,
		ResolvedReplyId: t.ResolvedReplyID,
		IsSubscribed:    t.IsSubscribed,
		HasVoted:        t.HasVoted,
		UserVoteType:    t.UserVoteType,
		CreatedAt:       t.CreatedAt.Unix(),
		UpdatedAt:       t.UpdatedAt.Unix(),
		Category:        t.Category,
	}

	if t.DeletedAt != nil {
		thread.DeletedAt = t.DeletedAt.Unix()
	}

	thread.LastActivityAt = strconv.FormatInt(t.LastActivityAt.Unix(), 10)

	return thread
}

func (h *Handler) toProtoReply(r *model.Reply) *discussion.Reply {
	reply := &discussion.Reply{
		Id:           r.ID.Hex(),
		ThreadId:     r.ThreadID,
		TenantId:     r.TenantID,
		AuthorId:     r.AuthorID,
		AuthorName:   r.AuthorName,
		AuthorAvatar: r.AuthorAvatar,
		Content:      r.Content,
		Upvotes:      r.Upvotes,
		Downvotes:    r.Downvotes,
		IsAccepted:   r.IsAccepted,
		HasVoted:     r.HasVoted,
		UserVoteType: r.UserVoteType,
		CreatedAt:    r.CreatedAt.Unix(),
		UpdatedAt:    r.UpdatedAt.Unix(),
	}

	if r.DeletedAt != nil {
		reply.DeletedAt = r.DeletedAt.Unix()
	}

	// Convert nested replies
	if len(r.Replies) > 0 {
		reply.Replies = make([]*discussion.Reply, len(r.Replies))
		for i, nested := range r.Replies {
			reply.Replies[i] = h.toProtoReply(&nested)
		}
	}

	return reply
}

func (h *Handler) enrichThreadWithUserData(ctx context.Context, thread *model.Thread) {
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return
	}

	// Get user vote
	vote, _ := h.repo.GetUserVote(ctx, userID, "thread", thread.ID.Hex())
	thread.HasVoted = vote != 0
	thread.UserVoteType = vote

	// Check subscription
	subscribed, _ := h.repo.IsSubscribed(ctx, userID, thread.ID.Hex())
	thread.IsSubscribed = subscribed
}

func (h *Handler) enrichReplyWithUserData(ctx context.Context, reply *model.Reply) {
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return
	}

	vote, _ := h.repo.GetUserVote(ctx, userID, "reply", reply.ID.Hex())
	reply.HasVoted = vote != 0
	reply.UserVoteType = vote
}
