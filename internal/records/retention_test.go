package records

import (
	"errors"
	"os"
	"testing"
	"time"
)

func TestStoreTrimRemovesExpiredRecords(t *testing.T) {
	dir := t.TempDir()
	cfg := testStorageConfig(dir)
	cfg.maxAge = time.Hour
	store := newTestStore(t, cfg)

	for n := 1; n <= 3; n++ {
		if err := store.append(completedRecord(executionID(n), at(n))); err != nil {
			t.Fatalf("append %d: %v", n, err)
		}
	}

	cutoff := at(1).Add(time.Hour + time.Second)
	if err := store.trim(cutoff); err != nil {
		t.Fatalf("trim: %v", err)
	}

	if store.index.len() != 2 {
		t.Fatalf("retained records = %d, want 2", store.index.len())
	}
	if _, ok := store.index.get(executionID(1)); ok {
		t.Fatal("the expired record must be removed")
	}
	if _, ok := store.index.get(executionID(2)); !ok {
		t.Fatal("records within the age window must be kept")
	}
}

func TestStoreTrimEnforcesCountCap(t *testing.T) {
	dir := t.TempDir()
	cfg := testStorageConfig(dir)
	cfg.maxCount = 2
	store := newTestStore(t, cfg)

	for n := 1; n <= 3; n++ {
		if err := store.append(completedRecord(executionID(n), at(n))); err != nil {
			t.Fatalf("append %d: %v", n, err)
		}
	}

	if err := store.trim(at(10)); err != nil {
		t.Fatalf("trim: %v", err)
	}

	if store.index.len() != 2 {
		t.Fatalf("retained records = %d, want 2", store.index.len())
	}
	if _, ok := store.index.get(executionID(1)); ok {
		t.Fatal("the oldest record must be removed before newer ones")
	}
	if _, ok := store.index.get(executionID(3)); !ok {
		t.Fatal("the newest record must be kept")
	}
	if count := storedLineCount(t, dir); count != 2 {
		t.Fatalf("stored lines = %d, want 2", count)
	}
}

func TestStoreTrimEnforcesStorageCap(t *testing.T) {
	dir := t.TempDir()
	cfg := testStorageConfig(dir)
	store := newTestStore(t, cfg)

	for n := 1; n <= 3; n++ {
		if err := store.append(completedRecord(executionID(n), at(n))); err != nil {
			t.Fatalf("append %d: %v", n, err)
		}
	}

	line, err := encodeRecord(completedRecord(executionID(1), at(1)))
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	cfg.maxStorageBytes = int64(2 * len(line))
	store.cfg.maxStorageBytes = cfg.maxStorageBytes

	if err := store.trim(at(10)); err != nil {
		t.Fatalf("trim: %v", err)
	}

	if store.index.len() != 2 {
		t.Fatalf("retained records = %d, want 2", store.index.len())
	}
	if _, ok := store.index.get(executionID(1)); ok {
		t.Fatal("the oldest record must be removed to satisfy the storage cap")
	}
}

func TestStoreAppendTrimFailurePreventsCapExceed(t *testing.T) {
	dir := t.TempDir()
	cfg := testStorageConfig(dir)
	cfg.maxCount = 1
	failing := true
	ops := &hookOps{renameHook: func(oldPath, newPath string) error {
		if failing {
			return errors.New("rename denied")
		}

		return osFileOps{}.rename(oldPath, newPath)
	}}
	store := newTestStoreWithOps(t, cfg, ops, func() time.Time { return at(0) })

	if err := store.append(completedRecord(executionID(1), at(1))); err != nil {
		t.Fatalf("append first: %v", err)
	}

	if err := store.append(completedRecord(executionID(2), at(2))); err == nil {
		t.Fatal("append beyond the count cap must fail when retention cannot trim")
	}
	if !store.counters.historyDegraded() {
		t.Fatal("failed retention must mark history degraded")
	}
	if store.counters.snapshot().RetentionFailed == 0 {
		t.Fatal("failed retention must be counted")
	}
	if store.index.len() != 1 {
		t.Fatalf("index changed after a failed trim: %d records", store.index.len())
	}

	data, err := os.ReadFile(segmentPath(dir, 1))
	if err != nil {
		t.Fatalf("read segment: %v", err)
	}
	if count := len(splitLines(data)); count != 1 {
		t.Fatalf("source segment changed after a failed trim: %d lines", count)
	}

	failing = false
	if err := store.append(completedRecord(executionID(2), at(2))); err != nil {
		t.Fatalf("append after retention recovered: %v", err)
	}
	if _, ok := store.index.get(executionID(1)); ok {
		t.Fatal("the oldest record must be trimmed before the new record is accepted")
	}
	if _, ok := store.index.get(executionID(2)); !ok {
		t.Fatal("the new record must be retained after trim recovered")
	}
}

