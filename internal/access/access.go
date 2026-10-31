package access

import (
	"context"
	"errors"
	"strings"
)

// Gateway identity and admission defaults.
const (
	PrincipalID               = "gateway-client"
	DefaultMaxBodyBytes int64 = 32 << 20
	MinBodyBytes        int64 = 1 << 10
	MaxBodyBytesCeiling int64 = 128 << 20
)

var loopbackHosts = []string{"localhost", "127.0.0.1", "::1"}

// PolicyConfig carries validated gateway settings into the immutable runtime
// policy.
type PolicyConfig struct {
	APIKey               string
	AllowedHosts         []string
	AllowedHostsExplicit bool
	AllowedOrigins       []string
	MaxBodyBytes         int64
}

// Policy is an immutable, concurrency-safe access-control policy.
type Policy struct {
	keyDigest      [32]byte
	allowedHosts   map[string]struct{}
	allowedOrigins map[string]struct{}
	maxBodyBytes   int64
}

// NewPolicy validates and normalizes gateway access settings.
func NewPolicy(cfg PolicyConfig) (Policy, error) {
	if err := ValidateAPIKey(cfg.APIKey); err != nil {
		return Policy{}, err
	}

	limit := cfg.MaxBodyBytes
	if limit == 0 {
		limit = DefaultMaxBodyBytes
	}

	if limit < MinBodyBytes || limit > MaxBodyBytesCeiling {
		return Policy{}, errors.New("max body bytes must be between 1024 and 134217728")
	}

	hosts := cfg.AllowedHosts
	if len(hosts) == 0 {
		if cfg.AllowedHostsExplicit {
			return Policy{}, errors.New("allowed hosts must not be empty when explicitly set")
		}

		hosts = loopbackHosts
	}

	normalizedHosts, err := ParseAllowedHosts(strings.Join(hosts, ","))
	if err != nil {
		return Policy{}, err
	}

	origins, err := ParseAllowedOrigins(strings.Join(cfg.AllowedOrigins, ","))
	if err != nil {
		return Policy{}, err
	}

	policy := Policy{
		keyDigest:      digestKey(cfg.APIKey),
		allowedHosts:   make(map[string]struct{}, len(normalizedHosts)),
		allowedOrigins: make(map[string]struct{}, len(origins)),
		maxBodyBytes:   limit,
	}

	for _, host := range normalizedHosts {
		policy.allowedHosts[host] = struct{}{}
	}

	for _, origin := range origins {
		policy.allowedOrigins[origin] = struct{}{}
	}

	return policy, nil
}

// MaxBodyBytes returns the effective request-body bound.
func (p Policy) MaxBodyBytes() int64 {
	if p.maxBodyBytes == 0 {
		return DefaultMaxBodyBytes
	}

	return p.maxBodyBytes
}

func (p Policy) hostAllowed(host string) bool {
	_, ok := p.allowedHosts[host]
	return ok
}

func (p Policy) originAllowed(origin string) bool {
	_, ok := p.allowedOrigins[origin]
	return ok
}

// Metadata is the in-memory request identity envelope.
type Metadata struct {
	Principal         string
	ProviderSessionID string
	Host              string
	Origin            string
}

type metadataKey struct{}

// WithMetadata attaches request identity to a context.
func WithMetadata(ctx context.Context, metadata Metadata) context.Context {
	return context.WithValue(ctx, metadataKey{}, metadata)
}

// MetadataFrom reads request identity from a context.
func MetadataFrom(ctx context.Context) (Metadata, bool) {
	metadata, ok := ctx.Value(metadataKey{}).(Metadata)
	return metadata, ok
}
