package user

import (
	"context"
	"errors"

	"github.com/amnayem/skillforge/shared/pb/userpb"
	"github.com/amnayem/skillforge/shared/pkg/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	userpb.UnimplementedUserServiceServer
	repo Repository
}

func NewHandler(repo Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) CreateUser(ctx context.Context, req *userpb.CreateUserRequest) (*userpb.UserResponse, error) {
	if err := auth.RequireRole(ctx, "tenant_admin", "super_admin"); err != nil {
		return nil, err
	}
	if req.Email == "" {
		return nil, status.Errorf(codes.InvalidArgument, "email is required")
	}
	role := req.Role
	if role == "" {
		role = "student"
	}
	u, err := h.repo.Create(ctx, req.TenantId, req.Email, req.FirstName, req.LastName, role)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create user: %v", err)
	}
	return toProto(u), nil
}

func (h *Handler) GetUser(ctx context.Context, req *userpb.GetUserRequest) (*userpb.UserResponse, error) {
	if req.UserId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "user_id is required")
	}
	u, err := h.repo.GetByID(ctx, req.UserId)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, status.Errorf(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "get user: %v", err)
	}
	return toProto(u), nil
}

func (h *Handler) UpdateUser(ctx context.Context, req *userpb.UpdateUserRequest) (*userpb.UserResponse, error) {
	if req.UserId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "user_id is required")
	}
	u, err := h.repo.Update(ctx, req.UserId, req.FirstName, req.LastName, req.AvatarUrl, req.Bio)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, status.Errorf(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "update user: %v", err)
	}
	return toProto(u), nil
}

func (h *Handler) DeleteUser(ctx context.Context, req *userpb.DeleteUserRequest) (*userpb.DeleteUserResponse, error) {
	if err := auth.RequireRole(ctx, "tenant_admin", "super_admin"); err != nil {
		return nil, err
	}
	if req.UserId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "user_id is required")
	}
	if err := h.repo.Delete(ctx, req.UserId); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, status.Errorf(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "delete user: %v", err)
	}
	return &userpb.DeleteUserResponse{Success: true}, nil
}

func (h *Handler) ListUsers(ctx context.Context, req *userpb.ListUsersRequest) (*userpb.ListUsersResponse, error) {
	if err := auth.RequireRole(ctx, "tenant_admin", "super_admin"); err != nil {
		return nil, err
	}
	users, total, err := h.repo.List(ctx, req.TenantId, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list users: %v", err)
	}
	resp := &userpb.ListUsersResponse{Total: int32(total)}
	for _, u := range users {
		resp.Users = append(resp.Users, toProto(u))
	}
	return resp, nil
}

func toProto(u UserRecord) *userpb.UserResponse {
	return &userpb.UserResponse{
		UserId:    u.ID,
		Email:     u.Email,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Role:      u.Role,
		TenantId:  u.TenantID,
		AvatarUrl: u.AvatarURL,
		Bio:       u.Bio,
		CreatedAt: u.CreatedAt.Unix(),
		UpdatedAt: u.UpdatedAt.Unix(),
	}
}
