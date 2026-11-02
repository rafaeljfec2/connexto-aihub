package access

import (
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"
)

const (
	keyMinLength = 32
	keyMaxLength = 512
)

// ValidateAPIKey enforces the RFC 6750 b64token grammar used for the static
// gateway client key. Errors never contain the supplied value.
func ValidateAPIKey(key string) error {
	if len(key) < keyMinLength || len(key) > keyMaxLength {
		return errors.New("api key must contain between 32 and 512 ASCII characters")
	}

	nonPadding := 0
	padding := false
	for index := 0; index < len(key); index++ {
		char := key[index]
		if char == '=' {
			padding = true
			continue
		}

		if padding {
			return errors.New("api key may only use trailing '=' padding")
		}

		if !validTokenByte(char) {
			return errors.New("api key contains a character outside the allowed b64token set")
		}

		nonPadding++
	}

	if nonPadding == 0 {
		return errors.New("api key must contain at least one non-padding character")
	}

	return nil
}

func validTokenByte(char byte) bool {
	switch {
	case char >= 'a' && char <= 'z':
		return true
	case char >= 'A' && char <= 'Z':
		return true
	case char >= '0' && char <= '9':
		return true
	case char == '.', char == '_', char == '~', char == '+', char == '/', char == '-':
		return true
	default:
		return false
	}
}

func digestKey(key string) [sha256.Size]byte {
	return sha256.Sum256([]byte(key))
}

// adminKeyMatches reports whether token is the administrative key. It is
// false when no administrative key is configured.
func (p Policy) adminKeyMatches(token string) bool {
	if !p.adminEnabled || ValidateAPIKey(token) != nil {
		return false
	}

	digest := digestKey(token)
	return subtle.ConstantTimeCompare(p.adminDigest[:], digest[:]) == 1
}

func (p Policy) keyMatches(token string) bool {
	if ValidateAPIKey(token) != nil {
		return false
	}

	digest := digestKey(token)
	return subtle.ConstantTimeCompare(p.keyDigest[:], digest[:]) == 1
}

// bearerToken accepts exactly one Authorization value with the grammar
// "Bearer <token>". Grammar failures are not distinguished from a wrong key.
func bearerToken(header http.Header) (string, bool) {
	values := header.Values("Authorization")
	if len(values) != 1 {
		return "", false
	}

	value := values[0]
	const prefix = "bearer "
	if len(value) <= len(prefix) || !strings.EqualFold(value[:len(prefix)], prefix) {
		return "", false
	}

	return value[len(prefix):], true
}
