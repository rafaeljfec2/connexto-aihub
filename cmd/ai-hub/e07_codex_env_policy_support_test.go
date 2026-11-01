package main

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

type e07CodexTreeEntry struct {
	Root       string
	Path       string
	Mode       os.FileMode
	Size       int64
	ModifiedAt int64
	Metadata   string
}

type e07CodexTreeRoot struct {
	Name string
	Path string
}

// e07AccountHomeDirectory resolves the account home independently of $HOME. A
// validation process started with an isolated HOME must still observe the real
// global Codex state root, otherwise the before/after comparison would inspect
// no tree and pass without evidence.
func e07AccountHomeDirectory() (string, error) {
	current, err := user.Current()
	if err != nil || current.HomeDir == "" {
		return "", errors.New("account home directory could not be resolved")
	}
	return current.HomeDir, nil
}

func e07CodexTreeRoots() ([]e07CodexTreeRoot, error) {
	accountHome, err := e07AccountHomeDirectory()
	if err != nil {
		return nil, err
	}
	processHome, err := os.UserHomeDir()
	if err != nil {
		return nil, errors.New("global Codex state roots could not be resolved")
	}
	return e07CodexTreeRootsFor(accountHome, processHome), nil
}

// e07CodexTreeRootsFor lists every Codex-managed state root. The account home is
// always present and is never replaced by an isolated process home.
func e07CodexTreeRootsFor(accountHome, processHome string) []e07CodexTreeRoot {
	roots := []e07CodexTreeRoot{
		{Name: "default_codex_home", Path: filepath.Join(accountHome, ".codex")},
		{Name: "process_home_codex_home", Path: filepath.Join(processHome, ".codex")},
	}
	configured := []struct{ name, variable string }{
		{name: "codex_home", variable: "CODEX_HOME"},
		{name: "codex_sqlite_home", variable: "CODEX_SQLITE_HOME"},
	}
	for _, root := range configured {
		if path := os.Getenv(root.variable); path != "" {
			roots = append(roots, e07CodexTreeRoot{Name: root.name, Path: path})
		}
	}
	return roots
}

func e07SnapshotCodexTree() ([]e07CodexTreeEntry, error) {
	roots, err := e07CodexTreeRoots()
	if err != nil {
		return nil, err
	}
	return e07SnapshotCodexTreeAt(roots)
}

func e07SnapshotCodexTreeAt(roots []e07CodexTreeRoot) ([]e07CodexTreeEntry, error) {
	entries := make([]e07CodexTreeEntry, 0, 32)
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
			if len(entries) >= e07IntegrityMaxEntries {
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
			metadata, err := e07CodexEntryMetadata(path, absolute, info)
			if err != nil {
				return err
			}
			entries = append(entries, e07CodexTreeEntry{
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
func e07CodexEntryMetadata(path, root string, info os.FileInfo) (string, error) {
	metadata := e07CodexStableMetadata(info)
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

func e07CodexStableMetadata(info os.FileInfo) string {
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
func e07CodexTreeLinkCount(entries []e07CodexTreeEntry) int {
	links := 0
	for _, entry := range entries {
		if entry.Mode&os.ModeSymlink != 0 {
			links++
		}
	}
	return links
}

func e07CodexTreeUnchanged(before, after []e07CodexTreeEntry) bool {
	return reflect.DeepEqual(before, after)
}

func e07RequireStableCodexTree() ([]e07CodexTreeEntry, error) {
	active, err := e07CodexProcessActive()
	if err != nil {
		return nil, err
	}
	if active {
		return nil, errors.New("active Codex process prevents global state validation")
	}
	before, err := e07SnapshotCodexTree()
	if err != nil {
		return nil, err
	}
	time.Sleep(250 * time.Millisecond)
	after, err := e07SnapshotCodexTree()
	if err != nil {
		return nil, err
	}
	if !e07CodexTreeUnchanged(before, after) {
		return nil, errors.New("global Codex tree changed before the isolated validation launch")
	}
	return after, nil
}
