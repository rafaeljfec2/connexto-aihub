package support

import (
	"context"
	"errors"
)

type E07FakeRecordReader struct {
	ListRecords    []E07ExecutionRecord
	ListErr        error
	Records        map[string]E07ExecutionRecord
	GetErr         error
	OnList         func()
	ContextErrs    []error
	GetContextErrs []error
}

func (reader *E07FakeRecordReader) List(ctx context.Context) ([]E07ExecutionRecord, error) {
	reader.ContextErrs = append(reader.ContextErrs, ctx.Err())
	if reader.OnList != nil {
		reader.OnList()
	}
	if reader.ListErr != nil {
		return nil, reader.ListErr
	}
	return append([]E07ExecutionRecord(nil), reader.ListRecords...), nil
}

func (reader *E07FakeRecordReader) Get(ctx context.Context, executionID string) (E07ExecutionRecord, error) {
	reader.GetContextErrs = append(reader.GetContextErrs, ctx.Err())
	if reader.GetErr != nil {
		return E07ExecutionRecord{}, reader.GetErr
	}
	record, ok := reader.Records[executionID]
	if !ok {
		return E07ExecutionRecord{}, errors.New("record not found")
	}
	return record, nil
}
