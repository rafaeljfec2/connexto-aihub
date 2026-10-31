package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"aihub/internal/providers"
)

func storeTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type failingTemp struct {
	base     tempFile
	writeErr error
	syncErr  error
	closeErr error
}

func (t *failingTemp) Write(data []byte) (int, error) {
	if t.writeErr != nil {
		return 0, t.writeErr
	}

	return t.base.Write(data)
}

func (t *failingTemp) Sync() error {
	if t.syncErr != nil {
		return t.syncErr
	}

	return t.base.Sync()
}

func (t *failingTemp) Close() error {
	err := t.base.Close()
	if t.closeErr != nil {
		return t.closeErr
	}

	return err
}

type failingOps struct {
	base       fileOps
	openErr    error
	writeErr   error
	syncErr    error
	closeErr   error
	renameErr  error
	syncDirErr error
}

func (o *failingOps) openTemp(path string) (tempFile, error) {
	if o.openErr != nil {
		return nil, o.openErr
	}

	base, err := o.base.openTemp(path)
	if err != nil {
		return nil, err
	}

	return &failingTemp{base: base, writeErr: o.writeErr, syncErr: o.syncErr, closeErr: o.closeErr}, nil
}

func (o *failingOps) rename(oldPath, newPath string) error {
	if o.renameErr != nil {
		return o.renameErr
	}

	return o.base.rename(oldPath, newPath)
}

func (o *failingOps) remove(path string) error {
	return o.base.remove(path)
}

func (o *failingOps) syncDir(dir string) error {
	if o.syncDirErr != nil {
		return o.syncDirErr
	}

	return o.base.syncDir(dir)
}

func storeTestStore(dir string, ops fileOps) *Store {
	return &Store{
		dir:    dir,
		path:   filepath.Join(dir, snapshotFileName),
		logger: storeTestLogger(),
		ops:    ops,
	}
}

func storeTestSnapshot(modelID string) Snapshot {
	now := time.Date(2026, 10, 30, 18, 0, 0, 0, time.UTC)
	entry := NormalizeEntry(providers.OpenCodeGoID, providers.ModelDescriptor{
		ID:   modelID,
		Name: "Model " + modelID,
	}, now)

	return Snapshot{
		Version: SnapshotVersion,
		SavedAt: now,
		Providers: map[string]ProviderSnapshot{
			providers.OpenCodeGoID: {
				Sync: ProviderSyncState{
					LastSuccessAt: &now,
					Outcome:       SyncOutcomeSuccess,
					ErrorClass:    SyncErrorNone,
				},
				Models: []ModelEntry{entry},
			},
		},
	}
}

func TestStoreRoundTrip(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "data")
	store := storeTestStore(dir, osOps{})
	ctx := context.Background()

	if _, err := store.Save(ctx, storeTestSnapshot("model-a")); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	fileInfo, err := os.Stat(filepath.Join(dir, snapshotFileName))
	if err != nil {
		t.Fatalf("stat snapshot: %v", err)
	}

	if got := fileInfo.Mode().Perm(); got != 0o600 {
		t.Fatalf("snapshot mode = %o, want 600", got)
	}

	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat data dir: %v", err)
	}

	if got := dirInfo.Mode().Perm(); got != 0o700 {
		t.Fatalf("data dir mode = %o, want 700", got)
	}

	snapshot, result := store.Load(ctx)
	if result.QuarantinedPath != "" || result.WritesDisabled {
		t.Fatalf("Load() result = %+v, want clean load", result)
	}

	if snapshot.Version != SnapshotVersion || snapshot.SavedAt.IsZero() {
		t.Fatalf("snapshot header = version %d saved %v", snapshot.Version, snapshot.SavedAt)
	}

	provider, ok := snapshot.Providers[providers.OpenCodeGoID]
	if !ok {
		t.Fatal("provider missing after load")
	}

	if len(provider.Models) != 1 {
		t.Fatalf("model count = %d, want 1", len(provider.Models))
	}

	entry := provider.Models[0]
	if entry.ProviderID != providers.OpenCodeGoID {
		t.Fatalf("provider id = %q, want %q", entry.ProviderID, providers.OpenCodeGoID)
	}

	if entry.ModelID != "model-a" {
		t.Fatalf("model id = %q, want model-a", entry.ModelID)
	}

	if len(entry.Capabilities) != len(capabilityVocabulary) {
		t.Fatalf("capability count = %d, want %d", len(entry.Capabilities), len(capabilityVocabulary))
	}

	if provider.Sync.Outcome != SyncOutcomeSuccess {
		t.Fatalf("sync outcome = %q, want success", provider.Sync.Outcome)
	}
}

