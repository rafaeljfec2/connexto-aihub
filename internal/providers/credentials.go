package providers

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"

	"aihub/internal/config"
)

// ErrCredentialUnreadable reports a stored credential that cannot be
// authenticated with the configured key or bound provider. The message never
// includes ciphertext or plaintext.
var ErrCredentialUnreadable = errors.New("stored provider credential cannot be decrypted")

// CredentialCipher seals provider credentials with AES-256-GCM. The provider
// ID is bound as additional authenticated data, so a ciphertext copied to
// another provider row fails to open. The key is supplied from the process
// environment (AIHUB_CREDENTIAL_KEY) and never stored in PostgreSQL.
type CredentialCipher struct {
	aead cipher.AEAD
}

// NewCredentialCipher builds the cipher from a 32-byte key.
func NewCredentialCipher(key []byte) (*CredentialCipher, error) {
	if len(key) != config.CredentialKeyBytes {
		return nil, fmt.Errorf("credential key must be %d bytes", config.CredentialKeyBytes)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("build credential cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("build credential cipher: %w", err)
	}

	return &CredentialCipher{aead: aead}, nil
}

// Seal encrypts a credential for one provider. The output is nonce||ciphertext.
func (c *CredentialCipher) Seal(providerID, credential string) ([]byte, error) {
	return c.seal(providerID, credential)
}

// SealManagement protects an administrative provider credential in a distinct cryptographic context.
func (c *CredentialCipher) SealManagement(providerID, credential string) ([]byte, error) {
	return c.seal(providerID+"\x00management", credential)
}

func (c *CredentialCipher) seal(contextID, credential string) ([]byte, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate credential nonce: %w", err)
	}

	sealed := c.aead.Seal(nonce, nonce, []byte(credential), []byte(contextID))

	return sealed, nil
}

// Open decrypts a credential previously sealed for the same provider.
func (c *CredentialCipher) Open(providerID string, sealed []byte) (string, error) {
	return c.open(providerID, sealed)
}

// OpenManagement opens an administrative credential only in its own cryptographic context.
func (c *CredentialCipher) OpenManagement(providerID string, sealed []byte) (string, error) {
	return c.open(providerID+"\x00management", sealed)
}

func (c *CredentialCipher) open(contextID string, sealed []byte) (string, error) {
	nonceSize := c.aead.NonceSize()
	if len(sealed) < nonceSize+c.aead.Overhead() {
		return "", ErrCredentialUnreadable
	}

	plaintext, err := c.aead.Open(nil, sealed[:nonceSize], sealed[nonceSize:], []byte(contextID))
	if err != nil {
		return "", ErrCredentialUnreadable
	}

	return string(plaintext), nil
}
