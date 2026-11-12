package userauth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

func randomBytes(n int) ([]byte, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return nil, fmt.Errorf("random bytes: %w", err)
	}
	return buf, nil
}

// NewOpaqueToken returns a URL-safe opaque token and its SHA-256 digest for persistence.
func NewOpaqueToken() (plain string, digest []byte, err error) {
	raw, err := randomBytes(32)
	if err != nil {
		return "", nil, err
	}
	plain = base64.RawURLEncoding.EncodeToString(raw)
	sum := sha256.Sum256([]byte(plain))
	return plain, sum[:], nil
}

func hashOpaque(plain string) []byte {
	sum := sha256.Sum256([]byte(plain))
	return sum[:]
}
