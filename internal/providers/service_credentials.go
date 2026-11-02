package providers

import (
	"context"
	"strings"
)

func (s *Service) credentialOf(ctx context.Context, record Record) (string, error) {
	sealed, err := s.repo.SealedCredential(ctx, record.ID)
	if err != nil {
		return "", err
	}
	if sealed == nil {
		return "", nil
	}
	return s.cipher.Open(record.ID, sealed)
}

func (s *Service) sealOptional(id, credential string) ([]byte, error) {
	if credential == "" {
		return nil, nil
	}
	return s.seal(id, credential)
}

func (s *Service) seal(id, credential string) ([]byte, error) {
	if err := validateCredential(credential); err != nil {
		return nil, err
	}
	return s.cipher.Seal(id, strings.TrimSpace(credential))
}
