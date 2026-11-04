package providers

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

const accountQueryTimeout = 5 * time.Second

// QuotaService orchestrates concurrent official quota queries across all registered accounts.
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
	return &QuotaService{
		adminService: adminService,
		adapters:     adapters,
		logger:       logger,
	}
}

// QueryAllAccounts discovers all configured provider accounts and queries their quotas concurrently.
func (s *QuotaService) QueryAllAccounts(ctx context.Context) (ProviderQuotasResponse, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	records, err := s.adminService.List(ctx)
	if err != nil {
		return ProviderQuotasResponse{
			Snapshots: []ProviderQuotaSnapshot{},
			FetchedAt: now,
		}, err
	}

	if len(records) == 0 {
		return ProviderQuotasResponse{
			Snapshots: []ProviderQuotaSnapshot{},
			FetchedAt: now,
		}, nil
	}

	snapshots := make([]ProviderQuotaSnapshot, len(records))
	var wg sync.WaitGroup

	for i, record := range records {
		wg.Add(1)
		go func(idx int, rec Record) {
			defer wg.Done()

			subCtx, cancel := context.WithTimeout(ctx, accountQueryTimeout)
			defer cancel()

			snapshot := ProviderQuotaSnapshot{
				AccountID:    rec.ID,
				AccountLabel: rec.Name,
				ProviderID:   rec.Type,
				ProviderName: rec.Name,
				Status:       QuotaStatusUnsupported,
				Scope:        ScopeUnknown,
				Quotas:       []ProviderQuotaMetric{},
				Source: ProviderQuotaSource{
					Type:      "provider_api",
					Name:      rec.Type,
					Scope:     ScopeUnknown,
					FetchedAt: now,
				},
			}

			if !rec.Enabled {
				snapshot.Status = QuotaStatusUnsupported
				snapshot.Error = &ProviderQuotaError{
					Code:    "account_disabled",
					Message: "Conta desabilitada nas configurações",
				}
				snapshots[idx] = snapshot
				return
			}

			if !rec.HasCredential {
				snapshot.Status = QuotaStatusUnconfiguredCredential
				snapshot.Error = &ProviderQuotaError{
					Code:    "unconfigured_credential",
					Message: "Conta sem credencial configurada",
				}
				snapshots[idx] = snapshot
				return
			}

			adapter, supported := s.adapters[rec.Type]
			if !supported {
				snapshot.Status = QuotaStatusUnsupported
				snapshot.Error = &ProviderQuotaError{
					Code:    "unsupported",
					Message: "Provedor não disponibiliza API pública oficial de franquia/saldo com a credencial atual",
				}
				snapshots[idx] = snapshot
				return
			}

			cred, err := s.adminService.credentialOf(subCtx, rec)
			if err != nil {
				s.logger.Error("failed to decrypt credential for quota query",
					slog.String("account_id", rec.ID),
					slog.String("provider_type", rec.Type),
					slog.Any("error", err),
				)
				snapshot.Status = QuotaStatusError
				snapshot.Error = &ProviderQuotaError{
					Code:    "credential_decrypt_failed",
					Message: "Falha ao recuperar credencial da conta",
				}
				snapshots[idx] = snapshot
				return
			}

			result, err := adapter.FetchQuota(subCtx, rec.Endpoint, cred)
			if err != nil {
				s.logger.Error("quota adapter query failed",
					slog.String("account_id", rec.ID),
					slog.String("provider_type", rec.Type),
					slog.Any("error", err),
				)
				snapshot.Status = QuotaStatusError
				snapshot.Error = &ProviderQuotaError{
					Code:    "adapter_error",
					Message: err.Error(),
				}
				snapshots[idx] = snapshot
				return
			}

			// Ensure identifiers reflect the registered account
			result.AccountID = rec.ID
			result.AccountLabel = rec.Name
			result.ProviderID = rec.Type
			result.ProviderName = rec.Name
			snapshots[idx] = result
		}(i, record)
	}

	wg.Wait()

	return ProviderQuotasResponse{
		Snapshots: snapshots,
		FetchedAt: now,
	}, nil
}

