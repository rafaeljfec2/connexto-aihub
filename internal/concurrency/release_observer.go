package concurrency

import (
	"context"

	"aihub/internal/execution"
)

type releaseObserver struct {
	service *Service
	next    execution.ResultObserver
}

// NewReleaseObserver releases the concurrency slot when E05 finishes, then delegates.
func NewReleaseObserver(service *Service, next execution.ResultObserver) execution.ResultObserver {
	if service == nil {
		return next
	}
	return &releaseObserver{service: service, next: next}
}

func (o *releaseObserver) OnExecutionFinished(finished execution.FinishedExecution) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultOperationTimeout)
	defer cancel()
	_ = o.service.Release(ctx, finished.Result.ExecutionID)
	if o.next != nil {
		o.next.OnExecutionFinished(finished)
	}
}
