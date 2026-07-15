package identifier

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

const credentialBytes = 16

// CredentialGenerator 为 Admission 生成不可预测的公开凭证码。
type CredentialGenerator interface {
	NewCredential() (string, error)
}

type CryptoCredentialGenerator struct{}

func (CryptoCredentialGenerator) NewCredential() (string, error) {
	buffer := make([]byte, credentialBytes)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate admission credential: %w", err)
	}
	return hex.EncodeToString(buffer), nil
}
