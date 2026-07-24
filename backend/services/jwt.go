package services

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"study_plan_backend/config"
)

type Claims struct {
	UserID uint   `json:"user_id"`
	OpenID string `json:"openid"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// SignToken 为用户签发 JWT
func SignToken(userID uint, openid, role string) (string, error) {
	cfg := config.App
	expiresAt := time.Now().Add(time.Duration(cfg.JWTExpireHours) * time.Hour)
	claims := Claims{
		UserID: userID,
		OpenID: openid,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "study_plan",
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString([]byte(cfg.JWTSecret))
}

// ParseToken 解析并校验 JWT
func ParseToken(tokenStr string) (*Claims, error) {
	cfg := config.App
	claims := &Claims{}
	tok, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(cfg.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}
	if !tok.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
