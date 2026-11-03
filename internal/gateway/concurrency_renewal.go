package gateway

import (
	"context"
	"errors"
	"time"

	"aihub/internal/concurrency"
)

func (h *responsesHandler) startConcurrencyRenewal(parent context.Context, executionID string, cancelExecution context.CancelFunc) func() {
	if h.concurrency == nil {
		return func() {}
	}
	ctx, cancel := context.WithCancel(parent)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(concurrency.DefaultLeaseTTL / 3)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := h.concurrency.RenewLease(ctx, executionID); err != nil {
					if errors.Is(err, concurrency.ErrNotFound) {
						h.logger.WarnContext(ctx, "concurrency lease expired during execution",
							"execution_id", executionID)
						cancelExecution()
						return
					}
					h.logger.WarnContext(ctx, "concurrency lease renew failed",
						"execution_id", executionID, "error", err)
				}
			}
		}
	}()
	return func() {
		cancel()
		<-done
	}
}
