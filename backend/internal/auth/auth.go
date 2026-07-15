package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"cloudcostdash/internal/models"
)

type Claims struct {
	UserID    uint             `json:"userId"`
	Role      models.Role      `json:"role"`
	ScopeType models.ScopeType `json:"scopeType"`
	ScopeID   *uint            `json:"scopeId,omitempty"`
	jwt.RegisteredClaims
}

func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func IssueToken(secret string, user models.User) (string, error) {
	claims := Claims{
		UserID:    user.ID,
		Role:      user.Role,
		ScopeType: user.ScopeType,
		ScopeID:   user.ScopeID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ParseToken(secret, tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
