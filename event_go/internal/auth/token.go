package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/qw2261/soulmarker/event_go/internal/model"
)

type TokenManager interface {
	SignUser(user *model.User, issuedAt time.Time, ttl time.Duration) (string, error)
	VerifyUser(rawToken string) (*model.UserClaims, error)
}

type JWTManager struct {
	secret []byte
}

var ErrInvalidToken = errors.New("invalid user token")

func NewJWTManager(secret string) *JWTManager {
	return &JWTManager{secret: []byte(secret)}
}

func (m *JWTManager) SignUser(user *model.User, issuedAt time.Time, ttl time.Duration) (string, error) {
	claims := &model.UserClaims{
		UserID:      user.ID,
		Name:        user.Name,
		Contact:     user.Contact,
		AuthVersion: user.AuthVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(issuedAt.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(issuedAt),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

func (m *JWTManager) VerifyUser(rawToken string) (*model.UserClaims, error) {
	claims := &model.UserClaims{}
	token, err := jwt.ParseWithClaims(rawToken, claims, func(token *jwt.Token) (interface{}, error) {
		return m.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, fmt.Errorf("verify user token: %w", err)
	}
	if token == nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