func TestStoreLoadMissingFile(t *testing.T) {
	store := storeTestStore(t.TempDir(), osOps{})

	snapshot, result := store.Load(context.Background())
	if result.QuarantinedPath != "" || result.WritesDisabled {
		t.Fatalf("Load() result = %+v, want clean empty load", result)
	}

	if snapshot.Version != SnapshotVersion || len(snapshot.Providers) != 0 {
		t.Fatalf("snapshot = %+v, want empty versioned snapshot", snapshot)
	}
}

func TestStoreQuarantinesCorruptSnapshot(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, snapshotFileName)

	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatalf("write corrupt snapshot: %v", err)
	}

	store := storeTestStore(dir, osOps{})
	snapshot, result := store.Load(context.Background())

	if result.QuarantinedPath == "" {
		t.Fatal("QuarantinedPath = empty, want quarantine")
	}

	if result.WritesDisabled {
		t.Fatal("WritesDisabled = true, want false after successful quarantine")
	}

	if len(snapshot.Providers) != 0 {
		t.Fatalf("snapshot providers = %d, want 0", len(snapshot.Providers))
	}

	quarantined, err := os.ReadFile(result.QuarantinedPath)
	if err != nil {
		t.Fatalf("read quarantine: %v", err)
	}

	if string(quarantined) != "{" {
		t.Fatalf("quarantine content = %q, want %q", quarantined, "{")
	}

	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("catalog path still exists after quarantine: %v", err)
	}

	if _, err := store.Save(context.Background(), storeTestSnapshot("model-a")); err != nil {
		t.Fatalf("Save() after quarantine error = %v", err)
	}

	after, err := os.ReadFile(result.QuarantinedPath)
	if err != nil || string(after) != "{" {
		t.Fatalf("quarantine changed after later save: %q, err=%v", after, err)
	}
}

func TestStoreQuarantineFailureDisablesWrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, snapshotFileName)

	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatalf("write corrupt snapshot: %v", err)
	}

	store := storeTestStore(dir, &failingOps{base: osOps{}, renameErr: errors.New("rename denied")})
	_, result := store.Load(context.Background())

	if !result.WritesDisabled {
		t.Fatal("WritesDisabled = false, want true")
	}

	if result.QuarantinedPath != "" {
		t.Fatalf("QuarantinedPath = %q, want empty", result.QuarantinedPath)
	}

	data, err := os.ReadFile(path)
	if err != nil || string(data) != "{" {
		t.Fatalf("corrupt file changed: %q, err=%v", data, err)
	}

	if _, err := store.Save(context.Background(), storeTestSnapshot("model-a")); !errors.Is(err, ErrWritesDisabled) {
		t.Fatalf("Save() error = %v, want ErrWritesDisabled", err)
	}

	data, err = os.ReadFile(path)
	if err != nil || string(data) != "{" {
		t.Fatalf("corrupt file changed after Save attempt: %q, err=%v", data, err)
	}
}

func TestStoreQuarantinesUnsupportedVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, snapshotFileName)

	if err := os.WriteFile(path, []byte(`{"version":2,"providers":{}}`), 0o600); err != nil {
		t.Fatalf("write snapshot: %v", err)
	}

	store := storeTestStore(dir, osOps{})
	_, result := store.Load(context.Background())

	if result.QuarantinedPath == "" || !strings.Contains(result.QuarantinedPath, "unsupported-v2") {
		t.Fatalf("QuarantinedPath = %q, want unsupported-v2 name", result.QuarantinedPath)
	}

	if result.WritesDisabled {
		t.Fatal("WritesDisabled = true, want false")
	}
}

func TestStoreReadErrorDisablesWrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, snapshotFileName)

	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatalf("mkdir snapshot path: %v", err)
	}

	store := storeTestStore(dir, osOps{})
	_, result := store.Load(context.Background())

	if !result.WritesDisabled {
		t.Fatal("WritesDisabled = false, want true")
	}

	if result.QuarantinedPath != "" {
		t.Fatalf("QuarantinedPath = %q, want empty", result.QuarantinedPath)
	}
}

