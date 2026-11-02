package access

import (
	"net/http"
	"strings"
)

// Authorize enforces Host/Origin policy and static bearer authentication.
func (p Policy) Authorize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, ok := normalizeRequestHost(r.Host)
		if !ok || !p.hostAllowed(host) {
			forbidden(w)
			return
		}

		origin, ok := requestOrigin(r.Header)
		if !ok || origin != "" && !p.originAllowed(origin) {
			forbidden(w)
			return
		}

		metadata := Metadata{Host: host, Origin: origin}
		if !healthExempt(r) {
			principal, authorized := p.authenticate(r)
			if !authorized {
				unauthorized(w)
				return
			}

			metadata.Principal = principal
		}

		next.ServeHTTP(w, r.WithContext(WithMetadata(r.Context(), metadata)))
	})
}

func requestOrigin(header http.Header) (string, bool) {
	values := header.Values("Origin")
	if len(values) == 0 {
		return "", true
	}

	if len(values) != 1 {
		return "", false
	}

	origin, err := canonicalOrigin(values[0])
	if err != nil {
		return "", false
	}

	return origin, true
}

func healthExempt(r *http.Request) bool {
	if r.Method != http.MethodGet {
		return false
	}

	return r.URL.Path == "/healthz" || r.URL.Path == "/readyz"
}

// authenticate selects the credential for the route namespace: the
// administrative API under /api/ accepts only the administrative key, and the
// gateway routes accept only the gateway key. The two keys never cross.
func (p Policy) authenticate(r *http.Request) (string, bool) {
	token, ok := bearerToken(r.Header)
	if !ok {
		return "", false
	}

	if strings.HasPrefix(r.URL.Path, adminPathPrefix) {
		return AdminPrincipalID, p.adminKeyMatches(token)
	}

	return PrincipalID, p.keyMatches(token)
}
