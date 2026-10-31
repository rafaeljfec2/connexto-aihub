package records

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"time"
)

// trim applies age, count, and storage limits to the retained history.
func (s *store) trim(now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.trimLocked(now, 0, 0)
}

func (s *store) trimLocked(now time.Time, pendingBytes int64, pendingRecords int) error {
	if s.closed || s.writesDisabled {
		return nil
	}

	records := s.index.all()
	remove := s.selectRemovalsLocked(records, now,
		s.cfg.maxCount-pendingRecords,
		s.cfg.maxStorageBytes-pendingBytes,
	)
	if len(remove) == 0 {
		return nil
	}

	if s.active != nil {
		if err := s.closeActiveLocked(); err != nil {
			s.counters.record(failureRetention)

			return err
		}
	}

	sequences, err := s.segmentSequencesLocked()
	if err != nil {
		s.counters.record(failureRetention)

		return err
	}

	for _, sequence := range sequences {
		dropped, changed, compactErr := s.compactSegmentLocked(sequence, remove)
		if compactErr != nil {
			s.counters.record(failureRetention)

			return compactErr
		}
		if !changed {
			continue
		}
		for _, id := range dropped {
			s.index.remove(id)
		}
	}

	return nil
}

// selectRemovalsLocked chooses expired records first, then the oldest records
// needed to satisfy the count and storage caps.
func (s *store) selectRemovalsLocked(records []Record, now time.Time, maxCount int, maxStorageBytes int64) map[string]struct{} {
	if maxCount < 0 {
		maxCount = 0
	}
	if maxStorageBytes < 0 {
		maxStorageBytes = 0
	}

	sort.Slice(records, func(left, right int) bool { return recordAscending(records[left], records[right]) })

	remove := make(map[string]struct{}, len(records))
	cutoff := now.Add(-s.cfg.maxAge)
	for _, record := range records {
		if record.CompletedAt.Before(cutoff) {
			remove[record.ExecutionID] = struct{}{}
		}
	}

	kept := len(records) - len(remove)
	for _, record := range records {
		if kept <= maxCount {
			break
		}
		if _, removed := remove[record.ExecutionID]; removed {
			continue
		}
		remove[record.ExecutionID] = struct{}{}
		kept--
	}

	sizes := make(map[string]int64, len(records))
	total := int64(0)
	for _, record := range records {
		if _, removed := remove[record.ExecutionID]; removed {
			continue
		}
		size := int64(0)
		if line, err := encodeRecord(record); err == nil {
			size = int64(len(line))
		}
		sizes[record.ExecutionID] = size
		total += size
	}
	for _, record := range records {
		if total <= maxStorageBytes {
			break
		}
		if _, removed := remove[record.ExecutionID]; removed {
			continue
		}
		remove[record.ExecutionID] = struct{}{}
		total -= sizes[record.ExecutionID]
	}

	return remove
}

// compactSegmentLocked rewrites one segment without the removed rows through a
// same-directory temporary file and an atomic rename.
func (s *store) compactSegmentLocked(sequence int64, remove map[string]struct{}) ([]string, bool, error) {
	path := filepath.Join(s.cfg.dir, segmentFileName(sequence))

	file, err := s.ops.openRead(path)
	if err != nil {
		return nil, false, err
	}

	reader := bufio.NewReaderSize(file, maxRecordLineBytes+64)
	kept := make([][]byte, 0, 16)
	dropped := make([]string, 0)
	seen := make(map[string]struct{}, 16)
	lines := 0

	for {
		line, readErr := reader.ReadBytes('\n')
		if len(line) > 0 {
			if line[len(line)-1] != '\n' {
				_ = file.Close()

				return nil, false, fmt.Errorf("segment %s has an incomplete line", segmentFileName(sequence))
			}

			record, decodeErr := decodeRecord(line[:len(line)-1])
			if decodeErr != nil {
				_ = file.Close()

				return nil, false, decodeErr
			}

			lines++
			_, removed := remove[record.ExecutionID]
			_, duplicate := seen[record.ExecutionID]
			switch {
			case removed:
				dropped = append(dropped, record.ExecutionID)
			case duplicate:
			default:
				seen[record.ExecutionID] = struct{}{}
				kept = append(kept, append([]byte(nil), line...))
			}
		}
		if readErr != nil {
			if !errors.Is(readErr, io.EOF) {
				_ = file.Close()

				return nil, false, readErr
			}

			break
		}
	}
	if err := file.Close(); err != nil {
		return nil, false, err
	}
	if len(kept) == lines {
		return nil, false, nil
	}

	tempPath := filepath.Join(s.cfg.dir, fmt.Sprintf(".%s.tmp-%d", segmentFileName(sequence), s.tempSequence))
	s.tempSequence++

	temp, err := s.ops.openTemp(tempPath)
	if err != nil {
		return nil, false, err
	}
	for _, line := range kept {
		if _, err := writeAll(temp, line); err != nil {
			_ = temp.Close()
			_ = s.ops.remove(tempPath)

			return nil, false, err
		}
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		_ = s.ops.remove(tempPath)
		s.counters.record(failureSync)

		return nil, false, err
	}
	if err := temp.Close(); err != nil {
		s.counters.record(failureClose)
		s.disableWritesLocked("close_failed")
		_ = s.ops.remove(tempPath)

		return nil, false, err
	}
	if err := s.ops.rename(tempPath, path); err != nil {
		_ = s.ops.remove(tempPath)

		return nil, false, err
	}
	if err := s.ops.syncDir(s.cfg.dir); err != nil {
		s.counters.record(failureSync)
	}

	return dropped, true, nil
}

func recordAscending(left, right Record) bool {
	if !left.CompletedAt.Equal(right.CompletedAt) {
		return left.CompletedAt.Before(right.CompletedAt)
	}

	return left.ExecutionID < right.ExecutionID
}