func TestStoreSaveFailuresPreservePreviousSnapshot(t *testing.T) {
	cases := []struct {
		name string
		ops  *failingOps
	}{
		{name: "open", ops: &failingOps{base: osOps{}, openErr: errors.New("open denied")}},
		{name: "write", ops: &failingOps{base: osOps{}, writeErr: errors.New("write denied")}},
		{name: "sync", ops: &failingOps{base: osOps{}, syncErr: errors.New("sync denied")}},
		{name: "close", ops: &failingOps{base: osOps{}, closeErr: errors.New("close denied")}},
		{name: "rename", ops: &failingOps{base: osOps{}, renameErr: errors.New("rename denied")}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			dir := t.TempDir()
			ctx := context.Background()

			if _, err := storeTestStore(dir, osOps{}).Save(ctx, storeTestSnapshot("model-a")); err != nil {
				t.Fatalf("seed Save() error = %v", err)
			}

			path := filepath.Join(dir, snapshotFileName)
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read before: %v", err)
			}

			if _, err := storeTestStore(dir, testCase.ops).Save(ctx, storeTestSnapshot("model-b")); err == nil {
				t.Fatal("Save() error = nil, want failure")
			}

			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read after: %v", err)
			}

			if string(after) != string(before) {
				t.Fatalf("previous snapshot changed by failed save")
			}

			if _, err := os.Stat(filepath.Join(dir, tempFileName)); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("temp file still exists after failure: %v", err)
			}

			loaded, result := storeTestStore(dir, osOps{}).Load(ctx)
			if result.QuarantinedPath != "" || result.WritesDisabled {
				t.Fatalf("Load() result = %+v, want clean load", result)
			}

			if got := loaded.Providers[providers.OpenCodeGoID].Models[0].ModelID; got != "model-a" {
				t.Fatalf("loaded model = %q, want model-a", got)
			}
		})
	}
}

func TestStoreDirectorySyncFailureIsDurabilityUncertain(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	if _, err := storeTestStore(dir, osOps{}).Save(ctx, storeTestSnapshot("model-a")); err != nil {
		t.Fatalf("seed Save() error = %v", err)
	}

	result, err := storeTestStore(dir, &failingOps{base: osOps{}, syncDirErr: errors.New("dir sync denied")}).
		Save(ctx, storeTestSnapshot("model-b"))
	if err != nil {
		t.Fatalf("Save() error = %v, want nil for durability-uncertain write", err)
	}

	if !result.DurabilityUncertain {
		t.Fatal("DurabilityUncertain = false, want true")
	}

	loaded, loadResult := storeTestStore(dir, osOps{}).Load(ctx)
	if loadResult.QuarantinedPath != "" || loadResult.WritesDisabled {
		t.Fatalf("Load() result = %+v, want clean load", loadResult)
	}

	if got := loaded.Providers[providers.OpenCodeGoID].Models[0].ModelID; got != "model-b" {
		t.Fatalf("loaded model = %q, want model-b", got)
	}
}

