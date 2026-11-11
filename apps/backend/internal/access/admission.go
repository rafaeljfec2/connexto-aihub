package access

import (
	"bytes"
	"io"
	"mime"
	"net/http"
	"strings"
)

const catalogMaxBodyBytes int64 = 4 << 10

type admissionKind uint8

const (
	admissionNone admissionKind = iota
	admissionResponses
	admissionCatalogSync
)

// Admission is an immutable per-route body and header admission policy. The
// zero value performs no admission.
type Admission struct {
	kind       admissionKind
	limit      int64
	newSession func() (string, error)
}

// ResponsesAdmission bounds provider requests and resolves execution sessions.
func ResponsesAdmission(maxBodyBytes int64) Admission {
	if maxBodyBytes <= 0 {
		maxBodyBytes = DefaultMaxBodyBytes
	}

	return Admission{kind: admissionResponses, limit: maxBodyBytes, newSession: NewSessionID}
}

// CatalogSyncAdmission preserves the E03 catalog body bound of 4 KiB.
func CatalogSyncAdmission(maxBodyBytes int64) Admission {
	if maxBodyBytes <= 0 || maxBodyBytes > catalogMaxBodyBytes {
		maxBodyBytes = catalogMaxBodyBytes
	}

	return Admission{kind: admissionCatalogSync, limit: maxBodyBytes}
}

// Apply installs the admission checks before the next handler.
func (a Admission) Apply(next http.Handler) http.Handler {
	if a.kind == admissionNone {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.admit(w, r) {
			next.ServeHTTP(w, r)
		}
	})
}

func (a Admission) admit(w http.ResponseWriter, r *http.Request) bool {
	if !validContentEncoding(r.Header) {
		unsupportedMediaType(w)
		return false
	}

	if len(r.Header.Values("Content-Type")) > 1 || len(r.Header.Values("X-Opencode-Session")) > 1 {
		invalidRequest(w, "duplicate request headers are not allowed")
		return false
	}

	if a.kind == admissionResponses {
		if !a.resolveSession(w, r) {
			return false
		}
	}

	if !a.validMediaType(r) {
		unsupportedMediaType(w)
		return false
	}

	if r.ContentLength > a.limit {
		payloadTooLarge(w)
		return false
	}

	body, ok := a.readBoundedBody(w, r)
	if !ok {
		return false
	}

	r.Body = io.NopCloser(bytes.NewReader(body))
	r.ContentLength = int64(len(body))

	if a.kind == admissionResponses {
		metadata, _ := MetadataFrom(r.Context())
		metadata.ProviderSessionID = r.Header.Get("X-Opencode-Session")
		*r = *r.WithContext(WithMetadata(r.Context(), metadata))
	}

	return true
}

func validContentEncoding(header http.Header) bool {
	values := header.Values("Content-Encoding")
	if len(values) == 0 {
		return true
	}

	return len(values) == 1 && values[0] == "identity"
}

func (a Admission) resolveSession(w http.ResponseWriter, r *http.Request) bool {
	values := r.Header.Values("X-Opencode-Session")
	if len(values) == 0 || len(values) == 1 && values[0] == "" {
		generator := a.newSession
		if generator == nil {
			generator = NewSessionID
		}

		session, err := generator()
		if err != nil {
			writeError(w, http.StatusInternalServerError, ErrorInternal, "could not create a provider session")
			return false
		}

		r.Header.Set("X-Opencode-Session", session)
		return true
	}

	if len(values) != 1 || !ValidSessionID(values[0]) {
		invalidRequest(w, "provider session header is invalid")
		return false
	}

	return true
}

func (a Admission) validMediaType(r *http.Request) bool {
	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		if a.kind == admissionCatalogSync && r.ContentLength == 0 {
			return true
		}

		return false
	}

	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil || !strings.EqualFold(mediaType, "application/json") {
		return false
	}

	if len(params) == 0 {
		return true
	}

	if len(params) != 1 {
		return false
	}

	charset, ok := params["charset"]
	return ok && strings.EqualFold(charset, "utf-8")
}

func (a Admission) readBoundedBody(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	if r.Body == nil {
		return []byte{}, true
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, a.limit+1))
	_ = r.Body.Close()
	if err != nil {
		invalidRequest(w, "request body could not be read")
		return nil, false
	}

	if int64(len(body)) > a.limit {
		payloadTooLarge(w)
		return nil, false
	}

	return body, true
}
