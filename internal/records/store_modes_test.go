package records

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreEnforcesRestrictiveFileModes(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "records")
	store := newTestStore(t, testStorageConfig(dir))

	if err := store.append(completedRecord(executionID(1), at(1))); err != nil {
		t.Fatalf("append: %v", err)
	}

	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat directory: %v", err)
	}
	if dirInfo.Mode().Perm() != 0o700 {
		t.Fatalf("directory mode = %o, want 700", dirInfo.Mode().Perm())
	}

	fileInfo, err := os.Stat(filepath.Join(dir, segmentFileName(1)))
	if err != nil {
		t.Fatalf("stat segment: %v", err)
	}
	if fileInfo.Mode().Perm() != 0o600 {
		t.Fatalf("segment mode = %o, want 600", fileInfo.Mode().Perm())
	}
}

func TestStoreRotationFailureKeepsPriorRecords(t *testing.T) {
	dir := t.TempDir()
	cfg := testStorageConfig(dir)
	cfg.segmentBytes = 256
	opens := 0
	ops := &hookOps{openExclusiveHook: func(path string) (appendFile, error) {
		opens++
		if opens > 1 {
			return nil, errors.New("segment denied")
		}

		return osFileOps{}.openExclusive(path)
	}}
	store := newTestStoreWithOps(t, cfg, ops, func() time.Time { return at(0) })

	if err := store.append(completedRecord(executionID(1), at(1))); err != nil {
		t.Fatalf("append first: %v", err)
	}
	if err := store.append(completedRecord(executionID(2), at(2))); err == nil {
		t.Fatal("append must fail when the next segment cannot be created")
	}
	if store.counters.snapshot().RotationFailed == 0 || !store.counters.historyDegraded() {
		t.Fatal("rotation failure must be counted and degrade history")
	}
	if _, ok := store.index.get(executionID(1)); !ok {
		t.Fatal("records written before the rotation failure must remain queryable")
	}
	if _, ok := store.index.get(executionID(2)); ok {
		t.Fatal("a record rejected by rotation must not be indexed")
	}
	if count := storedLineCount(t, dir); count != 1 {
		t.Fatalf("stored lines = %d, want the prior record only", count)
	}
}
