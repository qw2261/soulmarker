package store

import (
	"errors"
	"testing"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/model"
)

func TestRecoveryEmailBindingIsVerifiedUniqueAndRevokesOldCredentials(t *testing.T) {
	s, err := NewStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	phoneUser := &model.User{Name: "历史手机用户", Contact: "13800138000", PasswordHash: "hash"}
	existingEmailUser := &model.User{Name: "邮箱用户", Contact: "taken@example.com", PasswordHash: "hash"}
	for _, user := range []*model.User{phoneUser, existingEmailUser} {
		if err := s.CreateUser(user); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Date(2032, 3, 4, 5, 6, 7, 0, time.UTC)
	if err := s.CreateRecoveryEmailToken(
		phoneUser.ID, "taken@example.com", "taken-token-hash", now, now.Add(30*time.Minute),
	); !errors.Is(err, model.ErrRecoveryEmailInUse) {
		t.Fatalf("bound email was accepted: %v", err)
	}
	if err := s.CreatePasswordResetToken(phoneUser.ID, "old-reset-hash", now.Add(-2*time.Minute), now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateRecoveryEmailToken(
		phoneUser.ID, "recovery@example.com", "recovery-token-hash", now, now.Add(30*time.Minute),
	); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateRecoveryEmailToken(
		phoneUser.ID, "other@example.com", "other-token-hash", now.Add(30*time.Second), now.Add(30*time.Minute),
	); !errors.Is(err, model.ErrRecoveryEmailRateLimit) {
		t.Fatalf("recovery email rate limit missing: %v", err)
	}
	if err := s.ConfirmRecoveryEmail("invalid-token-hash", now); !errors.Is(err, model.ErrRecoveryEmailInvalid) {
		t.Fatalf("invalid verification token accepted: %v", err)
	}
	if err := s.ConfirmRecoveryEmail("recovery-token-hash", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	bound, err := s.GetUserByID(phoneUser.ID)
	if err != nil {
		t.Fatal(err)
	}
	if bound == nil || bound.RecoveryEmail != "recovery@example.com" || bound.RecoveryEmailVerifiedAt == nil || bound.AuthVersion != 2 {
		t.Fatalf("unexpected bound user: %+v", bound)
	}
	byEmail, err := s.GetUserByRecoveryEmail("recovery@example.com")
	if err != nil || byEmail == nil || byEmail.ID != phoneUser.ID {
		t.Fatalf("verified recovery lookup failed: user=%+v err=%v", byEmail, err)
	}
	if err := s.ConfirmRecoveryEmail("recovery-token-hash", now.Add(2*time.Minute)); !errors.Is(err, model.ErrRecoveryEmailInvalid) {
		t.Fatalf("verification token was reusable: %v", err)
	}
	if err := s.ResetPassword("old-reset-hash", "new-hash", now.Add(2*time.Minute)); !errors.Is(err, model.ErrPasswordResetInvalid) {
		t.Fatalf("old reset token survived email binding: %v", err)
	}
}
