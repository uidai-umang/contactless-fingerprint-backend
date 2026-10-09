package auth

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const AccessTokenTTL = 15 * time.Minute

type Claims struct {
	OperatorID string `json:"operator_id"`
	jwt.RegisteredClaims
}

func accessSecret() ([]byte, error) {
	secret := os.Getenv("JWT_ACCESS_SECRET")
	if secret == "" {
		return nil, errors.New("JWT_ACCESS_SECRET is not set")
	}
	return []byte(secret), nil
}

func GenerateAccessToken(operatorID string) (string, error) {
	secret, err := accessSecret()
	if err != nil {
		return "", err
	}

	now := time.Now()
	claims := Claims{
		OperatorID: operatorID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   operatorID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenTTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func ValidateAccessToken(tokenString string) (*Claims, error) {
	secret, err := accessSecret()
	if err != nil {
		return nil, err
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