func TestStoreRejectsInvalidSnapshots(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Snapshot)
	}{
		{name: "empty model id", mutate: func(s *Snapshot) {
			provider := s.Providers[providers.OpenCodeGoID]
			provider.Models[0].ModelID = ""
			s.Providers[providers.OpenCodeGoID] = provider
		}},
		{name: "duplicate model", mutate: func(s *Snapshot) {
			provider := s.Providers[providers.OpenCodeGoID]
			provider.Models = append(provider.Models, provider.Models[0])
			s.Providers[providers.OpenCodeGoID] = provider
		}},
		{name: "unknown capability", mutate: func(s *Snapshot) {
			provider := s.Providers[providers.OpenCodeGoID]
			entry := provider.Models[0]
			entry.Capabilities["made_up"] = entry.Capabilities["text_generation"]
			provider.Models[0] = entry
			s.Providers[providers.OpenCodeGoID] = provider
		}},
		{name: "missing capability", mutate: func(s *Snapshot) {
			provider := s.Providers[providers.OpenCodeGoID]
			entry := provider.Models[0]
			delete(entry.Capabilities, "text_generation")
			provider.Models[0] = entry
			s.Providers[providers.OpenCodeGoID] = provider
		}},
		{name: "invalid state", mutate: func(s *Snapshot) {
			provider := s.Providers[providers.OpenCodeGoID]
			entry := provider.Models[0]
			claim := entry.Capabilities["text_generation"]
			claim.State = "bogus"
			entry.Capabilities["text_generation"] = claim
			provider.Models[0] = entry
			s.Providers[providers.OpenCodeGoID] = provider
		}},
		{name: "unknown with explicit confidence", mutate: func(s *Snapshot) {
			provider := s.Providers[providers.OpenCodeGoID]
			entry := provider.Models[0]
			claim := entry.Capabilities["text_generation"]
			claim.State = CapabilityUnknown
			claim.Confidence = ConfidenceExplicit
			entry.Capabilities["text_generation"] = claim
			provider.Models[0] = entry
			s.Providers[providers.OpenCodeGoID] = provider
		}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			dir := t.TempDir()
			snapshot := storeTestSnapshot("model-a")
			testCase.mutate(&snapshot)

			data, err := json.Marshal(snapshot)
			if err != nil {
				t.Fatalf("marshal snapshot: %v", err)
			}

			if err := os.WriteFile(filepath.Join(dir, snapshotFileName), data, 0o600); err != nil {
				t.Fatalf("write snapshot: %v", err)
			}

			_, result := storeTestStore(dir, osOps{}).Load(context.Background())
			if result.QuarantinedPath == "" {
				t.Fatal("QuarantinedPath = empty, want quarantine")
			}

			if result.WritesDisabled {
				t.Fatal("WritesDisabled = true, want false")
			}
		})
	}
}

func TestStoreIgnoresOrphanTempFile(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	if _, err := storeTestStore(dir, osOps{}).Save(ctx, storeTestSnapshot("model-a")); err != nil {
		t.Fatalf("seed Save() error = %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, tempFileName), []byte("garbage"), 0o600); err != nil {
		t.Fatalf("write orphan temp: %v", err)
	}

	loaded, result := storeTestStore(dir, osOps{}).Load(ctx)
	if result.QuarantinedPath != "" || result.WritesDisabled {
		t.Fatalf("Load() result = %+v, want clean load", result)
	}

	if got := loaded.Providers[providers.OpenCodeGoID].Models[0].ModelID; got != "model-a" {
		t.Fatalf("loaded model = %q, want model-a", got)
	}
}

func TestStoreSaveHonorsCancelledContext(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	if _, err := storeTestStore(dir, osOps{}).Save(ctx, storeTestSnapshot("model-a")); err != nil {
		t.Fatalf("seed Save() error = %v", err)
	}

	path := filepath.Join(dir, snapshotFileName)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read before: %v", err)
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := storeTestStore(dir, osOps{}).Save(cancelled, storeTestSnapshot("model-b")); err == nil {
		t.Fatal("Save() error = nil, want cancellation")
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read after: %v", err)
	}

	if string(after) != string(before) {
		t.Fatal("cancelled save changed the snapshot")
	}
}

func TestStoreQuarantineSurvivesLaterPersistenceFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, snapshotFileName)

	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatalf("write corrupt snapshot: %v", err)
	}

	store := storeTestStore(dir, osOps{})
	_, result := store.Load(context.Background())

	if result.QuarantinedPath == "" {
		t.Fatal("QuarantinedPath = empty, want quarantine")
	}

	store.ops = &failingOps{base: osOps{}, renameErr: errors.New("rename denied")}

	if _, err := store.Save(context.Background(), storeTestSnapshot("model-a")); err == nil {
		t.Fatal("Save() error = nil, want failure")
	}

	quarantined, err := os.ReadFile(result.QuarantinedPath)
	if err != nil {
		t.Fatalf("read quarantine: %v", err)
	}

	if string(quarantined) != "{" {
		t.Fatalf("quarantine changed after later persistence failure: %q", quarantined)
	}

	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("catalog path exists after no-valid-snapshot failure: %v", err)
	}
}

func TestStoreRestrictsPreexistingDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "data")

	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatalf("mkdir data dir: %v", err)
	}

	if _, err := storeTestStore(dir, osOps{}).Save(context.Background(), storeTestSnapshot("model-a")); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat data dir: %v", err)
	}

	if got := info.Mode().Perm(); got != 0o700 {
		t.Fatalf("data dir mode = %o, want 700", got)
	}
}
