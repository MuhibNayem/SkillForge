package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	auth "github.com/amnayem/skillforge/services/auth"
	"github.com/amnayem/skillforge/shared/pb/authpb"
)

// mockRepo implements auth.Repository for unit testing.
type mockRepo struct {
	users map[string]auth.UserRecord
}

func newMockRepo() *mockRepo {
	return &mockRepo{users: make(map[string]auth.UserRecord)}
}

func (m *mockRepo) CreateUser(ctx context.Context, tenantID, email, passwordHash, firstName, lastName string) (auth.UserRecord, error) {
	if _, exists := m.users[email]; exists {
		return auth.UserRecord{}, auth.ErrUserAlreadyExists
	}
	u := auth.UserRecord{
		ID: "test-user-id", TenantID: tenantID, Email: email, PasswordHash: passwordHash,
		FirstName: firstName, LastName: lastName, Role: "student", IsActive: true, CreatedAt: time.Now(),
	}
	m.users[email] = u
	return u, nil
}

func (m *mockRepo) GetUserByEmail(ctx context.Context, email string) (auth.UserRecord, error) {
	u, ok := m.users[email]
	if !ok {
		return auth.UserRecord{}, auth.ErrUserNotFound
	}
	return u, nil
}

func (m *mockRepo) GetUserByID(ctx context.Context, id string) (auth.UserRecord, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return auth.UserRecord{}, auth.ErrUserNotFound
}

func (m *mockRepo) StoreRefreshToken(ctx context.Context, userID, token string, ttl time.Duration) error {
	return nil
}

func (m *mockRepo) ValidateRefreshToken(ctx context.Context, userID, token string) (bool, error) {
	return true, nil
}

func (m *mockRepo) DeleteRefreshToken(ctx context.Context, userID string) error {
	return nil
}

func TestRegister(t *testing.T) {
	repo := newMockRepo()
	h := auth.NewHandler("test-secret", repo)
	resp, err := h.Register(context.Background(), &authpb.RegisterRequest{
		Email: "test@example.com", Password: "password123",
		FirstName: "Test", LastName: "User",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if resp.UserId == "" {
		t.Error("expected non-empty user_id")
	}
}

func TestRegister_Duplicate(t *testing.T) {
	repo := newMockRepo()
	h := auth.NewHandler("test-secret", repo)
	_, _ = h.Register(context.Background(), &authpb.RegisterRequest{
		Email: "dup@example.com", Password: "password123", FirstName: "A", LastName: "B",
	})
	_, err := h.Register(context.Background(), &authpb.RegisterRequest{
		Email: "dup@example.com", Password: "password123", FirstName: "A", LastName: "B",
	})
	if err == nil {
		t.Error("expected error for duplicate registration")
	}
}

func TestLogin(t *testing.T) {
	repo := newMockRepo()
	h := auth.NewHandler("test-secret", repo)
	// Register first
	_, _ = h.Register(context.Background(), &authpb.RegisterRequest{
		Email: "login@example.com", Password: "password123", FirstName: "Test", LastName: "User",
	})
	resp, err := h.Login(context.Background(), &authpb.LoginRequest{
		Email: "login@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected non-empty access_token")
	}
	if resp.ExpiresAt == 0 {
		t.Error("expected non-zero expires_at")
	}
}

func TestLogin_NotFound(t *testing.T) {
	repo := newMockRepo()
	h := auth.NewHandler("test-secret", repo)
	_, err := h.Login(context.Background(), &authpb.LoginRequest{
		Email: "nobody@example.com", Password: "password123",
	})
	if err == nil {
		t.Error("expected error for non-existent user")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	repo := newMockRepo()
	h := auth.NewHandler("test-secret", repo)
	_, _ = h.Register(context.Background(), &authpb.RegisterRequest{
		Email: "wrong@example.com", Password: "correct", FirstName: "A", LastName: "B",
	})
	_, err := h.Login(context.Background(), &authpb.LoginRequest{
		Email: "wrong@example.com", Password: "incorrect",
	})
	if err == nil {
		t.Error("expected error for wrong password")
	}
}

func TestLogin_Empty(t *testing.T) {
	repo := newMockRepo()
	h := auth.NewHandler("test-secret", repo)
	_, err := h.Login(context.Background(), &authpb.LoginRequest{})
	if err == nil {
		t.Error("expected error for empty credentials")
	}
}

func TestValidateToken(t *testing.T) {
	repo := newMockRepo()
	h := auth.NewHandler("test-secret", repo)
	_, _ = h.Register(context.Background(), &authpb.RegisterRequest{
		Email: "validate@example.com", Password: "password123", FirstName: "V", LastName: "T",
	})
	loginResp, _ := h.Login(context.Background(), &authpb.LoginRequest{
		Email: "validate@example.com", Password: "password123",
	})
	resp, err := h.ValidateToken(context.Background(), &authpb.ValidateTokenRequest{
		AccessToken: loginResp.AccessToken,
	})
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	if !resp.IsValid {
		t.Error("expected token to be valid")
	}
	if resp.UserId == "" {
		t.Error("expected non-empty user_id")
	}
}

func TestValidateToken_Invalid(t *testing.T) {
	repo := newMockRepo()
	h := auth.NewHandler("test-secret", repo)
	resp, err := h.ValidateToken(context.Background(), &authpb.ValidateTokenRequest{
		AccessToken: "invalid-token",
	})
	if err != nil {
		t.Fatalf("ValidateToken should not error: %v", err)
	}
	if resp.IsValid {
		t.Error("expected token to be invalid")
	}
}

// verify errors package works for sentinel errors
func TestSentinelErrors(t *testing.T) {
	if !errors.Is(auth.ErrUserNotFound, auth.ErrUserNotFound) {
		t.Error("sentinel error mismatch")
	}
}
