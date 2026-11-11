package records

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func appendRaw(t *testing.T, path, raw string) {
	t.Helper()

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatalf("open raw segment: %v", err)
	}
	defer func() { _ = file.Close() }()

	if _, err := file.WriteString(raw); err != nil {
		t.Fatalf("append raw bytes: %v", err)
	}
}

func segmentPath(dir string, sequence int64) string {
	return filepath.Join(dir, segmentFileName(sequence))
}

func TestStoreRecoversValidPrefixFromIncompleteTail(t *testing.T) {
	dir := t.TempDir()
	cfg := testStorageConfig(dir)

	store := newTestStore(t, cfg)
	first := completedRecord(executionID(1), at(1))
	second := completedRecord(executionID(2), at(2))
	if err := store.append(first); err != nil {
		t.Fatalf("append first: %v", err)
	}
	if err := store.append(second); err != nil {
		t.Fatalf("append second: %v", err)
	}
	if err := store.close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	path := segmentPath(dir, 1)
	appendRaw(t, path, `{"schema_version":1,"execution_id":"`)

	reopened := newTestStore(t, cfg)
	if reopened.index.len() != 2 {
		t.Fatalf("recovered records = %d, want 2", reopened.index.len())
	}
	if !reopened.counters.historyDegraded() {
		t.Fatal("incomplete tail must mark history degraded")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read segment: %v", err)
	}
	if !strings.HasSuffix(string(data), "\n") {
		t.Fatal("recovery must truncate the source to the last validated LF boundary")
	}
	if lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n"); len(lines) != 2 {
		t.Fatalf("recovered lines = %d, want 2", len(lines))
	}

	if err := reopened.append(completedRecord(executionID(3), at(3))); err != nil {
		t.Fatalf("append after confirmed truncation: %v", err)
	}
}

func TestStoreRecoversValidPrefixFromCorruptMiddleLine(t *testing.T) {
	dir := t.TempDir()
	cfg := testStorageConfig(dir)

	store := newTestStore(t, cfg)
	if err := store.append(completedRecord(executionID(1), at(1))); err != nil {
		t.Fatalf("append: %v", err)
	}
	if err := store.close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	valid, err := encodeRecord(completedRecord(executionID(2), at(2)))
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	appendRaw(t, segmentPath(dir, 1), "{\"schema_version\":1,\"broken\":true}\n"+string(valid))

	reopened := newTestStore(t, cfg)
	if reopened.index.len() != 1 {
		t.Fatalf("recovered records = %d, want only the valid prefix", reopened.index.len())
	}
	if _, ok := reopened.index.get(executionID(1)); !ok {
		t.Fatal("the validated prefix must stay queryable")
	}
	if _, ok := reopened.index.get(executionID(2)); ok {
		t.Fatal("records after the corrupt line must not be indexed")
	}

	data, err := os.ReadFile(segmentPath(dir, 1))
	if err != nil {
		t.Fatalf("read segment: %v", err)
	}
	if strings.Contains(string(data), "broken") || strings.Contains(string(data), executionID(2)) {
		t.Fatal("the unvalidated suffix must be truncated without being copied")
	}
}

func TestStoreRecoveryReadErrorDisablesWritesAndKeepsStartup(t *testing.T) {
	dir := t.TempDir()
	cfg := testStorageConfig(dir)

	store := newTestStore(t, cfg)
	if err := store.append(completedRecord(executionID(1), at(1))); err != nil {
		t.Fatalf("append: %v", err)
	}
	if err := store.close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	ops := &hookOps{openReadHook: func(string) (io.ReadCloser, error) {
		return nil, errors.New("read denied")
	}}

	reopened := newTestStoreWithOps(t, cfg, ops, func() time.Time { return at(0) })
	if reopened.index.len() != 0 {
		t.Fatalf("recovered records = %d, want 0", reopened.index.len())
	}
	if !reopened.counters.historyDegraded() {
		t.Fatal("read failure must mark history degraded")
	}
	if reopened.counters.snapshot().RecoveryFailed == 0 {
		t.Fatal("read failure must be counted")
	}
	if err := reopened.append(completedRecord(executionID(2), at(2))); err == nil {
		t.Fatal("writes must stay disabled after an unconfirmed read")
	}
}

func TestStoreTruncationFailureKeepsSourceAndDisablesWrites(t *testing.T) {
	dir := t.TempDir()
	cfg := testStorageConfig(dir)

	store := newTestStore(t, cfg)
	if err := store.append(completedRecord(executionID(1), at(1))); err != nil {
		t.Fatalf("append: %v", err)
	}
	if err := store.close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	path := segmentPath(dir, 1)
	appendRaw(t, path, "{\"incomplete\":")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read before: %v", err)
	}

	ops := &hookOps{}
	wrapAppend(ops, func(file *hookFile) {
		file.truncateHook = func(int64) error { return errors.New("truncate denied") }
	})

	reopened := newTestStoreWithOps(t, cfg, ops, func() time.Time { return at(0) })
	if reopened.index.len() != 1 {
		t.Fatalf("recovered records = %d, want the valid prefix", reopened.index.len())
	}
	if !reopened.counters.historyDegraded() {
		t.Fatal("unconfirmed truncation must mark history degraded")
	}
	if err := reopened.append(completedRecord(executionID(2), at(2))); err == nil {
		t.Fatal("writes must stay disabled when truncation cannot be confirmed")
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read after: %v", err)
	}
	if string(before) != string(after) {
		t.Fatal("the source segment must stay untouched when truncation fails")
	}
}
