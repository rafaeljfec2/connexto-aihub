package records

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testStorageConfig(dir string) storageConfig {
	return storageConfig{
		dir:               dir,
		maxAge:            720 * time.Hour,
		maxCount:          1000,
		maxStorageBytes:   64 << 20,
		segmentBytes:      1 << 20,
		retentionInterval: time.Hour,
	}
}

func newTestStore(t *testing.T, cfg storageConfig) *store {
	t.Helper()

	return newTestStoreWithOps(t, cfg, osFileOps{}, func() time.Time { return at(0) })
}

func newTestStoreWithOps(t *testing.T, cfg storageConfig, ops fileOps, now func() time.Time) *store {
	t.Helper()

	store := newStore(cfg, testLogger(), ops, now, newIndex(), newCounters())
	if err := store.start(); err != nil {
		t.Fatalf("store.start() returned error: %v", err)
	}

	t.Cleanup(func() { _ = store.close() })

	return store
}

func TestStoreAppendsAndRecoversRecords(t *testing.T) {
	dir := t.TempDir()
	store := newTestStore(t, testStorageConfig(dir))

	first := completedRecord(executionID(1), at(1))
	second := failedRecord(executionID(2), at(2), CauseTransport)

	if err := store.append(first); err != nil {
		t.Fatalf("append first: %v", err)
	}
	if err := store.append(second); err != nil {
		t.Fatalf("append second: %v", err)
	}

	if record, ok := store.index.get(first.ExecutionID); !ok || !record.CompletedAt.Equal(first.CompletedAt) {
		t.Fatalf("indexed record = %+v, want the appended value", record)
	}

	if err := store.close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	reopened := newTestStore(t, testStorageConfig(dir))
	if reopened.index.len() != 2 {
		t.Fatalf("recovered records = %d, want 2", reopened.index.len())
	}
	if reopened.counters.historyDegraded() {
		t.Fatal("clean recovery must not report degraded history")
	}
}

