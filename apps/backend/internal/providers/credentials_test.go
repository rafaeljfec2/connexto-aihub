package providers_test

import (
	"bytes"
	"strings"
	"testing"

	"aihub/internal/config"
	"aihub/internal/providers"
)

const testSecret = "sk-test-secret-value-0001"

func testCipher(t *testing.T) *providers.CredentialCipher {
	t.Helper()

	cipher, err := providers.NewCredentialCipher(bytes.Repeat([]byte{0x11}, config.CredentialKeyBytes))
	if err != nil {
		t.Fatalf("NewCredentialCipher: %v", err)
	}

	return cipher
}

func TestCredentialCipherRoundTripsForTheSameProvider(t *testing.T) {
	cipher := testCipher(t)

	sealed, err := cipher.Seal("opencode-go", testSecret)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}

	if bytes.Contains(sealed, []byte(testSecret)) {
		t.Fatal("ciphertext contains the plaintext credential")
	}

	opened, err := cipher.Open("opencode-go", sealed)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	if opened != testSecret {
		t.Fatalf("Open returned a different credential")
	}
}

func TestCredentialCipherRejectsCiphertextBoundToAnotherProvider(t *testing.T) {
	cipher := testCipher(t)

	sealed, err := cipher.Seal("opencode-go", testSecret)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}

	if _, err := cipher.Open("openai-primary", sealed); err == nil {
		t.Fatal("ciphertext opened for a different provider id")
	}
}

func TestCredentialCipherRejectsTamperedAndTruncatedCiphertext(t *testing.T) {
	cipher := testCipher(t)

	sealed, err := cipher.Seal("opencode-go", testSecret)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}

	tampered := bytes.Clone(sealed)
	tampered[len(tampered)-1] ^= 0x01

	if _, err := cipher.Open("opencode-go", tampered); err == nil {
		t.Fatal("tampered ciphertext was accepted")
	}

	if _, err := cipher.Open("opencode-go", sealed[:5]); err == nil {
		t.Fatal("truncated ciphertext was accepted")
	}
}

func TestCredentialErrorsNeverEchoSecrets(t *testing.T) {
	cipher := testCipher(t)

	sealed, err := cipher.Seal("opencode-go", testSecret)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}

	_, err = cipher.Open("wrong-id", sealed)
	if err == nil || strings.Contains(err.Error(), testSecret) {
		t.Fatalf("open error = %v, want a generic message without the secret", err)
	}
}

func TestNewCredentialCipherRequiresAES256Key(t *testing.T) {
	if _, err := providers.NewCredentialCipher(make([]byte, 16)); err == nil {
		t.Fatal("a 16-byte key was accepted")
	}
}
