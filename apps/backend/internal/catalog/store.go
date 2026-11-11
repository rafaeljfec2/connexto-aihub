package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	snapshotFileName = "catalog.json"
	tempFileName     = "catalog.json.tmp"
	maxSnapshotBytes = 32 << 20
)

// LoadResult describes how a snapshot load degraded.
type LoadResult struct {
	QuarantinedPath string
	WritesDisabled  bool
}

// SaveResult reports durability when a write completed but its directory sync
// failed.
type SaveResult struct {
	DurabilityUncertain bool
}

// Store persists the catalog as a versioned JSON snapshot. A corrupt or
// unsupported file is quarantined before the store starts empty; if quarantine
// fails, writes are disabled so recoverable data is never overwritten.
type Store struct {
	dir    string
	path   string
	logger *slog.Logger
	ops    fileOps

	mu             sync.Mutex
	writesDisabled bool
}

// NewStore builds a snapshot store rooted at dir. The directory is created on
// the first write; construction never touches the filesystem.
func NewStore(dir string, logger *slog.Logger) *Store {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	return &Store{
		dir:    dir,
		path:   filepath.Join(dir, snapshotFileName),
		logger: logger,
		ops:    osOps{},
	}
}

// Load reads the snapshot. It never returns an error: every failure degrades to
// an empty snapshot with a diagnostic, so startup cannot fail because of the
// catalog.
func (s *Store) Load(ctx context.Context) (Snapshot, LoadResult) {
	empty := newEmptySnapshot()

	file, err := os.Open(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return empty, LoadResult{}
	}

	if err != nil {
		s.disableWrites()
		s.logger.Warn("catalog snapshot read failed; writes disabled for this session", "error", err)

		return empty, LoadResult{WritesDisabled: true}
	}

	defer func() { _ = file.Close() }()

	data, err := io.ReadAll(io.LimitReader(file, maxSnapshotBytes+1))
	if err != nil {
		s.disableWrites()
		s.logger.Warn("catalog snapshot read failed; writes disabled for this session", "error", err)

		return empty, LoadResult{WritesDisabled: true}
	}

	if len(data) > maxSnapshotBytes {
		return s.quarantineAndStartEmpty(snapshotFileName+": snapshot exceeds 32 MiB", "catalog.corrupt")
	}

	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return s.quarantineAndStartEmpty("catalog snapshot is not valid JSON", "catalog.corrupt")
	}

	if snapshot.Version != SnapshotVersion {
		if snapshot.Version > SnapshotVersion {
			prefix := fmt.Sprintf("catalog.unsupported-v%d", snapshot.Version)

			return s.quarantineAndStartEmpty(
				fmt.Sprintf("catalog snapshot version %d is newer than %d", snapshot.Version, SnapshotVersion),
				prefix,
			)
		}

		return s.quarantineAndStartEmpty("catalog snapshot version is missing or invalid", "catalog.corrupt")
	}

	if err := snapshot.validate(); err != nil {
		return s.quarantineAndStartEmpty("catalog snapshot failed validation", "catalog.corrupt")
	}

	for providerID, provider := range snapshot.Providers {
		models := make([]ModelEntry, len(provider.Models))
		copy(models, provider.Models)

		for index := range models {
			models[index].ProviderID = providerID
		}

		provider.Models = models
		snapshot.Providers[providerID] = provider
	}

	return snapshot, LoadResult{}
}

// Save atomically replaces the snapshot. The target is only replaced after the
// temp file is written, synced and closed, so a failure leaves the previous
// valid snapshot byte-identical.
func (s *Store) Save(ctx context.Context, snapshot Snapshot) (SaveResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.writesDisabled {
		return SaveResult{}, ErrWritesDisabled
	}

	if err := ctx.Err(); err != nil {
		return SaveResult{}, err
	}

	if snapshot.SavedAt.IsZero() {
		snapshot.SavedAt = time.Now().UTC()
	}

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return SaveResult{}, fmt.Errorf("marshal catalog snapshot: %w", err)
	}

	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return SaveResult{}, fmt.Errorf("create catalog directory: %w", err)
	}

	if err := os.Chmod(s.dir, 0o700); err != nil {
		s.logger.Warn("catalog data directory could not be restricted to 0700", "error", err)
	}

	tempPath := filepath.Join(s.dir, tempFileName)

	temp, err := s.ops.openTemp(tempPath)
	if err != nil {
		return SaveResult{}, fmt.Errorf("open catalog temp file: %w", err)
	}

	committed := false

	defer func() {
		if !committed {
			_ = s.ops.remove(tempPath)
		}
	}()

	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()

		return SaveResult{}, fmt.Errorf("write catalog temp file: %w", err)
	}

	if err := temp.Sync(); err != nil {
		_ = temp.Close()

		return SaveResult{}, fmt.Errorf("sync catalog temp file: %w", err)
	}

	if err := temp.Close(); err != nil {
		return SaveResult{}, fmt.Errorf("close catalog temp file: %w", err)
	}

	if err := ctx.Err(); err != nil {
		return SaveResult{}, err
	}

	if err := s.ops.rename(tempPath, s.path); err != nil {
		return SaveResult{}, fmt.Errorf("replace catalog snapshot: %w", err)
	}

	committed = true

	if err := s.ops.syncDir(s.dir); err != nil {
		s.logger.Warn("catalog snapshot durability uncertain; directory sync failed", "error", err)

		return SaveResult{DurabilityUncertain: true}, nil
	}

	return SaveResult{}, nil
}

func (s *Store) quarantineAndStartEmpty(reason, prefix string) (Snapshot, LoadResult) {
	target, err := s.quarantine(prefix)
	if err != nil {
		s.disableWrites()
		s.logger.Warn("catalog snapshot quarantine failed; writes disabled for this session",
			"reason", reason, "error", err)

		return newEmptySnapshot(), LoadResult{WritesDisabled: true}
	}

	s.logger.Warn("catalog snapshot quarantined", "reason", reason, "path", target)

	return newEmptySnapshot(), LoadResult{QuarantinedPath: target}
}

func (s *Store) disableWrites() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.writesDisabled = true
}

func (s *Store) quarantine(prefix string) (string, error) {
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return "", err
	}

	stamp := time.Now().UTC().Format("20060102T150405.000000000Z")

	for attempt := 0; attempt < 100; attempt++ {
		suffix := ""
		if attempt > 0 {
			suffix = fmt.Sprintf("-%d", attempt)
		}

		target := filepath.Join(s.dir, fmt.Sprintf("%s-%s%s.json", prefix, stamp, suffix))

		_, err := os.Stat(target)
		if err == nil {
			continue
		}

		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}

		if err := s.ops.rename(s.path, target); err != nil {
			return "", err
		}

		return target, nil
	}

	return "", errors.New("could not allocate a unique quarantine name")
}

func newEmptySnapshot() Snapshot {
	return Snapshot{
		Version:   SnapshotVersion,
		Providers: make(map[string]ProviderSnapshot),
	}
}

type tempFile interface {
	io.Writer
	Sync() error
	Close() error
}

type fileOps interface {
	openTemp(path string) (tempFile, error)
	rename(oldPath, newPath string) error
	remove(path string) error
	syncDir(dir string) error
}

type osOps struct{}

func (osOps) openTemp(path string) (tempFile, error) {
	return os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
}

func (osOps) rename(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}

func (osOps) remove(path string) error {
	return os.Remove(path)
}

func (osOps) syncDir(dir string) error {
	directory, err := os.Open(dir)
	if err != nil {
		return err
	}

	defer func() { _ = directory.Close() }()

	return directory.Sync()
}
