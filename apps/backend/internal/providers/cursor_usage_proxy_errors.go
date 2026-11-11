package providers

import "fmt"

// ProxyCursorUsageFailedError is returned when a Connexto Proxy Cursor usage/auth
// call fails (non-2xx or unusable response). Detail is safe for client-facing
// messages. Code mirrors the proxy JSON "status" when present (e.g. browser_unavailable).
// Use errors.Is(err, ErrProxyCursorUsageFailed).
type ProxyCursorUsageFailedError struct {
	Status int
	Detail string
	Code   string
}

func (e *ProxyCursorUsageFailedError) Error() string {
	if e == nil {
		return ErrProxyCursorUsageFailed.Error()
	}
	if e.Detail != "" {
		return fmt.Sprintf("%v: upstream status %d: %s", ErrProxyCursorUsageFailed, e.Status, e.Detail)
	}
	return fmt.Sprintf("%v: upstream status %d", ErrProxyCursorUsageFailed, e.Status)
}

// Is implements errors.Is so callers can use errors.Is(err, ErrProxyCursorUsageFailed).
func (e *ProxyCursorUsageFailedError) Is(target error) bool {
	return target == ErrProxyCursorUsageFailed
}

func newProxyCursorUsageFailed(status int, body []byte) error {
	detail, code := extractProxyUpstreamDetail(body)
	return &ProxyCursorUsageFailedError{
		Status: status,
		Detail: detail,
		Code:   code,
	}
}
