package auth

import (
	"context"
	"errors"
	"time"

	"github.com/amnayem/skillforge/shared/pb/authpb"
	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	authpb.UnimplementedAuthServiceServer
	repo      Repository
	jwtSecret []byte
}

func NewHandler(jwtSecret string, repo Repository) *Handler {
	return &Handler{jwtSecret: []byte(jwtSecret), repo: repo}
}

func (h *Handler) Register(ctx context.Context, req *authpb.RegisterRequest) (*authpb.RegisterResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, status.Errorf(codes.InvalidArgument, "email and password are required")
	}

	hash, err := HashPassword(req.Password)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "hash password: %v", err)
	}

	// Default tenant for Phase 1
	tenantID := "00000000-0000-0000-0000-000000000001"

	user, err := h.repo.CreateUser(ctx, tenantID, req.Email, hash, req.FirstName, req.LastName)
	if err != nil {
		if errors.Is(err, ErrUserAlreadyExists) {
			return nil, status.Errorf(codes.AlreadyExists, "email already registered")
		}
		return nil, status.Errorf(codes.Internal, "create user: %v", err)
	}

	return &authpb.RegisterResponse{
		UserId:  user.ID,
		Message: "registration successful",
	}, nil
}

func (h *Handler) Login(ctx context.Context, req *authpb.LoginRequest) (*authpb.LoginResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, status.Errorf(codes.InvalidArgument, "email and password are required")
	}

	user, err := h.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, status.Errorf(codes.NotFound, "invalid credentials")
		}
		return nil, status.Errorf(codes.Internal, "fetch user: %v", err)
	}

	if !CheckPassword(req.Password, user.PasswordHash) {
		return nil, status.Errorf(codes.Unauthenticated, "invalid credentials")
	}

	expiresAt := time.Now().Add(24 * time.Hour)
	claims := jwt.MapClaims{
		"sub":       user.ID,
		"email":     user.Email,
		"role":      user.Role,
		"tenant_id": user.TenantID,
		"exp":       expiresAt.Unix(),
		"iat":       time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString(h.jwtSecret)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "sign token: %v", err)
	}

	// Store refresh token in Redis (7 days TTL)
	refreshToken := "refresh-" + user.ID + "-" + time.Now().Format("20060102150405")
	_ = h.repo.StoreRefreshToken(ctx, user.ID, refreshToken, 7*24*time.Hour)

	return &authpb.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt.Unix(),
	}, nil
}

func (h *Handler) ValidateToken(ctx context.Context, req *authpb.ValidateTokenRequest) (*authpb.ValidateTokenResponse, error) {
	if req.AccessToken == "" {
		return &authpb.ValidateTokenResponse{IsValid: false}, nil
	}

	token, err := jwt.Parse(req.AccessToken, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return h.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return &authpb.ValidateTokenResponse{IsValid: false}, nil
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return &authpb.ValidateTokenResponse{IsValid: false}, nil
	}

	return &authpb.ValidateTokenResponse{
		IsValid:  true,
		UserId:   claims["sub"].(string),
		Email:    claims["email"].(string),
		Role:     claims["role"].(string),
		TenantId: claims["tenant_id"].(string),
	}, nil
}
