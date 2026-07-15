package auth

import (
	"testing"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/model"
)

func TestJWTManagerSignAndVerifyUser(t *testing.T) {
	manager := NewJWTManager("test-secret-with-enough-entropy")
	issuedAt := time.Now().UTC().Add(-time.Minute).Truncate(time.Second)
	user := &model.User{ID: 42, Name: "测试用户", Contact: "user@example.com"}

	token, err := manager.SignUser(user, issuedAt, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := manager.VerifyUser(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != user.ID || claims.Name != user.Name || claims.Contact != user.Contact {
		t.Fatalf("unexpected claims: %+v", claims)
	}
	if claims.IssuedAt == nil || !claims.IssuedAt.Time.Equal(issuedAt) {
		t.Fatalf("unexpected issued_at: %+v", claims.IssuedAt)
	}
	if claims.ExpiresAt == nil || !claims.ExpiresAt.Time.Equal(issuedAt.Add(time.Hour)) {
		t.Fatalf("unexpected expires_at: %+v", claims.ExpiresAt)
	}
}

func TestJWTManagerRejectsWrongSecretAndMalformedToken(t *testing.T) {
	issuer := NewJWTManager("issuer-secret")
	verifier := NewJWTManager("different-secret")
	user := &model.User{ID: 1, Name: "用户", Contact: "user@example.com"}
	token, err := issuer.SignUser(user, time.Now(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := verifier.VerifyUser(token); err == nil {
		t.Fatal("expected wrong-secret verification to fail")
	}
	if _, err := verifier.VerifyUser("not-a-jwt"); err == nil {
		t.Fatal("expected malformed token verification to fail")
	}
}
