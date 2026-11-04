package providers

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

const accountQueryTimeout = 5 * time.Second

// QuotaService orchestrates independent official quota reads for configured accounts.
type QuotaService struct {
	adminService *Service
	adapters     map[string]QuotaAdapter
	logger       *slog.Logger
}

// NewQuotaService constructs a quota orchestration service.
func NewQuotaService(adminService *Service, adapters map[string]QuotaAdapter, logger *slog.Logger) *QuotaService {
	if adapters == nil {
		adapters = make(map[string]QuotaAdapter)
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &QuotaService{adminService: adminService, adapters: adapters, logger: logger}
}

// QueryAllAccounts returns one snapshot per supported credential scope and configured account.
func (s *QuotaService) QueryAllAccounts(ctx context.Context) (ProviderQuotasResponse, error) {
	fetchedAt := time.Now().UTC().Format(time.RFC3339)
	records, err := s.adminService.List(ctx)
	if err != nil {
		return ProviderQuotasResponse{Snapshots: []ProviderQuotaSnapshot{}, FetchedAt: fetchedAt}, err
	}
	if len(records) == 0 {
		return ProviderQuotasResponse{Snapshots: []ProviderQuotaSnapshot{}, FetchedAt: fetchedAt}, nil
	}

	perAccount := make([][]ProviderQuotaSnapshot, len(records))
	var wait sync.WaitGroup
	for index, record := range records {
		wait.Add(1)
		go func(index int, record Record) {
			defer wait.Done()
			perAccount[index] = s.queryConfiguredAccount(ctx, record)
		}(index, record)
	}
	wait.Wait()

	snapshots := make([]ProviderQuotaSnapshot, 0, len(records)*2)
	for _, accountSnapshots := range perAccount {
		snapshots = append(snapshots, accountSnapshots...)
	}
	return ProviderQuotasResponse{Snapshots: snapshots, FetchedAt: fetchedAt}, nil
}
