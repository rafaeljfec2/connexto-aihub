package main

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

type e07SSEObservation struct {
	Status      int
	DataEvents  int
	FirstDataAt time.Duration
	LastDataAt  time.Duration
	CompleteAt  time.Duration
	Progressive bool
}

func e07TemporaryGatewayKey(providerKey string) (string, error) {
	secret := make([]byte, 48)
	if _, err := rand.Read(secret); err != nil {
		return "", errors.New("temporary gateway key generation failed")
	}
	key := base64.RawStdEncoding.EncodeToString(secret)
	if key == providerKey {
		return "", errors.New("temporary gateway key collided with provider key")
	}
	return key, nil
}

func e07MeasureSSE(url, gatewayKey string, payload []byte) (e07SSEObservation, error) {
	request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return e07SSEObservation{}, errors.New("streaming request could not be created")
	}
	request.Header.Set("Authorization", "Bearer "+gatewayKey)
	request.Header.Set("Content-Type", "application/json")
	started := time.Now()
	response, err := (&http.Client{
		Timeout:       e07LiveRequestTimeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
	}).Do(request)
	if err != nil {
		return e07SSEObservation{}, errors.New("streaming request failed")
	}
	defer func() { _ = response.Body.Close() }()
	observation := e07SSEObservation{Status: response.StatusCode}
	if response.StatusCode != http.StatusOK || !strings.HasPrefix(response.Header.Get("Content-Type"), "text/event-stream") {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 1<<20))
		observation.CompleteAt = time.Since(started)
		return observation, nil
	}
	scanner := bufio.NewScanner(response.Body)
	scanner.Buffer(make([]byte, 4*1024), 1<<20)
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "data:") && strings.TrimSpace(strings.TrimPrefix(scanner.Text(), "data:")) != "[DONE]" {
			observation.DataEvents++
			at := time.Since(started)
			if observation.FirstDataAt == 0 {
				observation.FirstDataAt = at
			}
			observation.LastDataAt = at
		}
	}
	observation.CompleteAt = time.Since(started)
	if scanner.Err() != nil {
		return e07SSEObservation{}, errors.New("streaming response could not be read")
	}
	observation.Progressive = observation.DataEvents >= 2 &&
		observation.FirstDataAt < observation.LastDataAt && observation.LastDataAt < observation.CompleteAt
	return observation, nil
}
