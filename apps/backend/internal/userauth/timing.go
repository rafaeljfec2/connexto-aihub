package userauth

import "sync"

var (
	dummyPasswordHash string
	dummyHashOnce     sync.Once
	dummyHashErr      error
)

func dummyHashForTiming() string {
	dummyHashOnce.Do(func() {
		dummyPasswordHash, dummyHashErr = HashPassword("aihub-login-timing-mitigation")
	})
	return dummyPasswordHash
}

// MitigateLoginTiming runs Argon2 verification work so unknown emails take similar time as wrong passwords.
func MitigateLoginTiming(password string) {
	hash := dummyHashForTiming()
	if dummyHashErr != nil || hash == "" {
		return
	}
	_, _ = VerifyPassword(hash, password)
}
