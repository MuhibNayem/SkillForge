package notification

import (
	"context"
	"errors"
	"time"

	notificationpb "github.com/amnayem/skillforge/shared/pb/notification"
	"github.com/amnayem/skillforge/shared/pkg/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Handler implements notificationpb.NotificationServiceServer.
type Handler struct {
	notificationpb.UnimplementedNotificationServiceServer
	repo Repository
}

func NewHandler(repo Repository) *Handler {
	return &Handler{repo: repo}
}

// UpdatePreferences stores the user's notification opt-in settings.
// Users can only update their own preferences; admins may update any user.
func (h *Handler) UpdatePreferences(ctx context.Context, req *notificationpb.UpdatePreferencesRequest) (*notificationpb.PreferencesResponse, error) {
	callerID, err := auth.GetUserID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}
	// Enforce ownership: users may only touch their own prefs unless they are an admin.
	targetID := req.UserId
	if targetID == "" {
		targetID = callerID
	}
	if targetID != callerID {
		if err := auth.RequireRole(ctx, "tenant_admin", "super_admin"); err != nil {
			return nil, err
		}
	}

	p, err := h.repo.UpsertPreferences(ctx, PreferencesRecord{
		UserID:             targetID,
		EmailCourseUpdates: req.EmailCourseUpdates,
		EmailMarketing:     req.EmailMarketing,
		InAppMentions:      req.InAppMentions,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "upsert preferences: %v", err)
	}
	return toProtoPreferences(p), nil
}

// GetPreferences retrieves notification opt-in settings for a user.
func (h *Handler) GetPreferences(ctx context.Context, req *notificationpb.GetPreferencesRequest) (*notificationpb.PreferencesResponse, error) {
	callerID, err := auth.GetUserID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}
	targetID := req.UserId
	if targetID == "" {
		targetID = callerID
	}
	if targetID != callerID {
		if err := auth.RequireRole(ctx, "tenant_admin", "super_admin"); err != nil {
			return nil, err
		}
	}

	p, err := h.repo.GetPreferences(ctx, targetID)
	if err != nil {
		if errors.Is(err, ErrPreferencesNotFound) {
			// Return defaults (all true) rather than a 404.
			return toProtoPreferences(PreferencesRecord{
				UserID:             targetID,
				EmailCourseUpdates: true,
				EmailMarketing:     false,
				InAppMentions:      true,
				UpdatedAt:          time.Now(),
			}), nil
		}
		return nil, status.Errorf(codes.Internal, "get preferences: %v", err)
	}
	return toProtoPreferences(p), nil
}

func toProtoPreferences(p PreferencesRecord) *notificationpb.PreferencesResponse {
	return &notificationpb.PreferencesResponse{
		Preferences: &notificationpb.UserPreferences{
			UserId:             p.UserID,
			EmailCourseUpdates: p.EmailCourseUpdates,
			EmailMarketing:     p.EmailMarketing,
			InAppMentions:      p.InAppMentions,
			UpdatedAt:          timestamppb.New(p.UpdatedAt),
		},
	}
}
