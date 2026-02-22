package user_test

import (
	"context"
	"testing"
	"time"

	user "github.com/amnayem/skillforge/services/user"
	"github.com/amnayem/skillforge/shared/pb/userpb"
	"github.com/amnayem/skillforge/shared/pkg/auth"
)

func newAuthCtx(role, userID, tenantID string) context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, auth.RoleKey, role)
	ctx = context.WithValue(ctx, auth.UserIDKey, userID)
	ctx = context.WithValue(ctx, auth.TenantIDKey, tenantID)
	return ctx
}

type mockRepo struct {
	users map[string]user.UserRecord
}

func newMockRepo() *mockRepo {
	return &mockRepo{users: make(map[string]user.UserRecord)}
}

func (m *mockRepo) Create(ctx context.Context, tenantID, email, firstName, lastName, role string) (user.UserRecord, error) {
	u := user.UserRecord{
		ID: "user-" + email, TenantID: tenantID, Email: email, FirstName: firstName,
		LastName: lastName, Role: role, IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	m.users[u.ID] = u
	return u, nil
}

func (m *mockRepo) GetByID(ctx context.Context, id string) (user.UserRecord, error) {
	u, ok := m.users[id]
	if !ok {
		return user.UserRecord{}, user.ErrUserNotFound
	}
	return u, nil
}

func (m *mockRepo) Update(ctx context.Context, id, firstName, lastName, avatarURL, bio string) (user.UserRecord, error) {
	u, ok := m.users[id]
	if !ok {
		return user.UserRecord{}, user.ErrUserNotFound
	}
	u.FirstName = firstName
	u.LastName = lastName
	u.AvatarURL = avatarURL
	u.Bio = bio
	m.users[id] = u
	return u, nil
}

func (m *mockRepo) Delete(ctx context.Context, id string) error {
	if _, ok := m.users[id]; !ok {
		return user.ErrUserNotFound
	}
	delete(m.users, id)
	return nil
}

func (m *mockRepo) List(ctx context.Context, tenantID string, page, pageSize int) ([]user.UserRecord, int, error) {
	var results []user.UserRecord
	for _, u := range m.users {
		if u.TenantID == tenantID {
			results = append(results, u)
		}
	}
	return results, len(results), nil
}

func TestCreateUser(t *testing.T) {
	h := user.NewHandler(newMockRepo())
	ctx := newAuthCtx("tenant_admin", "admin-1", "t1")
	resp, err := h.CreateUser(ctx, &userpb.CreateUserRequest{
		Email: "new@example.com", FirstName: "Jane", LastName: "Doe", Role: "student", TenantId: "t1",
	})
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if resp.Email != "new@example.com" {
		t.Errorf("expected email new@example.com, got %s", resp.Email)
	}
}

func TestCreateUser_EmptyEmail(t *testing.T) {
	h := user.NewHandler(newMockRepo())
	_, err := h.CreateUser(newAuthCtx("tenant_admin", "admin-1", "t1"), &userpb.CreateUserRequest{})
	if err == nil {
		t.Error("expected error for empty email")
	}
}

func TestGetUser(t *testing.T) {
	repo := newMockRepo()
	h := user.NewHandler(repo)
	resp, _ := h.CreateUser(newAuthCtx("tenant_admin", "admin-1", "t1"), &userpb.CreateUserRequest{
		Email: "get@example.com", FirstName: "Get", LastName: "Test", TenantId: "t1",
	})
	got, err := h.GetUser(context.Background(), &userpb.GetUserRequest{UserId: resp.UserId})
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	if got.Email != "get@example.com" {
		t.Errorf("expected get@example.com, got %s", got.Email)
	}
}

func TestGetUser_NotFound(t *testing.T) {
	h := user.NewHandler(newMockRepo())
	_, err := h.GetUser(context.Background(), &userpb.GetUserRequest{UserId: "nonexistent"})
	if err == nil {
		t.Error("expected error for nonexistent user")
	}
}

func TestDeleteUser(t *testing.T) {
	repo := newMockRepo()
	h := user.NewHandler(repo)
	ctx := newAuthCtx("tenant_admin", "admin-1", "t1")
	resp, _ := h.CreateUser(ctx, &userpb.CreateUserRequest{
		Email: "del@example.com", FirstName: "Del", LastName: "Test", TenantId: "t1",
	})
	delResp, err := h.DeleteUser(ctx, &userpb.DeleteUserRequest{UserId: resp.UserId})
	if err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}
	if !delResp.Success {
		t.Error("expected success=true")
	}
}
