package identifier

import (
	"encoding/base64"
	"testing"
)

func TestCryptoResetTokenGeneratorProducesIndependent256BitTokens(t *testing.T) {
	generator := CryptoResetTokenGenerator{}
	first, err := generator.NewResetToken()
	if err != nil {
		t.Fatal(err)
	}
	second, err := generator.NewResetToken()
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("password reset tokens must be independent")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(first)
	if err != nil || len(decoded) != resetTokenBytes {
		t.Fatalf("unexpected reset token: bytes=%d err=%v", len(decoded), err)
	}
}
