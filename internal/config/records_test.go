package config_test

import (
	"strings"
	"testing"
	"time"

	"aihub/internal/config"
)

func TestLoadUsesRecordsDefaults(t *testing.T) {
	unsetAIHUBEnv(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.Records.MaxAge != 720*time.Hour {
		t.Fatalf("records.max_age = %v, want 720h", cfg.Records.MaxAge)
	}
	if cfg.Records.MaxCount != 10000 {
		t.Fatalf("records.max_count = %d, want 10000", cfg.Records.MaxCount)
	}
	if cfg.Records.MaxStorageBytes != 134217728 {
		t.Fatalf("records.max_storage_bytes = %d, want 134217728", cfg.Records.MaxStorageBytes)
	}
	if cfg.Records.SegmentBytes != 8388608 {
		t.Fatalf("records.segment_bytes = %d, want 8388608", cfg.Records.SegmentBytes)
	}
	if cfg.Records.RetentionInterval != time.Hour {
		t.Fatalf("records.retention_interval = %v, want 1h", cfg.Records.RetentionInterval)
	}
}

func TestLoadAppliesRecordsEnvironmentOverrides(t *testing.T) {
	unsetAIHUBEnv(t)

	t.Setenv(config.EnvRecordsMaxAge, "48h")
	t.Setenv(config.EnvRecordsMaxCount, "500")
	t.Setenv(config.EnvRecordsMaxStorageBytes, "33554432")
	t.Setenv(config.EnvRecordsSegmentBytes, "4194304")
	t.Setenv(config.EnvRecordsRetentionInterval, "15m")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.Records.MaxAge != 48*time.Hour ||
		cfg.Records.MaxCount != 500 ||
		cfg.Records.MaxStorageBytes != 33554432 ||
		cfg.Records.SegmentBytes != 4194304 ||
		cfg.Records.RetentionInterval != 15*time.Minute {
		t.Fatalf("records config = %+v, want the environment overrides", cfg.Records)
	}
}

func TestLoadRejectsInvalidRecordsConfiguration(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{name: "age below range", key: config.EnvRecordsMaxAge, value: "30m"},
		{name: "age above range", key: config.EnvRecordsMaxAge, value: "8761h"},
		{name: "age unparsable", key: config.EnvRecordsMaxAge, value: "thirty days"},
		{name: "count zero", key: config.EnvRecordsMaxCount, value: "0"},
		{name: "count above range", key: config.EnvRecordsMaxCount, value: "100001"},
		{name: "count unparsable", key: config.EnvRecordsMaxCount, value: "many"},
		{name: "storage below range", key: config.EnvRecordsMaxStorageBytes, value: "1048576"},
		{name: "storage above range", key: config.EnvRecordsMaxStorageBytes, value: "2147483648"},
		{name: "segment below range", key: config.EnvRecordsSegmentBytes, value: "32768"},
		{name: "segment above range", key: config.EnvRecordsSegmentBytes, value: "67108864"},
		{name: "interval below range", key: config.EnvRecordsRetentionInterval, value: "30s"},
		{name: "interval above range", key: config.EnvRecordsRetentionInterval, value: "25h"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			unsetAIHUBEnv(t)
			t.Setenv(test.key, test.value)

			_, err := config.Load()
			if err == nil {
				t.Fatalf("Load() must reject %s=%s", test.key, test.value)
			}
			if !strings.Contains(err.Error(), "records.") {
				t.Fatalf("error = %q, want it to identify the records setting", err)
			}
		})
	}
}

func TestLoadRejectsSegmentSizeIncompatibleWithStorageCap(t *testing.T) {
	tests := []struct {
		name    string
		storage string
		segment string
	}{
		{name: "segment too close to storage", storage: "16777216", segment: "16711680"},
		{name: "segment above half storage", storage: "33554432", segment: "20971520"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			unsetAIHUBEnv(t)
			t.Setenv(config.EnvRecordsMaxStorageBytes, test.storage)
			t.Setenv(config.EnvRecordsSegmentBytes, test.segment)

			_, err := config.Load()
			if err == nil {
				t.Fatalf("Load() must reject storage=%s segment=%s", test.storage, test.segment)
			}
			if !strings.Contains(err.Error(), "records.segment_bytes") {
				t.Fatalf("error = %q, want it to identify records.segment_bytes", err)
			}
		})
	}
}