func TestStoreAppendTrimsBeforeStorageCap(t *testing.T) {
	dir := t.TempDir()
	cfg := testStorageConfig(dir)
	line, err := encodeRecord(completedRecord(executionID(1), at(1)))
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	cfg.maxStorageBytes = int64(2 * len(line))
	store := newTestStore(t, cfg)

	for n := 1; n <= 3; n++ {
		if err := store.append(completedRecord(executionID(n), at(n))); err != nil {
			t.Fatalf("append %d: %v", n, err)
		}
		if stored := storedSegmentBytes(t, dir); stored > cfg.maxStorageBytes {
			t.Fatalf("stored bytes = %d, want at most %d", stored, cfg.maxStorageBytes)
		}
	}

	if store.index.len() != 2 {
		t.Fatalf("retained records = %d, want 2", store.index.len())
	}
	if _, ok := store.index.get(executionID(1)); ok {
		t.Fatal("the oldest record must be removed before the new record is accepted")
	}
	if _, ok := store.index.get(executionID(3)); !ok {
		t.Fatal("the newest record must be retained")
	}
}

func TestStoreStartFailsWhenModeCannotBeRestricted(t *testing.T) {
	ops := &hookOps{chmodHook: func(string, os.FileMode) error { return errors.New("permission denied") }}
	store := newStore(testStorageConfig(t.TempDir()), testLogger(), ops, func() time.Time { return at(0) }, newIndex(), newCounters())

	if err := store.start(); err == nil {
		t.Fatal("start must fail when the records directory cannot be restricted")
	}
}

func TestStoreCompactionCloseFailureDisablesWrites(t *testing.T) {
	dir := t.TempDir()
	cfg := testStorageConfig(dir)
	cfg.maxCount = 1
	ops := &hookOps{openTempHook: func(path string) (appendFile, error) {
		file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			return nil, err
		}

		return &hookFile{File: file, closeHook: func() error { return errors.New("close denied") }}, nil
	}}
	store := newTestStoreWithOps(t, cfg, ops, func() time.Time { return at(0) })

	if err := store.append(completedRecord(executionID(1), at(1))); err != nil {
		t.Fatalf("append first: %v", err)
	}
	if err := store.append(completedRecord(executionID(2), at(2))); err == nil {
		t.Fatal("append must fail when the compaction temporary file cannot be closed")
	}
	if !store.writesDisabled {
		t.Fatal("a compaction close error must disable further writes")
	}
	if store.counters.snapshot().CloseFailed == 0 || !store.counters.historyDegraded() {
		t.Fatal("compaction close failure must be counted and degrade history")
	}

	data, err := os.ReadFile(segmentPath(dir, 1))
	if err != nil {
		t.Fatalf("read segment: %v", err)
	}
	if count := len(splitLines(data)); count != 1 {
		t.Fatalf("source segment changed after a failed compaction: %d lines", count)
	}

	if err := store.append(completedRecord(executionID(3), at(3))); !errors.Is(err, errWritesDisabled) {
		t.Fatalf("append after close failure = %v, want errWritesDisabled", err)
	}
}
