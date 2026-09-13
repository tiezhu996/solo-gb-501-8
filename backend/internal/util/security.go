package util

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"sterile-packaging-release-control/backend/internal/constants"
)

type Claims struct {
	UserID      uint           `json:"uid"`
	Username    string         `json:"username"`
	DisplayName string         `json:"displayName"`
	Role        constants.Role `json:"role"`
	jwt.RegisteredClaims
}

func HashPassword(password string) (string, error) {
	if len(password) < 6 {
		return "", errors.New("password must contain at least six characters")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

func ComparePassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func SignToken(secret string, ttl time.Duration, userID uint, username, displayName string, role constants.Role) (string, time.Time, error) {
	now := time.Now()
	expires := now.Add(ttl)
	claims := Claims{
		UserID: userID, Username: username, DisplayName: displayName, Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: "sterile-packaging-release-control", Subject: fmt.Sprint(userID),
			IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(expires),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	return signed, expires, err
}

func ParseToken(secret, tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %s", token.Method.Alg())
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse access token: %w", err)
	}
	if !token.Valid {
		return nil, errors.New("invalid access token")
	}
	return claims, nil
}
