package records

import "aihub/internal/execution"

// recordFromExecution projects one immutable E05 terminal result onto the
// schema-v1 record. The caller has already validated the finished value.
func recordFromExecution(finished execution.FinishedExecution) (Record, error) {
	record := Record{
		SchemaVersion:     SchemaVersion,
		ExecutionID:       finished.Result.ExecutionID,
		ProviderID:        finished.Target.ProviderID,
		StartedAt:         finished.StartedAt.UTC(),
		DurationNS:        int64(finished.Result.Duration),
		Outcome:           string(finished.Result.Outcome),
		Phase:             string(finished.Result.Phase),
		ResponseCommitted: finished.Result.Committed,
	}
	record.CompletedAt = record.StartedAt.Add(finished.Result.Duration)

	if finished.RequestID != "" {
		record.RequestID = pointer(finished.RequestID)
	}
	if finished.Target.ModelID != "" {
		record.ModelID = pointer(finished.Target.ModelID)
	}
	if finished.Result.Cause != "" {
		record.Cause = pointer(string(finished.Result.Cause))
	}
	if finished.Result.UpstreamStatus != 0 {
		record.UpstreamStatus = pointer(finished.Result.UpstreamStatus)
	}

	if err := record.Validate(); err != nil {
		return Record{}, err
	}

	return record, nil
}
