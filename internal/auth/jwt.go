package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid or expired token")
)

type TokenClaims struct {
	UserID string `json:"user_id"`
	OrgID  string `json:"org_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type TokenService struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	issuer     string
}

func NewTokenService(privateKey *rsa.PrivateKey, publicKey *rsa.PublicKey, issuer string) *TokenService {
	return &TokenService{
		privateKey: privateKey,
		publicKey:  publicKey,
		issuer:     issuer,
	}
}

// NewTokenServiceFromPaths loads RSA keys from files, or generates ephemeral dev keys only if env is development or test.
func NewTokenServiceFromPaths(privatePath, publicPath, issuer, env string) (*TokenService, error) {
	if privatePath != "" && publicPath != "" {
		privBytes, err := os.ReadFile(privatePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read private key: %w", err)
		}
		pubBytes, err := os.ReadFile(publicPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read public key: %w", err)
		}

		privKey, err := jwt.ParseRSAPrivateKeyFromPEM(privBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse RSA private key: %w", err)
		}
		pubKey, err := jwt.ParseRSAPublicKeyFromPEM(pubBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse RSA public key: %w", err)
		}

		return NewTokenService(privKey, pubKey, issuer), nil
	}

	// Hard-gate ephemeral fallback behind explicit development or test environment
	if env != "development" && env != "test" {
		return nil, fmt.Errorf("RSA key paths (JWT_PRIVATE_KEY_PATH, JWT_PUBLIC_KEY_PATH) are strictly required in environment: %q", env)
	}

	// Ephemeral development key pair
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ephemeral RSA key: %w", err)
	}
	return NewTokenService(privKey, &privKey.PublicKey, issuer), nil
}

func (s *TokenService) GenerateToken(userID, orgID, role string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := TokenClaims{
		UserID: userID,
		OrgID:  orgID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    s.issuer,
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(s.privateKey)
}

func (s *TokenService) ValidateToken(tokenString string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.publicKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*TokenClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}
