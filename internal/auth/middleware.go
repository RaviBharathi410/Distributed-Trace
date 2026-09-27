package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/RaviBharathi410/distributedtrace/internal/domain"
	"github.com/RaviBharathi410/distributedtrace/internal/repository/postgres"
)

type contextKey string

const (
	OrgIDKey  contextKey = "org_id"
	UserIDKey contextKey = "user_id"
	RoleKey   contextKey = "role"
)

func GetOrgID(ctx context.Context) string {
	if val, ok := ctx.Value(OrgIDKey).(string); ok {
		return val
	}
	return ""
}

func GetUserID(ctx context.Context) string {
	if val, ok := ctx.Value(UserIDKey).(string); ok {
		return val
	}
	return ""
}

func GetRole(ctx context.Context) string {
	if val, ok := ctx.Value(RoleKey).(string); ok {
		return val
	}
	return ""
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

// RequireUserAuth validates JWT from Authorization: Bearer <token> for user-facing API routes.
func RequireUserAuth(tokenService *TokenService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeJSONError(w, http.StatusUnauthorized, "missing Authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				writeJSONError(w, http.StatusUnauthorized, "invalid Authorization header format, expected 'Bearer <token>'")
				return
			}

			tokenString := parts[1]
			claims, err := tokenService.ValidateToken(tokenString)
			if err != nil {
				writeJSONError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), OrgIDKey, claims.OrgID)
			ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, RoleKey, claims.Role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireIngestAuth supports dual-mode auth: API Key (X-API-Key or Bearer dt_...) OR User JWT.
func RequireIngestAuth(apiKeyRepo *postgres.APIKeyRepository, tokenService *TokenService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get("X-API-Key")

			if apiKey == "" {
				authHeader := r.Header.Get("Authorization")
				if authHeader != "" {
					parts := strings.SplitN(authHeader, " ", 2)
					if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
						tokenCandidate := parts[1]
						if strings.HasPrefix(tokenCandidate, "dt_live_") || strings.HasPrefix(tokenCandidate, "dt_test_") {
							apiKey = tokenCandidate
						} else {
							// Try validating as JWT
							claims, err := tokenService.ValidateToken(tokenCandidate)
							if err == nil {
								ctx := context.WithValue(r.Context(), OrgIDKey, claims.OrgID)
								ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
								ctx = context.WithValue(ctx, RoleKey, claims.Role)
								next.ServeHTTP(w, r.WithContext(ctx))
								return
							}
						}
					}
				}
			}

			if apiKey != "" {
				hash := HashAPIKey(apiKey)
				keyRecord, err := apiKeyRepo.GetByHash(r.Context(), hash)
				if err != nil || keyRecord == nil {
					writeJSONError(w, http.StatusUnauthorized, "invalid api key")
					return
				}
				if keyRecord.RevokedAt != nil {
					writeJSONError(w, http.StatusUnauthorized, "api key has been revoked")
					return
				}

				// Asynchronously update last used timestamp
				go func() {
					_ = apiKeyRepo.UpdateLastUsed(context.Background(), keyRecord.ID)
				}()

				ctx := context.WithValue(r.Context(), OrgIDKey, keyRecord.OrgID.String())
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			writeJSONError(w, http.StatusUnauthorized, "authentication required (provide X-API-Key or Bearer token)")
		})
	}
}

// RequireRole enforces role-based access control.
func RequireRole(roles ...domain.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userRole := domain.Role(GetRole(r.Context()))
			for _, allowed := range roles {
				if userRole == allowed || userRole == domain.RoleOwner {
					next.ServeHTTP(w, r)
					return
				}
			}
			writeJSONError(w, http.StatusForbidden, "insufficient permissions for this action")
		})
	}
}
