package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrPasswordTooShort = errors.New("password must be at least 12 characters")
	ErrPasswordTooLong  = errors.New("password must not be longer than 72 characters")
)

func HashPassword(password string) (string, error) {
	if len(password) < 12 {
		return "", ErrPasswordTooShort
	}
	if len(password) > 72 {
		return "", ErrPasswordTooLong
	}

	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func VerifyPassword(password, hash string) error {
	if len(password) < 12 {
		return ErrPasswordTooShort
	}
	if len(password) > 72 {
		return ErrPasswordTooLong
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// DummyVerifyPassword runs a dummy bcrypt comparison to mitigate timing attacks/user enumeration.
func DummyVerifyPassword() {
	dummyHash := "$2a$12$LvyW85g7p19mP6D0p/T.IeX3m6Z/6mNl6d.k8d5.g7a2zP5iTz8qW"
	_ = bcrypt.CompareHashAndPassword([]byte(dummyHash), []byte("dummypassword123"))
}
