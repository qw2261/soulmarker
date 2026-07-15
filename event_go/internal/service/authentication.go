package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/clock"
	"github.com/qw2261/soulmarker/event_go/internal/emailaddr"
	"github.com/qw2261/soulmarker/event_go/internal/identifier"
	"github.com/qw2261/soulmarker/event_go/internal/model"
	"github.com/qw2261/soulmarker/event_go/internal/notification"
	"golang.org/x/crypto/bcrypt"
)

const (
	MinPasswordLength = 8
	MaxPasswordLength = 72
)

var ErrRecoveryEmailFormatInvalid = fmt.Errorf("恢复邮箱格式无效")

type AuthenticationRepository interface {
	GetUserByContact(contact string) (*model.User, error)
	GetUserByRecoveryEmail(email string) (*model.User, error)
	CreatePasswordResetToken(userID int64, tokenHash string, createdAt, expiresAt time.Time) error
	ResetPassword(tokenHash, passwordHash string, resetAt time.Time) error
	RevokeUserSessions(userID int64, revokedAt time.Time) error
	CreateRecoveryEmailToken(userID int64, email, tokenHash string, createdAt, expiresAt time.Time) error
	InvalidateRecoveryEmailTokens(userID int64, invalidatedAt time.Time) error
	ConfirmRecoveryEmail(tokenHash string, confirmedAt time.Time) error
}

type AuthenticationService struct {
	repository       AuthenticationRepository
	clock            clock.Clock
	tokens           identifier.ResetTokenGenerator
	sender           notification.AuthenticationEmailSender
	baseURL          string
	resetTTL         time.Duration
	recoveryEmailTTL time.Duration
}

func NewAuthenticationService(
	repository AuthenticationRepository,
	businessClock clock.Clock,
	tokens identifier.ResetTokenGenerator,
	sender notification.AuthenticationEmailSender,
	baseURL string,
	resetTTL time.Duration,
	recoveryEmailTTL time.Duration,
) *AuthenticationService {
	return &AuthenticationService{
		repository: repository, clock: businessClock, tokens: tokens,
		sender: sender, baseURL: strings.TrimRight(baseURL, "/"),
		resetTTL: resetTTL, recoveryEmailTTL: recoveryEmailTTL,
	}
}

func ValidPassword(password string) bool {
	length := len([]byte(password))
	return length >= MinPasswordLength && length <= MaxPasswordLength
}

func resetTokenHash(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}

func (s *AuthenticationService) RequestPasswordReset(ctx context.Context, contact string) error {
	rawToken, err := s.tokens.NewResetToken()
	if err != nil {
		return err
	}
	email, valid := emailaddr.Normalize(contact)
	if !valid {
		return nil
	}
	user, err := s.repository.GetUserByRecoveryEmail(email)
	if err != nil {
		return fmt.Errorf("load password reset user: %w", err)
	}
	if user == nil {
		return nil
	}
	now := s.clock.Now().UTC()
	expiresAt := now.Add(s.resetTTL)
	if err := s.repository.CreatePasswordResetToken(user.ID, resetTokenHash(rawToken), now, expiresAt); err != nil {
		return err
	}
	resetURL, err := url.Parse(s.baseURL + "/reset-password")
	if err != nil {
		return fmt.Errorf("build password reset URL: %w", err)
	}
	query := resetURL.Query()
	query.Set("token", rawToken)
	resetURL.RawQuery = query.Encode()
	if err := s.sender.SendPasswordReset(ctx, user.RecoveryEmail, resetURL.String(), expiresAt); err != nil {
		return fmt.Errorf("deliver password reset: %w", err)
	}
	return nil
}

func (s *AuthenticationService) RequestRecoveryEmail(ctx context.Context, user *model.User, email, password string) error {
	normalizedEmail, valid := emailaddr.Normalize(email)
	if !valid {
		return ErrRecoveryEmailFormatInvalid
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return model.ErrInvalidCreds
	}
	if user.RecoveryEmailVerifiedAt != nil && user.RecoveryEmail == normalizedEmail {
		return model.ErrRecoveryEmailBound
	}
	rawToken, err := s.tokens.NewResetToken()
	if err != nil {
		return err
	}
	now := s.clock.Now().UTC()
	expiresAt := now.Add(s.recoveryEmailTTL)
	if err := s.repository.CreateRecoveryEmailToken(
		user.ID, normalizedEmail, resetTokenHash(rawToken), now, expiresAt,
	); err != nil {
		return err
	}
	verificationURL, err := url.Parse(s.baseURL + "/verify-recovery-email")
	if err != nil {
		return fmt.Errorf("build recovery email verification URL: %w", err)
	}
	query := verificationURL.Query()
	query.Set("token", rawToken)
	verificationURL.RawQuery = query.Encode()
	if err := s.sender.SendRecoveryEmailVerification(ctx, normalizedEmail, verificationURL.String(), expiresAt); err != nil {
		if invalidateErr := s.repository.InvalidateRecoveryEmailTokens(user.ID, now); invalidateErr != nil {
			return fmt.Errorf("deliver recovery email verification: %w; invalidate token: %v", err, invalidateErr)
		}
		return fmt.Errorf("deliver recovery email verification: %w", err)
	}
	return nil
}

func (s *AuthenticationService) ConfirmRecoveryEmail(rawToken string) error {
	if strings.TrimSpace(rawToken) == "" {
		return model.ErrRecoveryEmailInvalid
	}
	return s.repository.ConfirmRecoveryEmail(resetTokenHash(rawToken), s.clock.Now())
}

func (s *AuthenticationService) ResetPassword(rawToken, password string) error {
	if strings.TrimSpace(rawToken) == "" {
		return model.ErrPasswordResetInvalid
	}
	if !ValidPassword(password) {
		return fmt.Errorf("password must be between %d and %d bytes", MinPasswordLength, MaxPasswordLength)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash reset password: %w", err)
	}
	return s.repository.ResetPassword(resetTokenHash(rawToken), string(hash), s.clock.Now())
}

func (s *AuthenticationService) Logout(userID int64) error {
	return s.repository.RevokeUserSessions(userID, s.clock.Now())
}
