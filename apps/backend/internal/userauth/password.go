package userauth

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argonTime    = 2
	argonMemory  = 64 * 1024
	argonThreads = 4
	argonKeyLen  = 32
	argonSaltLen = 16
)

type argon2Params struct {
	salt     []byte
	want     []byte
	timeCost uint32
	memory   uint32
	threads  uint8
}

// HashPassword returns an Argon2id PHC-style encoded hash string.
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", errors.New("password must not be empty")
	}

	salt, err := randomBytes(argonSaltLen)
	if err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)
	return encoded, nil
}

// VerifyPassword compares a password with an encoded Argon2id hash in constant time on the derived key.
func VerifyPassword(encoded, password string) (bool, error) {
	params, err := parseArgon2ID(encoded)
	if err != nil {
		return false, err
	}

	got := argon2.IDKey([]byte(password), params.salt, params.timeCost, params.memory, params.threads, uint32(len(params.want)))
	return subtle.ConstantTimeCompare(got, params.want) == 1, nil
}

func parseArgon2ID(encoded string) (argon2Params, error) {
	if !strings.HasPrefix(encoded, "$argon2id$") {
		return argon2Params{}, errors.New("unsupported password hash")
	}

	parts := strings.Split(encoded, "$")
	if len(parts) != 6 {
		return argon2Params{}, errors.New("malformed password hash")
	}

	var version int
	var memory, timeCost uint32
	var threads uint8
	if _, scanErr := fmt.Sscanf(parts[2], "v=%d", &version); scanErr != nil {
		return argon2Params{}, errors.New("malformed password hash version")
	}
	if _, scanErr := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &timeCost, &threads); scanErr != nil {
		return argon2Params{}, errors.New("malformed password hash params")
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return argon2Params{}, err
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return argon2Params{}, err
	}

	return argon2Params{salt: salt, want: hash, timeCost: timeCost, memory: memory, threads: threads}, nil
}
