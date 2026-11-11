package access_test

import (
	"strings"
	"testing"

	"aihub/internal/access"
)

const testGatewayKey = "test-gateway-key-0123456789abcdef"

func TestValidateAPIKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{name: "minimum length", key: strings.Repeat("a", 32)},
		{name: "maximum length", key: strings.Repeat("a", 512)},
		{name: "allowed punctuation", key: "abc.DEF_012-345~678+901/234567890"},
		{name: "trailing padding", key: strings.Repeat("a", 32) + "===="},
		{name: "too short", key: strings.Repeat("a", 31), wantErr: true},
		{name: "too long", key: strings.Repeat("a", 513), wantErr: true},
		{name: "empty", key: "", wantErr: true},
		{name: "only padding", key: strings.Repeat("=", 32), wantErr: true},
		{name: "internal padding", key: strings.Repeat("a", 31) + "=a", wantErr: true},
		{name: "space", key: "test gateway key 0123456789abcdef", wantErr: true},
		{name: "comma", key: "test,gateway,key,0123456789abcdef", wantErr: true},
		{name: "colon", key: "test:gateway:key:0123456789abcdef", wantErr: true},
		{name: "semicolon", key: "test;gateway;key;0123456789abcdef", wantErr: true},
		{name: "control character", key: "test-gateway-key-0123456789abc\x00", wantErr: true},
		{name: "unicode", key: "test-gateway-key-0123456789abcdé", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := access.ValidateAPIKey(test.key)
			if test.wantErr && err == nil {
				t.Fatalf("ValidateAPIKey() must reject %q", test.key)
			}

			if !test.wantErr && err != nil {
				t.Fatalf("ValidateAPIKey() returned error: %v", err)
			}
		})
	}
}

func TestValidateAPIKeyDoesNotEchoTheValue(t *testing.T) {
	const marker = "secret-marker-value"

	err := access.ValidateAPIKey(marker + ",invalid")
	if err == nil {
		t.Fatal("ValidateAPIKey() must fail")
	}

	if strings.Contains(err.Error(), marker) {
		t.Fatal("the error must not contain the key value")
	}
}

func TestNewPolicyRejectsInvalidKey(t *testing.T) {
	if _, err := access.NewPolicy(access.PolicyConfig{APIKey: "short"}); err == nil {
		t.Fatal("NewPolicy() must reject an invalid key")
	}
}
