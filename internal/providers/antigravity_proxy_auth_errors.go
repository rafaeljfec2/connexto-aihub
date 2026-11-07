package providers

import "fmt"

// ProxyAntigravityAuthFailedError is returned when the Connexto Proxy Antigravity
// auth call fails (non-2xx or unusable response). Detail is safe for client-facing
// messages (sanitized, length-capped). Use errors.Is(err, ErrProxyAntigravityAuthFailed).
type ProxyAntigravityAuthFailedError struct {
	Status int
	Detail string
}

func (e *ProxyAntigravityAuthFailedError) Error() string {
	if e == nil {
		return ErrProxyAntigravityAuthFailed.Error()
	}
	if e.Detail != "" {
		return fmt.Sprintf("%v: upstream status %d: %s", ErrProxyAntigravityAuthFailed, e.Status, e.Detail)
	}
	return fmt.Sprintf("%v: upstream status %d", ErrProxyAntigravityAuthFailed, e.Status)
}

// Is implements errors.Is so callers can use errors.Is(err, ErrProxyAntigravityAuthFailed).
func (e *ProxyAntigravityAuthFailedError) Is(target error) bool {
	return target == ErrProxyAntigravityAuthFailed
}

func newProxyAntigravityAuthFailed(status int, body []byte) error {
	detail, _ := extractProxyUpstreamDetail(body)
	return &ProxyAntigravityAuthFailedError{
		Status: status,
		Detail: detail,
	}
}

func extractProxyAntigravityAuthUpstreamDetail(raw []byte) string {
	detail, _ := extractProxyUpstreamDetail(raw)
	return detail
}
