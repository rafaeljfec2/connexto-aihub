package providers

import "testing"

type managementCredentialCipher interface {
	SealManagement(string, string) ([]byte, error)
	OpenManagement(string, []byte) (string, error)
}

func TestManagementCredentialCipherIsSeparateFromInferenceCredential(t *testing.T) {
	cipher, err := NewCredentialCipher([]byte("01234567890123456789012345678901"))
	if err != nil {
		t.Fatalf("NewCredentialCipher: %v", err)
	}
	manager, ok := interface{}(cipher).(managementCredentialCipher)
	if !ok {
		t.Fatal("credential cipher does not support a distinct Management API Key context")
	}

	sealed, err := manager.SealManagement("openrouter-primary", "management-secret")
	if err != nil {
		t.Fatalf("SealManagement: %v", err)
	}
	opened, err := manager.OpenManagement("openrouter-primary", sealed)
	if err != nil || opened != "management-secret" {
		t.Fatal("management credential did not round-trip independently")
	}
	if _, err := cipher.Open("openrouter-primary", sealed); err == nil {
		t.Fatal("management ciphertext can be opened as an inference credential")
	}
}
