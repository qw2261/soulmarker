package store

import (
	"errors"
	"testing"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/model"
	"golang.org/x/crypto/bcrypt"
)

func TestPasswordResetIsSingleUseAndRevokesSessions(t *testing.T) {
	s, err := NewStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	oldHash, _ := bcrypt.GenerateFromPassword([]byte("old-password"), bcrypt.MinCost)
	user := &model.User{Name: "重置用户", Contact: "reset@example.com", PasswordHash: string(oldHash)}
	if err := s.CreateUser(user); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	if err := s.CreatePasswordResetToken(user.ID, "old-token-hash", now, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := s.CreatePasswordResetToken(user.ID, "rate-limited-hash", now.Add(30*time.Second), now.Add(time.Hour)); !errors.Is(err, model.ErrPasswordResetRateLimit) {
		t.Fatalf("password reset cooldown not enforced: %v", err)
	}
	if err := s.CreatePasswordResetToken(user.ID, "new-token-hash", now.Add(time.Minute), now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	newHash, _ := bcrypt.GenerateFromPassword([]byte("new-password"), bcrypt.MinCost)
	if err := s.ResetPassword("old-token-hash", string(newHash), now.Add(2*time.Minute)); !errors.Is(err, model.ErrPasswordResetInvalid) {
		t.Fatalf("superseded token accepted: %v", err)
	}
	if err := s.ResetPassword("new-token-hash", string(newHash), now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	updated, err := s.GetUserByID(user.ID)
	if err != nil || updated == nil {
		t.Fatalf("load reset user: user=%+v err=%v", updated, err)
	}
	if updated.AuthVersion != 2 {
		t.Fatalf("expected auth version 2 after reset, got %d", updated.AuthVersion)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(updated.PasswordHash), []byte("new-password")); err != nil {
		t.Fatalf("new password not persisted: %v", err)
	}
	if err := s.ResetPassword("new-token-hash", string(newHash), now.Add(3*time.Minute)); !errors.Is(err, model.ErrPasswordResetInvalid) {
		t.Fatalf("consumed token accepted: %v", err)
	}
	if err := s.RevokeUserSessions(user.ID, now.Add(4*time.Minute)); err != nil {
		t.Fatal(err)
	}
	updated, err = s.GetUserByID(user.ID)
	if err != nil || updated.AuthVersion != 3 {
		t.Fatalf("logout did not revoke sessions: user=%+v err=%v", updated, err)
	}
}

func TestPasswordResetRejectsExpiredToken(t *testing.T) {
	s, err := NewStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	user := &model.User{Name: "过期用户", Contact: "expired@example.com", PasswordHash: "hash"}
	if err := s.CreateUser(user); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	if err := s.CreatePasswordResetToken(user.ID, "expired-hash", now, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := s.ResetPassword("expired-hash", "new-hash", now.Add(time.Minute)); !errors.Is(err, model.ErrPasswordResetInvalid) {
		t.Fatalf("expired token accepted: %v", err)
	}
}
