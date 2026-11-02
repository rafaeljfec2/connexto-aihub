package support

import (
	"crypto/sha256"
	"errors"
	"hash"
	"io"
	"net/http"
	"sync"
)

type E07ResponseFidelityObservation struct {
	StatusCode   int
	ContentType  string
	BodyDigest   [32]byte
	BodyComplete bool
}

type E07DigestReadCloser struct {
	Body   io.ReadCloser
	Hasher hash.Hash
	Finish func([32]byte, bool)
	Once   sync.Once
}

func NewE07DigestReadCloser(body io.ReadCloser, finish func([32]byte, bool)) *E07DigestReadCloser {
	return &E07DigestReadCloser{Body: body, Hasher: sha256.New(), Finish: finish}
}

func (body *E07DigestReadCloser) Read(buffer []byte) (int, error) {
	count, err := body.Body.Read(buffer)
	if count > 0 {
		_, _ = body.Hasher.Write(buffer[:count])
	}
	if err == io.EOF {
		body.Complete(true)
	} else if err != nil {
		body.Complete(false)
	}
	return count, err
}

func (body *E07DigestReadCloser) Close() error {
	body.Complete(false)
	return body.Body.Close()
}

func (body *E07DigestReadCloser) Complete(complete bool) {
	body.Once.Do(func() {
		var digest [32]byte
		if complete {
			copy(digest[:], body.Hasher.Sum(nil))
		}
		body.Finish(digest, complete)
	})
}

func E07ClientResponseObservation(response *http.Response, body []byte, readErr error) E07ResponseFidelityObservation {
	observation := E07ResponseFidelityObservation{}
	if response != nil {
		observation.StatusCode = response.StatusCode
		observation.ContentType = response.Header.Get("Content-Type")
	}
	if readErr == nil {
		observation.BodyDigest = sha256.Sum256(body)
		observation.BodyComplete = true
	}
	return observation
}

func E07ResponseObservationsMatch(upstream, client E07ResponseFidelityObservation) bool {
	return upstream.BodyComplete && client.BodyComplete &&
		upstream.StatusCode == client.StatusCode &&
		upstream.ContentType == client.ContentType &&
		upstream.BodyDigest == client.BodyDigest
}

type FailingReadCloser struct{}

func (FailingReadCloser) Read([]byte) (int, error) { return 0, errors.New("synthetic read error") }

func (FailingReadCloser) Close() error { return nil }
