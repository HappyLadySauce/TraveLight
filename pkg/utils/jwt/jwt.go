package jwt

import (
	"errors"
	"fmt"
	"time"

	gjwt "github.com/golang-jwt/jwt/v5"
)

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

// Claims defines token payload.
// Claims 定义 Token 负载。
type Claims struct {
	UserID    uint64 `json:"user_id"`
	Username  string `json:"username"`
	TokenType string `json:"token_type"`
	gjwt.RegisteredClaims
}

// GenerateToken creates a signed jwt token.
// GenerateToken 创建已签名 JWT。
func GenerateToken(secret string, userID uint64, username string, tokenType string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:    userID,
		Username:  username,
		TokenType: tokenType,
		RegisteredClaims: gjwt.RegisteredClaims{
			IssuedAt:  gjwt.NewNumericDate(now),
			ExpiresAt: gjwt.NewNumericDate(now.Add(ttl)),
			NotBefore: gjwt.NewNumericDate(now),
			Subject:   fmt.Sprintf("%d", userID),
		},
	}
	token := gjwt.NewWithClaims(gjwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("sign token failed: %w", err)
	}
	return signed, nil
}

// ParseToken validates token and returns claims.
// ParseToken 校验 token 并返回 claims。
func ParseToken(secret string, tokenString string) (*Claims, error) {
	token, err := gjwt.ParseWithClaims(tokenString, &Claims{}, func(token *gjwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*gjwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token failed: %w", err)
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}
	return claims, nil
}
