package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
)

// Claims JWT claims 結構
type Claims struct {
	UserID    int64  `json:"user_id"`
	Email     string `json:"email"`
	SessionID string `json:"session_id"`
	jwt.RegisteredClaims
}

// GenerateToken 生成 JWT token
func GenerateToken(userID int64, email string, sessionID string, jwtSecret string, expiration time.Duration) (string, error) {
	claims := Claims{
		UserID:    userID,
		Email:     email,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}

// ValidateToken 驗證並解析 JWT token
func ValidateToken(tokenString string, jwtSecret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		// 檢查是否過期
		if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
			return nil, ErrExpiredToken
		}
		return claims, nil
	}

	return nil, ErrInvalidToken
}

// RefreshToken 刷新 token
func RefreshToken(tokenString string, jwtSecret string, expiration time.Duration) (string, error) {
	claims, err := ValidateToken(tokenString, jwtSecret)
	if err != nil {
		return "", err
	}

	// 生成新的 token（保留原有的 session ID）
	return GenerateToken(claims.UserID, claims.Email, claims.SessionID, jwtSecret, expiration)
}
