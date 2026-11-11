package providers

import (
	"context"
	"net/url"
	"strings"
)

type loopbackProxyRewriteRepository struct {
	next   Repository
	target *url.URL
}

// WrapLoopbackProxyEndpoint wraps repo so persisted Connexto Proxy provider
// records pointing to loopback (for example 127.0.0.1:9090 or localhost:9090
// from host-native runs) resolve to proxyBaseURL inside containers without
// mutating stored database records.
func WrapLoopbackProxyEndpoint(repo Repository, proxyBaseURL string) Repository {
	if repo == nil {
		return nil
	}

	target, err := url.Parse(strings.TrimSpace(proxyBaseURL))
	if err != nil || target.Host == "" || isLoopbackHostname(target.Hostname()) {
		return repo
	}

	return &loopbackProxyRewriteRepository{
		next:   repo,
		target: target,
	}
}

func (r *loopbackProxyRewriteRepository) Create(ctx context.Context, record NewRecord) (Record, error) {
	created, err := r.next.Create(ctx, record)
	if err != nil {
		return Record{}, err
	}
	return r.rewriteRecord(created), nil
}

func (r *loopbackProxyRewriteRepository) EnsureConfigured(ctx context.Context, record NewRecord) (Record, error) {
	ensured, err := r.next.EnsureConfigured(ctx, record)
	if err != nil {
		return Record{}, err
	}
	return r.rewriteRecord(ensured), nil
}

func (r *loopbackProxyRewriteRepository) Get(ctx context.Context, id string) (Record, error) {
	record, err := r.next.Get(ctx, id)
	if err != nil {
		return Record{}, err
	}
	return r.rewriteRecord(record), nil
}

func (r *loopbackProxyRewriteRepository) List(ctx context.Context) ([]Record, error) {
	records, err := r.next.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Record, len(records))
	for i, record := range records {
		out[i] = r.rewriteRecord(record)
	}
	return out, nil
}

func (r *loopbackProxyRewriteRepository) Update(ctx context.Context, id string, changes Changes) (Record, error) {
	updated, err := r.next.Update(ctx, id, changes)
	if err != nil {
		return Record{}, err
	}
	return r.rewriteRecord(updated), nil
}

func (r *loopbackProxyRewriteRepository) SetEnabled(ctx context.Context, id string, enabled bool) (Record, error) {
	updated, err := r.next.SetEnabled(ctx, id, enabled)
	if err != nil {
		return Record{}, err
	}
	return r.rewriteRecord(updated), nil
}

func (r *loopbackProxyRewriteRepository) SealedCredential(ctx context.Context, id string) ([]byte, error) {
	return r.next.SealedCredential(ctx, id)
}

func (r *loopbackProxyRewriteRepository) SealedManagementCredential(ctx context.Context, id string) ([]byte, error) {
	return r.next.SealedManagementCredential(ctx, id)
}

func (r *loopbackProxyRewriteRepository) rewriteRecord(record Record) Record {
	if !isConnextoProxyProviderType(record.Type) {
		return record
	}

	parsed, err := url.Parse(strings.TrimSpace(record.Endpoint))
	if err != nil || !isLoopbackHostname(parsed.Hostname()) {
		return record
	}

	parsed.Scheme = r.target.Scheme
	parsed.Host = r.target.Host
	if parsed.Path == "" {
		parsed.Path = r.target.Path
	}
	record.Endpoint = parsed.String()
	return record
}

func isConnextoProxyProviderType(providerType string) bool {
	switch providerType {
	case TypeAntigravity, TypeCursor, TypeConnextoProxy:
		return true
	default:
		return false
	}
}

func isLoopbackHostname(host string) bool {
	switch strings.ToLower(strings.TrimSpace(host)) {
	case "127.0.0.1", "localhost", "::1":
		return true
	default:
		return false
	}
}

