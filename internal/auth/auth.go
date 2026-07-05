package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func HashPassword(password string) (string, error) {
	return argon2id.CreateHash(password, argon2id.DefaultParams)
}

func CheckPasswordHash(password, hash string) (bool, error) {
	return argon2id.ComparePasswordAndHash(password, hash)
}

func MakeJWT(userID uuid.UUID, tokenSecret string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "chirpy-access",
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Hour)),
		Subject:   userID.String(),
	})

	return token.SignedString([]byte(tokenSecret))
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	claims := &jwt.RegisteredClaims{}

	_, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		return []byte(tokenSecret), nil
	})
	if err != nil {
		return uuid.UUID{}, err
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.UUID{}, err
	}

	return userID, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	authHeader := headers.Get("Authorization")
	parts := strings.Fields(authHeader)

	if len(parts) != 2 {
		return "", fmt.Errorf("Expected Auth header to have 2 parts for Bearer schema")
	}

	if parts[0] != "Bearer" {
		return "", fmt.Errorf("Expected Auth header to start with 'Bearer' for Bearer schema")
	}

	return parts[1], nil
}

func MakeRefreshToken() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		fmt.Printf("\n Error reading bytes: %v", err)
	}

	return hex.EncodeToString(bytes)
}

func GetAPIKey(headers http.Header) (string, error) {
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("No auth header provided")
	}

	fields := strings.Fields(authHeader)
	if len(fields) != 2 || fields[0] != "ApiKey" {
		return "", fmt.Errorf("Invalid auth header")
	}

	return fields[1], nil
}
