package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/RaviBharathi410/distributedtrace/internal/auth"
	"github.com/RaviBharathi410/distributedtrace/internal/domain"
	"github.com/RaviBharathi410/distributedtrace/internal/repository/postgres"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type AuthHandler struct {
	userRepo     *postgres.UserRepository
	orgRepo      *postgres.OrganizationRepository
	apiKeyRepo   *postgres.APIKeyRepository
	tokenService *auth.TokenService
}

func NewAuthHandler(
	userRepo *postgres.UserRepository,
	orgRepo *postgres.OrganizationRepository,
	apiKeyRepo *postgres.APIKeyRepository,
	tokenService *auth.TokenService,
) *AuthHandler {
	return &AuthHandler{
		userRepo:     userRepo,
		orgRepo:      orgRepo,
		apiKeyRepo:   apiKeyRepo,
		tokenService: tokenService,
	}
}

type RegisterRequest struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	Organization string `json:"organization"`
}

type RegisterResponse struct {
	Token        string               `json:"token"`
	User         domain.User          `json:"user"`
	Organization domain.Organization  `json:"organization"`
	APIKey       string               `json:"api_key"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	if req.Email == "" || req.Password == "" {
		WriteError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	// Check if user already exists
	existing, err := h.userRepo.GetByEmail(r.Context(), req.Email)
	if err == nil && existing != nil {
		WriteError(w, http.StatusConflict, "user with this email already exists")
		return
	}

	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	orgName := req.Organization
	if orgName == "" {
		orgName = "Default Organization"
	}

	now := time.Now().UTC()
	orgID := uuid.New()
	org := domain.Organization{
		ID:        orgID,
		Name:      orgName,
		Plan:      "self-hosted",
		CreatedAt: now,
	}

	if err := h.orgRepo.Create(r.Context(), &org); err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to create organization: "+err.Error())
		return
	}

	userID := uuid.New()
	user := domain.User{
		ID:           userID,
		Email:        req.Email,
		PasswordHash: passwordHash,
		OrgID:        orgID,
		Role:         domain.RoleOwner,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := h.userRepo.Create(r.Context(), &user); err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to create user: "+err.Error())
		return
	}

	// Generate default API key for the organization
	fullKey, keyHash, err := auth.GenerateAPIKey(true)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to generate API key")
		return
	}

	apiKeyID := uuid.New()
	apiKey := domain.APIKey{
		ID:        apiKeyID,
		OrgID:     orgID,
		Name:      "Default Ingestion Key",
		KeyHash:   keyHash,
		Prefix:    fullKey[:10],
		CreatedBy: userID,
		CreatedAt: now,
	}
	_ = h.apiKeyRepo.Create(r.Context(), &apiKey)

	// Generate JWT token valid for 7 days
	token, err := h.tokenService.GenerateToken(userID.String(), orgID.String(), string(domain.RoleOwner), 7*24*time.Hour)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to issue authentication token")
		return
	}

	WriteJSON(w, http.StatusCreated, RegisterResponse{
		Token:        token,
		User:         user,
		Organization: org,
		APIKey:       fullKey,
	})
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token        string              `json:"token"`
	User         domain.User         `json:"user"`
	Organization domain.Organization `json:"organization"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	user, err := h.userRepo.GetByEmail(r.Context(), req.Email)
	if err != nil || user == nil {
		auth.DummyVerifyPassword()
		WriteError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	if err := auth.VerifyPassword(req.Password, user.PasswordHash); err != nil {
		WriteError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	org, err := h.orgRepo.GetByID(r.Context(), user.OrgID)
	if err != nil || org == nil {
		WriteError(w, http.StatusInternalServerError, "failed to load organization")
		return
	}

	token, err := h.tokenService.GenerateToken(user.ID.String(), user.OrgID.String(), string(user.Role), 7*24*time.Hour)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to issue authentication token")
		return
	}

	WriteJSON(w, http.StatusOK, LoginResponse{
		Token:        token,
		User:         *user,
		Organization: *org,
	})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userIDStr := auth.GetUserID(r.Context())
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, "invalid session user id")
		return
	}

	user, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil || user == nil {
		WriteError(w, http.StatusNotFound, "user not found")
		return
	}

	org, err := h.orgRepo.GetByID(r.Context(), user.OrgID)
	if err != nil || org == nil {
		WriteError(w, http.StatusNotFound, "organization not found")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"user":         user,
		"organization": org,
	})
}

type CreateAPIKeyRequest struct {
	Name string `json:"name"`
	Prod bool   `json:"prod"`
}

type CreateAPIKeyResponse struct {
	ID        uuid.UUID `json:"id"`
	Key       string    `json:"key"`
	Prefix    string    `json:"prefix"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

func (h *AuthHandler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	var req CreateAPIKeyRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	if req.Name == "" {
		req.Name = "API Key"
	}

	orgIDStr := auth.GetOrgID(r.Context())
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid organization id")
		return
	}

	userIDStr := auth.GetUserID(r.Context())
	userID, _ := uuid.Parse(userIDStr)

	fullKey, keyHash, err := auth.GenerateAPIKey(req.Prod)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to generate key: "+err.Error())
		return
	}

	keyID := uuid.New()
	now := time.Now().UTC()
	key := domain.APIKey{
		ID:        keyID,
		OrgID:     orgID,
		Name:      req.Name,
		KeyHash:   keyHash,
		Prefix:    fullKey[:10],
		CreatedBy: userID,
		CreatedAt: now,
	}

	if err := h.apiKeyRepo.Create(r.Context(), &key); err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to save API key: "+err.Error())
		return
	}

	WriteJSON(w, http.StatusCreated, CreateAPIKeyResponse{
		ID:        keyID,
		Key:       fullKey,
		Prefix:    key.Prefix,
		Name:      key.Name,
		CreatedAt: now,
	})
}

func (h *AuthHandler) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	orgIDStr := auth.GetOrgID(r.Context())
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid organization id")
		return
	}

	keys, err := h.apiKeyRepo.ListByOrg(r.Context(), orgID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to list API keys: "+err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, keys)
}

func (h *AuthHandler) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	keyIDStr := chi.URLParam(r, "id")
	keyID, err := uuid.Parse(keyIDStr)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid key id")
		return
	}

	if err := h.apiKeyRepo.Revoke(r.Context(), keyID); err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to revoke key: "+err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}
