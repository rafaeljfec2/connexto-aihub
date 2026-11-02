package support

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"
)

type E07CodexTreeEntry struct {
	Root       string
	Path       string
	Mode       os.FileMode
	Size       int64
	ModifiedAt int64
	Metadata   string
}

type E07CodexTreeRoot struct {
	Name string
	Path string
}

// e07AccountHomeDirectory resolves the account home independently of $HOME. A
// validation process started with an isolated HOME must still observe the real
// global Codex state root, otherwise the before/after comparison would inspect
// no tree and pass without evidence.
func E07AccountHomeDirectory() (string, error) {
	current, err := user.Current()
	if err != nil || current.HomeDir == "" {
		return "", errors.New("account home directory could not be resolved")
	}
	return current.HomeDir, nil
}

func E07CodexTreeRoots() ([]E07CodexTreeRoot, error) {
	accountHome, err := E07AccountHomeDirectory()
	if err != nil {
		return nil, err
	}
	processHome, err := os.UserHomeDir()
	if err != nil {
		return nil, errors.New("global Codex state roots could not be resolved")
	}
	return E07CodexTreeRootsFor(accountHome, processHome), nil
}

// e07CodexTreeRootsFor lists every Codex-managed state root. The account home is
// always present and is never replaced by an isolated process home.
func E07CodexTreeRootsFor(accountHome, processHome string) []E07CodexTreeRoot {
	roots := []E07CodexTreeRoot{
		{Name: "default_codex_home", Path: filepath.Join(accountHome, ".codex")},
		{Name: "process_home_codex_home", Path: filepath.Join(processHome, ".codex")},
	}
	configured := []struct{ name, variable string }{
		{name: "codex_home", variable: "CODEX_HOME"},
		{name: "codex_sqlite_home", variable: "CODEX_SQLITE_HOME"},
	}
	for _, root := range configured {
		if path := os.Getenv(root.variable); path != "" {
			roots = append(roots, E07CodexTreeRoot{Name: root.name, Path: path})
		}
	}
	return roots
}

func E07SnapshotCodexTree() ([]E07CodexTreeEntry, error) {
	roots, err := E07CodexTreeRoots()
	if err != nil {
		return nil, err
	}
	return E07SnapshotCodexTreeAt(roots)
}

func E07SnapshotCodexTreeAt(roots []E07CodexTreeRoot) ([]E07CodexTreeEntry, error) {
	entries := make([]E07CodexTreeEntry, 0, 32)
	seenRoots := make(map[string]struct{}, len(roots))
	for _, root := range roots {
		absolute, err := filepath.Abs(root.Path)
		if err != nil {
			return nil, errors.New("global Codex state root could not be normalized")
		}
		if _, seen := seenRoots[absolute]; seen {
			continue
		}
		seenRoots[absolute] = struct{}{}
		if _, err := os.Lstat(absolute); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return nil, errors.New("global Codex state root could not be inspected")
		}
		err = filepath.WalkDir(absolute, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return errors.New("global Codex metadata tree could not be traversed")
			}
			if len(entries) >= E07IntegrityMaxEntries {
				return errors.New("global Codex metadata tree exceeded the validation bound")
			}
			info, err := os.Lstat(path)
			if err != nil {
				return errors.New("global Codex metadata could not be inspected")
			}
			relative, err := filepath.Rel(absolute, path)
			if err != nil {
				return errors.New("global Codex metadata entry could not be named")
			}
			metadata, err := E07CodexEntryMetadata(path, absolute, info)
			if err != nil {
				return err
			}
			entries = append(entries, E07CodexTreeEntry{
				Root: root.Name, Path: filepath.ToSlash(relative), Mode: info.Mode(), Size: info.Size(),
				ModifiedAt: info.ModTime().UnixNano(), Metadata: metadata,
			})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Root == entries[j].Root {
			return entries[i].Path < entries[j].Path
		}
		return entries[i].Root < entries[j].Root
	})
	return entries, nil
}

// e07CodexEntryMetadata describes one tree entry. Symbolic links are recorded
// with their target instead of being followed, so a link change is detected
// without reading anything behind it; a symbolic-link state root still fails
// closed because the protected tree would then be outside the observed path.
func E07CodexEntryMetadata(path, root string, info os.FileInfo) (string, error) {
	metadata := E07StableFileMetadata(info)
	if info.Mode()&os.ModeSymlink == 0 {
		return metadata, nil
	}
	if path == root {
		return "", errors.New("global Codex state root is a symbolic link")
	}
	target, err := os.Readlink(path)
	if err != nil {
		return "", errors.New("global Codex metadata link could not be inspected")
	}
	return metadata + "|target=" + target, nil
}

func E07StableFileMetadata(info os.FileInfo) string {
	value := reflect.ValueOf(info.Sys())
	if !value.IsValid() {
		return ""
	}
	if value.Kind() == reflect.Pointer {
		value = value.Elem()
	}
	if !value.IsValid() || value.Kind() != reflect.Struct {
		return ""
	}

	typeOfValue := value.Type()
	fields := make([]string, 0, value.NumField())
	for index := 0; index < value.NumField(); index++ {
		name := typeOfValue.Field(index).Name
		if strings.Contains(strings.ToLower(name), "atim") {
			continue
		}
		field := value.Field(index)
		if field.CanInterface() {
			fields = append(fields, fmt.Sprintf("%s=%v", name, field.Interface()))
		}
	}
	sort.Strings(fields)
	return strings.Join(fields, ";")
}

// e07CodexTreeLinkCount reports how many nested symbolic links the snapshot
// covers by target only. State written through such a link is outside the
// comparison, so the count is reported together with the integrity outcome.
func E07CodexTreeLinkCount(entries []E07CodexTreeEntry) int {
	links := 0
	for _, entry := range entries {
		if entry.Mode&os.ModeSymlink != 0 {
			links++
		}
	}
	return links
}

func E07CodexTreeUnchanged(before, after []E07CodexTreeEntry) bool {
	return reflect.DeepEqual(before, after)
}

func E07RequireStableCodexTree() ([]E07CodexTreeEntry, error) {
	active, err := E07CodexProcessActive()
	if err != nil {
		return nil, err
	}
	if active {
		return nil, errors.New("active Codex process prevents global state validation")
	}
	before, err := E07SnapshotCodexTree()
	if err != nil {
		return nil, err
	}
	time.Sleep(250 * time.Millisecond)
	after, err := E07SnapshotCodexTree()
	if err != nil {
		return nil, err
	}
	if !E07CodexTreeUnchanged(before, after) {
		return nil, errors.New("global Codex tree changed before the isolated validation launch")
	}
	return after, nil
}
