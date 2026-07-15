package identifier

import (
	"encoding/hex"
	"testing"
)

func TestCryptoCredentialGenerator(t *testing.T) {
	generator := CryptoCredentialGenerator{}
	first, err := generator.NewCredential()
	if err != nil {
		t.Fatal(err)
	}
	second, err := generator.NewCredential()
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("credentials must be unique")
	}
	decoded, err := hex.DecodeString(first)
	if err != nil || len(decoded) != credentialBytes {
		t.Fatalf("unexpected credential encoding %q", first)
	}
}
