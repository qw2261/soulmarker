package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/model"
)

func TestJWTManagerSignAndVerifyUser(t *testing.T) {
	manager := NewJWTManager("test-secret-with-enough-entropy")
	issuedAt := time.Now().UTC().Add(-time.Minute).Truncate(time.Second)
	user := &model.User{ID: 42, Name: "测试用户", Contact: "user@example.com", AuthVersion: 3}

	token, err := manager.SignUser(user, issuedAt, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := manager.VerifyUser(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != user.ID || claims.Name != user.Name || claims.Contact != user.Contact || claims.AuthVersion != user.AuthVersion {
		t.Fatalf("unexpected claims: %+v", claims)
	}
	if claims.IssuedAt == nil || !claims.IssuedAt.Time.Equal(issuedAt) {
		t.Fatalf("unexpected issued_at: %+v", claims.IssuedAt)
	}
	if claims.ExpiresAt == nil || !claims.ExpiresAt.Time.Equal(issuedAt.Add(time.Hour)) {
		t.Fatalf("unexpected expires_at: %+v", claims.ExpiresAt)
	}
}

// REG-BUG-G4-001 locks the delimiter-flood rejection path involved in GO-2025-3553.
// The bounded parser implementation is supplied by jwt/v5.2.2 or newer and is
// continuously verified by the pinned govulncheck CI gate.
func TestJWTManagerRejectsDelimiterFlood(t *testing.T) {
	manager := NewJWTManager("test-secret-with-enough-entropy")
	malicious := strings.Repeat(".", 100_000)
	if _, err := manager.VerifyUser(malicious); err == nil {
		t.Fatal("delimiter-flood token accepted")
	}
}

func TestJWTManagerRejectsExpiredToken(t *testing.T) {
	manager := NewJWTManager("test-secret-with-enough-entropy")
	token, err := manager.SignUser(&model.User{ID: 1, AuthVersion: 1}, time.Now().Add(-2*time.Hour), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.VerifyUser(token); err == nil {
		t.Fatal("expired token accepted")
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
