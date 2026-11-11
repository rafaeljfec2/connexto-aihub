package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	minRecordsAge               = time.Hour
	maxRecordsAge               = 8760 * time.Hour
	minRecordsCount             = 1
	maxRecordsCount             = 100000
	minRecordsStorageBytes      = 16 << 20
	maxRecordsStorageBytes      = 1 << 30
	minRecordsSegmentBytes      = 64 << 10
	maxRecordsSegmentBytes      = 32 << 20
	minRecordsRetentionInterval = time.Minute
	maxRecordsRetentionInterval = 24 * time.Hour
	recordsSegmentStorageMargin = 16 << 10
)

// Records groups the bounded execution-record retention settings.
type Records struct {
	MaxAge            time.Duration
	MaxCount          int
	MaxStorageBytes   int64
	SegmentBytes      int64
	RetentionInterval time.Duration
}

func defaultRecords() Records {
	return Records{
		MaxAge:            720 * time.Hour,
		MaxCount:          10000,
		MaxStorageBytes:   134217728,
		SegmentBytes:      8388608,
		RetentionInterval: time.Hour,
	}
}

func applyRecordsEnvironment(records *Records) error {
	durations := []struct {
		field  string
		env    string
		target *time.Duration
	}{
		{"records.max_age", EnvRecordsMaxAge, &records.MaxAge},
		{"records.retention_interval", EnvRecordsRetentionInterval, &records.RetentionInterval},
	}

	for _, item := range durations {
		value, ok := os.LookupEnv(item.env)
		if !ok {
			continue
		}

		parsed, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("%s (%s): %w", item.field, item.env, err)
		}

		*item.target = parsed
	}

	if value, ok := os.LookupEnv(EnvRecordsMaxCount); ok {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("records.max_count (%s) must be an integer: %w", EnvRecordsMaxCount, err)
		}

		records.MaxCount = parsed
	}

	integers := []struct {
		field  string
		env    string
		target *int64
	}{
		{"records.max_storage_bytes", EnvRecordsMaxStorageBytes, &records.MaxStorageBytes},
		{"records.segment_bytes", EnvRecordsSegmentBytes, &records.SegmentBytes},
	}

	for _, item := range integers {
		value, ok := os.LookupEnv(item.env)
		if !ok {
			continue
		}

		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("%s (%s) must be an integer: %w", item.field, item.env, err)
		}

		*item.target = parsed
	}

	return nil
}

func (r Records) validate() []error {
	var errs []error

	if r.MaxAge < minRecordsAge || r.MaxAge > maxRecordsAge {
		errs = append(errs, fmt.Errorf("records.max_age must be between %s and %s, got %s",
			minRecordsAge, maxRecordsAge, r.MaxAge))
	}
	if r.MaxCount < minRecordsCount || r.MaxCount > maxRecordsCount {
		errs = append(errs, fmt.Errorf("records.max_count must be between %d and %d, got %d",
			minRecordsCount, maxRecordsCount, r.MaxCount))
	}
	if r.MaxStorageBytes < minRecordsStorageBytes || r.MaxStorageBytes > maxRecordsStorageBytes {
		errs = append(errs, fmt.Errorf("records.max_storage_bytes must be between %d and %d, got %d",
			minRecordsStorageBytes, maxRecordsStorageBytes, r.MaxStorageBytes))
	}
	if r.SegmentBytes < minRecordsSegmentBytes || r.SegmentBytes > maxRecordsSegmentBytes {
		errs = append(errs, fmt.Errorf("records.segment_bytes must be between %d and %d, got %d",
			minRecordsSegmentBytes, maxRecordsSegmentBytes, r.SegmentBytes))
	}
	if r.RetentionInterval < minRecordsRetentionInterval || r.RetentionInterval > maxRecordsRetentionInterval {
		errs = append(errs, fmt.Errorf("records.retention_interval must be between %s and %s, got %s",
			minRecordsRetentionInterval, maxRecordsRetentionInterval, r.RetentionInterval))
	}

	if r.SegmentBytes > r.MaxStorageBytes-recordsSegmentStorageMargin {
		errs = append(errs, fmt.Errorf("records.segment_bytes must be at least %d bytes smaller than records.max_storage_bytes",
			recordsSegmentStorageMargin))
	}
	if r.SegmentBytes > r.MaxStorageBytes/2 {
		errs = append(errs, fmt.Errorf("records.segment_bytes must not exceed half of records.max_storage_bytes"))
	}

	return errs
}
