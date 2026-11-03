package gateway

import (
	"errors"
	"net/http"
)

func rejectUnknownQueryParams(r *http.Request, allowed ...string) error {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, name := range allowed {
		allowedSet[name] = struct{}{}
	}
	for name := range r.URL.Query() {
		if _, ok := allowedSet[name]; !ok {
			return errors.New("unknown query parameter")
		}
	}
	return nil
}
