package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/RaviBharathi410/distributedtrace/internal/domain"
)

func setupTestTokenService(t *testing.T) *TokenService {
	svc, err := NewTokenServiceFromPaths("", "", "test-issuer", "test")
	if err != nil {
		t.Fatalf("failed to create test token service: %v", err)
	}
	return svc
}

func TestRequireUserAuth(t *testing.T) {
	svc := setupTestTokenService(t)
	mw := RequireUserAuth(svc)

	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		orgID := GetOrgID(r.Context())
		userID := GetUserID(r.Context())
		role := GetRole(r.Context())

		if orgID != "org-a-111" {
			t.Errorf("expected org-a-111, got %q", orgID)
		}
		if userID != "user-123" {
			t.Errorf("expected user-123, got %q", userID)
		}
		if role != "owner" {
			t.Errorf("expected owner, got %q", role)
		}
		w.WriteHeader(http.StatusOK)
	})

	h := mw(dummyHandler)

	t.Run("Missing Authorization Header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/traces", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized, got %d", rec.Code)
		}
	})

	t.Run("Invalid Bearer Format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/traces", nil)
		req.Header.Set("Authorization", "Basic 12345")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized, got %d", rec.Code)
		}
	})

	t.Run("Valid Token Injects Context", func(t *testing.T) {
		token, err := svc.GenerateToken("user-123", "org-a-111", "owner", 1*time.Hour)
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/traces", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", rec.Code)
		}
	})
}

func TestRequireRole(t *testing.T) {
	adminOnlyHandler := RequireRole(domain.RoleAdmin)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	memberAllowedHandler := RequireRole(domain.RoleMember, domain.RoleAdmin)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("RoleOwner has superuser bypass on Admin routes", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), RoleKey, string(domain.RoleOwner))
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/api-keys", nil).WithContext(ctx)
		rec := httptest.NewRecorder()

		adminOnlyHandler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected RoleOwner to be allowed on Admin routes, got %d", rec.Code)
		}
	})

	t.Run("RoleAdmin is allowed on Admin routes", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), RoleKey, string(domain.RoleAdmin))
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/api-keys", nil).WithContext(ctx)
		rec := httptest.NewRecorder()

		adminOnlyHandler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected RoleAdmin to be allowed on Admin routes, got %d", rec.Code)
		}
	})

	t.Run("RoleMember is forbidden on Admin-only routes", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), RoleKey, string(domain.RoleMember))
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/api-keys", nil).WithContext(ctx)
		rec := httptest.NewRecorder()

		adminOnlyHandler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("expected RoleMember to be 403 Forbidden on Admin-only routes, got %d", rec.Code)
		}
	})

	t.Run("RoleReadOnly is forbidden on Admin-only routes", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), RoleKey, string(domain.RoleReadOnly))
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/api-keys", nil).WithContext(ctx)
		rec := httptest.NewRecorder()

		adminOnlyHandler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("expected RoleReadOnly to be 403 Forbidden on Admin-only routes, got %d", rec.Code)
		}
	})

	t.Run("RoleMember is allowed on Member routes", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), RoleKey, string(domain.RoleMember))
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/anomalies/123/status", nil).WithContext(ctx)
		rec := httptest.NewRecorder()

		memberAllowedHandler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected RoleMember to be allowed, got %d", rec.Code)
		}
	})

	t.Run("RoleReadOnly is forbidden on Member mutation routes", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), RoleKey, string(domain.RoleReadOnly))
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/anomalies/123/status", nil).WithContext(ctx)
		rec := httptest.NewRecorder()

		memberAllowedHandler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("expected RoleReadOnly to be 403 Forbidden, got %d", rec.Code)
		}
	})
}
