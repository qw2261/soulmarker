package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/model"
	"golang.org/x/crypto/bcrypt"
)

type fixedResetTokenGenerator struct {
	token string
	err   error
	calls int
}

func (g *fixedResetTokenGenerator) NewResetToken() (string, error) {
	g.calls++
	return g.token, g.err
}

type recordingResetSender struct {
	contact   string
	resetURL  string
	expiresAt time.Time
	err       error
}

func (s *recordingResetSender) SendPasswordReset(_ context.Context, contact, resetURL string, expiresAt time.Time) error {
	s.contact, s.resetURL, s.expiresAt = contact, resetURL, expiresAt
	return s.err
}

type fakeAuthenticationRepository struct {
	user          *model.User
	createdHash   string
	createdAt     time.Time
	expiresAt     time.Time
	resetHash     string
	passwordHash  string
	resetAt       time.Time
	revokedUserID int64
	err           error
}

func (r *fakeAuthenticationRepository) GetUserByContact(string) (*model.User, error) {
	return r.user, r.err
}

func (r *fakeAuthenticationRepository) CreatePasswordResetToken(_ int64, tokenHash string, createdAt, expiresAt time.Time) error {
	r.createdHash, r.createdAt, r.expiresAt = tokenHash, createdAt, expiresAt
	return r.err
}

func (r *fakeAuthenticationRepository) ResetPassword(tokenHash, passwordHash string, resetAt time.Time) error {
	r.resetHash, r.passwordHash, r.resetAt = tokenHash, passwordHash, resetAt
	return r.err
}

func (r *fakeAuthenticationRepository) RevokeUserSessions(userID int64, _ time.Time) error {
	r.revokedUserID = userID
	return r.err
}

func TestAuthenticationServiceRequestsOpaqueExpiringResetLink(t *testing.T) {
	now := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	repository := &fakeAuthenticationRepository{user: &model.User{ID: 7, Contact: "user@example.com"}}
	generator := &fixedResetTokenGenerator{token: "opaque-reset-token"}
	sender := &recordingResetSender{}
	service := NewAuthenticationService(repository, fixedClock{now: now}, generator, sender, "https://events.example.com/", 30*time.Minute)
	if err := service.RequestPasswordReset(context.Background(), "user@example.com"); err != nil {
		t.Fatal(err)
	}
	if repository.createdHash == "opaque-reset-token" || len(repository.createdHash) != 64 {
		t.Fatalf("reset token was not hashed: %q", repository.createdHash)
	}
	if !repository.createdAt.Equal(now) || !repository.expiresAt.Equal(now.Add(30*time.Minute)) {
		t.Fatalf("unexpected reset lifetime: created=%s expires=%s", repository.createdAt, repository.expiresAt)
	}
	if sender.contact != "user@example.com" || !strings.Contains(sender.resetURL, "/reset-password?token=opaque-reset-token") || !sender.expiresAt.Equal(repository.expiresAt) {
		t.Fatalf("unexpected reset delivery: %+v", sender)
	}
}

func TestAuthenticationServiceDoesNotRevealUnknownAccount(t *testing.T) {
	repository := &fakeAuthenticationRepository{}
	generator := &fixedResetTokenGenerator{token: "unused-but-generated"}
	sender := &recordingResetSender{}
	service := NewAuthenticationService(repository, fixedClock{}, generator, sender, "https://events.example.com", 30*time.Minute)
	if err := service.RequestPasswordReset(context.Background(), "missing@example.com"); err != nil {
		t.Fatal(err)
	}
	if generator.calls != 1 || repository.createdHash != "" || sender.contact != "" {
		t.Fatalf("unknown account changed reset state: generator=%d repository=%+v sender=%+v", generator.calls, repository, sender)
	}
}

func TestAuthenticationServiceResetsPasswordAndForwardsLogout(t *testing.T) {
	now := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	repository := &fakeAuthenticationRepository{}
	service := NewAuthenticationService(repository, fixedClock{now: now}, &fixedResetTokenGenerator{}, &recordingResetSender{}, "https://events.example.com", 30*time.Minute)
	if err := service.ResetPassword("one-time-token", "new-password"); err != nil {
		t.Fatal(err)
	}
	if repository.resetHash == "one-time-token" || len(repository.resetHash) != 64 || !repository.resetAt.Equal(now) {
		t.Fatalf("raw reset token reached repository: %+v", repository)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(repository.passwordHash), []byte("new-password")); err != nil {
		t.Fatalf("password was not securely hashed: %v", err)
	}
	if err := service.Logout(42); err != nil || repository.revokedUserID != 42 {
		t.Fatalf("logout not forwarded: user=%d err=%v", repository.revokedUserID, err)
	}
}

func TestAuthenticationServiceRejectsInvalidPasswordAndToken(t *testing.T) {
	repository := &fakeAuthenticationRepository{err: model.ErrPasswordResetInvalid}
	service := NewAuthenticationService(repository, fixedClock{}, &fixedResetTokenGenerator{}, &recordingResetSender{}, "https://events.example.com", 30*time.Minute)
	if err := service.ResetPassword("", "new-password"); !errors.Is(err, model.ErrPasswordResetInvalid) {
		t.Fatalf("empty token accepted: %v", err)
	}
	if err := service.ResetPassword("token", "short"); err == nil {
		t.Fatal("short password accepted")
	}
	if err := service.ResetPassword("token", "valid-password"); !errors.Is(err, model.ErrPasswordResetInvalid) {
		t.Fatalf("repository error not preserved: %v", err)
	}
}
