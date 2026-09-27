package auth

import (
	"strings"
	"testing"
	"time"
)

func TestPasswordHashing(t *testing.T) {
	t.Run("Valid password hashes and verifies", func(t *testing.T) {
		pwd := "SuperSecure1234!"
		hash, err := HashPassword(pwd)
		if err != nil {
			t.Fatalf("HashPassword failed: %v", err)
		}

		if err := VerifyPassword(pwd, hash); err != nil {
			t.Errorf("VerifyPassword failed for correct password: %v", err)
		}

		if err := VerifyPassword("WrongPassword123!", hash); err == nil {
			t.Errorf("VerifyPassword succeeded for incorrect password")
		}
	})

	t.Run("Rejects passwords shorter than 12 characters", func(t *testing.T) {
		_, err := HashPassword("short123")
		if err == nil {
			t.Errorf("expected error for password < 12 characters")
		}
	})

	t.Run("Rejects passwords longer than 72 characters", func(t *testing.T) {
		longPwd := strings.Repeat("A", 75)
		_, err := HashPassword(longPwd)
		if err == nil {
			t.Errorf("expected error for password > 72 characters")
		}
	})

	t.Run("DummyVerifyPassword executes without panicking", func(t *testing.T) {
		DummyVerifyPassword()
	})
}

func TestAPIKeyGeneration(t *testing.T) {
	t.Run("Generates production key with dt_live_ prefix", func(t *testing.T) {
		key, hash, err := GenerateAPIKey(true)
		if err != nil {
			t.Fatalf("GenerateAPIKey failed: %v", err)
		}

		if !strings.HasPrefix(key, "dt_live_") {
			t.Errorf("expected dt_live_ prefix, got %s", key)
		}

		if hash == "" {
			t.Errorf("expected non-empty key hash")
		}

		// Recomputing hash must match
		expectedHash := HashAPIKey(key)
		if hash != expectedHash {
			t.Errorf("hash mismatch: got %s, expected %s", hash, expectedHash)
		}
	})

	t.Run("Generates test key with dt_test_ prefix", func(t *testing.T) {
		key, hash, err := GenerateAPIKey(false)
		if err != nil {
			t.Fatalf("GenerateAPIKey failed: %v", err)
		}

		if !strings.HasPrefix(key, "dt_test_") {
			t.Errorf("expected dt_test_ prefix, got %s", key)
		}

		if hash == "" {
			t.Errorf("expected non-empty key hash")
		}
	})
}

func TestTokenService(t *testing.T) {
	t.Run("Generates and validates token in test env", func(t *testing.T) {
		svc, err := NewTokenServiceFromPaths("", "", "test-issuer", "test")
		if err != nil {
			t.Fatalf("failed to initialize TokenService: %v", err)
		}

		token, err := svc.GenerateToken("user-1", "org-1", "admin", 1*time.Hour)
		if err != nil {
			t.Fatalf("GenerateToken failed: %v", err)
		}

		claims, err := svc.ValidateToken(token)
		if err != nil {
			t.Fatalf("ValidateToken failed: %v", err)
		}

		if claims.UserID != "user-1" || claims.OrgID != "org-1" || claims.Role != "admin" {
			t.Errorf("unexpected claims: %+v", claims)
		}
	})

	t.Run("Rejects expired token", func(t *testing.T) {
		svc, _ := NewTokenServiceFromPaths("", "", "test-issuer", "test")
		// Generate with negative duration
		token, err := svc.GenerateToken("user-1", "org-1", "admin", -1*time.Hour)
		if err != nil {
			t.Fatalf("GenerateToken failed: %v", err)
		}

		_, err = svc.ValidateToken(token)
		if err == nil {
			t.Errorf("expected validation to fail for expired token")
		}
	})

	t.Run("Fails fast in production when PEM paths are empty", func(t *testing.T) {
		_, err := NewTokenServiceFromPaths("", "", "test-issuer", "production")
		if err == nil {
			t.Errorf("expected error in production when PEM paths are empty")
		}
	})
}
