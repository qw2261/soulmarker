package identifier

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

const resetTokenBytes = 32

type ResetTokenGenerator interface {
	NewResetToken() (string, error)
}

type CryptoResetTokenGenerator struct{}

func (CryptoResetTokenGenerator) NewResetToken() (string, error) {
	buffer := make([]byte, resetTokenBytes)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate password reset token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}
