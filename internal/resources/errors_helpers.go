package resources

import "errors"

// IsLimitExceeded reports quota rejections from Admit.
func IsLimitExceeded(err error) bool {
	return errors.Is(err, ErrLimitExceeded)
}