func TestStoreHandlesShortWrites(t *testing.T) {
	dir := t.TempDir()
	ops := &hookOps{}
	wrapAppend(ops, func(file *hookFile) {
		file.writeHook = func(data []byte) (int, error) {
			if len(data) == 0 {
				return 0, nil
			}

			return file.File.Write(data[:1])
		}
	})

	store := newTestStoreWithOps(t, testStorageConfig(dir), ops, func() time.Time { return at(0) })
	record := completedRecord(executionID(1), at(1))

	if err := store.append(record); err != nil {
		t.Fatalf("append with short writes: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, segmentFileName(1)))
	if err != nil {
		t.Fatalf("read segment: %v", err)
	}

	decoded, err := decodeRecord([]byte(strings.TrimSuffix(string(data), "\n")))
	if err != nil {
		t.Fatalf("decode written line: %v", err)
	}

	if decoded.ExecutionID != record.ExecutionID {
		t.Fatalf("written record = %s, want %s", decoded.ExecutionID, record.ExecutionID)
	}
}

func TestStoreAppendFailureKeepsIndexUnchanged(t *testing.T) {
	dir := t.TempDir()
	ops := &hookOps{}
	attempts := 0
	wrapAppend(ops, func(file *hookFile) {
		file.writeHook = func(data []byte) (int, error) {
			attempts++
			if attempts == 1 {
				return 0, errors.New("disk full")
			}

			return file.File.Write(data)
		}
	})

	store := newTestStoreWithOps(t, testStorageConfig(dir), ops, func() time.Time { return at(0) })
	failed := completedRecord(executionID(1), at(1))

	if err := store.append(failed); err == nil {
		t.Fatal("append must fail when the write fails")
	}

	if _, ok := store.index.get(failed.ExecutionID); ok {
		t.Fatal("failed append must not be query-visible")
	}

	if store.counters.snapshot().AppendFailed == 0 {
		t.Fatal("append failure counter was not incremented")
	}

	if err := store.append(completedRecord(executionID(2), at(2))); err != nil {
		t.Fatalf("later append must still be attempted: %v", err)
	}
}

func TestStoreRollbackFailureDisablesWrites(t *testing.T) {
	dir := t.TempDir()
	ops := &hookOps{}
	wrapAppend(ops, func(file *hookFile) {
		file.writeHook = func(data []byte) (int, error) {
			_, _ = file.File.Write(data[:10])
			return 10, errors.New("write interrupted")
		}
		file.truncateHook = func(int64) error { return errors.New("truncate denied") }
	})

	store := newTestStoreWithOps(t, testStorageConfig(dir), ops, func() time.Time { return at(0) })
	record := completedRecord(executionID(1), at(1))

	if err := store.append(record); err == nil {
		t.Fatal("ambiguous append must return an error")
	}

	snapshot := store.counters.snapshot()
	if snapshot.RollbackFailed == 0 {
		t.Fatal("rollback failure counter was not incremented")
	}
	if !store.counters.historyDegraded() {
		t.Fatal("rollback failure must mark history degraded")
	}
	if err := store.append(completedRecord(executionID(2), at(2))); err == nil {
		t.Fatal("writes must stay disabled after an unconfirmed rollback")
	}
}

func TestStoreRotatesSegments(t *testing.T) {
	dir := t.TempDir()
	cfg := testStorageConfig(dir)
	cfg.segmentBytes = 256
	store := newTestStore(t, cfg)

	for n := 1; n <= 3; n++ {
		if err := store.append(completedRecord(executionID(n), at(n))); err != nil {
			t.Fatalf("append %d: %v", n, err)
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read directory: %v", err)
	}

	segments := 0
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".jsonl") {
			segments++
		}
	}

	if segments < 3 {
		t.Fatalf("segments = %d, want at least one per record", segments)
	}

	page := store.index.page(Filter{Limit: 10})
	if len(page.Records) != 3 {
		t.Fatalf("query across segments returned %d records, want 3", len(page.Records))
	}
}

func TestStoreReportsSyncAndCloseFailures(t *testing.T) {
	t.Run("sync failure on close", func(t *testing.T) {
		dir := t.TempDir()
		ops := &hookOps{}
		wrapAppend(ops, func(file *hookFile) {
			file.syncHook = func() error { return errors.New("sync denied") }
		})

		store := newTestStoreWithOps(t, testStorageConfig(dir), ops, func() time.Time { return at(0) })
		if err := store.append(completedRecord(executionID(1), at(1))); err != nil {
			t.Fatalf("append: %v", err)
		}

		if err := store.close(); err == nil {
			t.Fatal("close must report the sync failure")
		}

		if store.counters.snapshot().SyncUncertain == 0 || !store.counters.historyDegraded() {
			t.Fatal("sync failure must be counted and degrade history")
		}
		if _, ok := store.index.get(executionID(1)); !ok {
			t.Fatal("a completed append must remain queryable after a sync failure")
		}
	})

	t.Run("close failure", func(t *testing.T) {
		dir := t.TempDir()
		ops := &hookOps{}
		wrapAppend(ops, func(file *hookFile) {
			file.closeHook = func() error { return errors.New("close denied") }
		})

		store := newTestStoreWithOps(t, testStorageConfig(dir), ops, func() time.Time { return at(0) })
		if err := store.append(completedRecord(executionID(1), at(1))); err != nil {
			t.Fatalf("append: %v", err)
		}

		if err := store.close(); err == nil {
			t.Fatal("close must report the close failure")
		}

		if store.counters.snapshot().CloseFailed == 0 || !store.counters.historyDegraded() {
			t.Fatal("close failure must be counted and degrade history")
		}
		if _, ok := store.index.get(executionID(1)); !ok {
			t.Fatal("a completed append must remain queryable after a close failure")
		}
	})
}
