package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type contextKey string

const (
	UserIDKey   contextKey = "user_id"
	TenantIDKey contextKey = "tenant_id"
	RoleKey     contextKey = "role"
)

// ParseToken validates a JWT and returns the claims
func ParseToken(tokenStr, secret string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}

// UnaryInterceptor intercepts gRPC requests, parses the JWT from the cookie metadata, and injects claims into Context.
func UnaryInterceptor(jwtSecret string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// By-pass auth for public endpoints (Login, Register, ValidateToken are called pre-auth)
		if strings.Contains(info.FullMethod, "AuthService/Login") ||
			strings.Contains(info.FullMethod, "AuthService/Register") ||
			strings.Contains(info.FullMethod, "AuthService/ValidateToken") {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		// Look for the "cookie" metadata which gRPC-Gateway passes forward
		cookies := md.Get("cookie")
		var tokenStr string
		for _, c := range cookies {
			header := http.Header{}
			header.Add("Cookie", c)
			req := http.Request{Header: header}
			cookie, err := req.Cookie("access_token")
			if err == nil && cookie != nil {
				tokenStr = cookie.Value
				break
			}
		}

		// Also check Authorization header for Bearer token fallback
		if tokenStr == "" {
			authHeader := md.Get("authorization")
			if len(authHeader) > 0 && strings.HasPrefix(authHeader[0], "Bearer ") {
				tokenStr = strings.TrimPrefix(authHeader[0], "Bearer ")
			}
		}

		if tokenStr == "" {
			return nil, status.Error(codes.Unauthenticated, "missing access token")
		}

		claims, err := ParseToken(tokenStr, jwtSecret)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
		}

		// Inject claims into context
		if sub, ok := claims["sub"].(string); ok {
			ctx = context.WithValue(ctx, UserIDKey, sub)
		}
		if tenant, ok := claims["tenant_id"].(string); ok {
			ctx = context.WithValue(ctx, TenantIDKey, tenant)
		}
		if role, ok := claims["role"].(string); ok {
			ctx = context.WithValue(ctx, RoleKey, role)
		}

		return handler(ctx, req)
	}
}

// GetUserID retrieves the user ID from context.
func GetUserID(ctx context.Context) (string, error) {
	if val, ok := ctx.Value(UserIDKey).(string); ok && val != "" {
		return val, nil
	}
	return "", status.Error(codes.Unauthenticated, "user_id not found in context")
}

// GetTenantID retrieves the tenant ID from context.
func GetTenantID(ctx context.Context) (string, error) {
	if val, ok := ctx.Value(TenantIDKey).(string); ok && val != "" {
		return val, nil
	}
	return "", status.Error(codes.Unauthenticated, "tenant_id not found in context")
}

// GetRole retrieves the role from context.
func GetRole(ctx context.Context) (string, error) {
	if val, ok := ctx.Value(RoleKey).(string); ok && val != "" {
		return val, nil
	}
	return "", status.Error(codes.Unauthenticated, "role not found in context")
}

// RequireRole returns PermissionDenied if the caller's role is not in the allowed list.
func RequireRole(ctx context.Context, allowed ...string) error {
	role, err := GetRole(ctx)
	if err != nil {
		return err
	}
	for _, a := range allowed {
		if role == a {
			return nil
		}
	}
	return status.Errorf(codes.PermissionDenied, "role %q is not authorized for this action", role)
}
