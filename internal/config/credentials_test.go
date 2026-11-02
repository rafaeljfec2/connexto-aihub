package config_test

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"

	"aihub/internal/config"
)

func TestLoadDecodesCredentialKey(t *testing.T) {
	unsetAIHUBEnv(t)

	want := bytes.Repeat([]byte{0x42}, config.CredentialKeyBytes)
	t.Setenv(config.EnvCredentialKey, base64.StdEncoding.EncodeToString(want))

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if !bytes.Equal(cfg.Credentials.Key, want) {
		t.Fatalf("credentials.key was not decoded from base64")
	}
}

func TestLoadRejectsCredentialKeyOfWrongLengthWithoutEchoingValue(t *testing.T) {
	unsetAIHUBEnv(t)

	short := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x01}, 16))
	t.Setenv(config.EnvCredentialKey, short)

	_, err := config.Load()
	if err == nil {
		t.Fatal("Load() accepted a 16-byte credential key")
	}

	if strings.Contains(err.Error(), short) {
		t.Fatalf("error message echoes the credential key: %v", err)
	}
}

func TestLoadRejectsNonBase64CredentialKey(t *testing.T) {
	unsetAIHUBEnv(t)

	t.Setenv(config.EnvCredentialKey, "not base64 !!")

	if _, err := config.Load(); err == nil {
		t.Fatal("Load() accepted a non-base64 credential key")
	}
}

func TestLoadRequiresCredentialKeyWhenPostgresPersistenceIsUsed(t *testing.T) {
	unsetAIHUBEnv(t)

	t.Setenv(config.EnvDatabaseURL, "postgres://localhost:5432/ai_hub")

	_, err := config.Load()
	if err == nil || !strings.Contains(err.Error(), config.EnvCredentialKey) {
		t.Fatalf("Load() error = %v, want a %s requirement", err, config.EnvCredentialKey)
	}
}

func TestLoadAllowsFileModeWithoutCredentialKey(t *testing.T) {
	unsetAIHUBEnv(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if len(cfg.Credentials.Key) != 0 {
		t.Fatalf("credentials.key = %d bytes, want none", len(cfg.Credentials.Key))
	}
}

func setTestCredentialKey(t *testing.T) {
	t.Helper()

	t.Setenv(config.EnvCredentialKey, base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x42}, config.CredentialKeyBytes)))
}
