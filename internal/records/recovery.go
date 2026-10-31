package records

import (
	"bufio"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// segmentSequencesLocked lists numeric segments oldest first and removes known
// uncommitted compaction temporary files.
func (s *store) segmentSequencesLocked() ([]int64, error) {
	entries, err := s.ops.readDir(s.cfg.dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	sequences := make([]int64, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if isTemporarySegment(name) {
			_ = s.ops.remove(filepath.Join(s.cfg.dir, name))

			continue
		}
		if entry.IsDir() {
			continue
		}

		sequence, ok := parseSegmentName(name)
		if ok {
			sequences = append(sequences, sequence)
		}
	}

	sort.Slice(sequences, func(left, right int) bool { return sequences[left] < sequences[right] })

	return sequences, nil
}

func isTemporarySegment(name string) bool {
	return strings.HasPrefix(name, "."+segmentPrefix) && strings.Contains(name, segmentSuffix+".tmp-")
}

// recoverSegmentLocked rebuilds the index from one segment and isolates the
// source from appends at the first invalid or incomplete line.
func (s *store) recoverSegmentLocked(sequence int64) {
	path := filepath.Join(s.cfg.dir, segmentFileName(sequence))

	file, err := s.ops.openRead(path)
	if err != nil {
		s.counters.record(failureRecovery)
		s.disableWritesLocked("segment_read_failed")
		s.logger.Warn("execution record recovery failed",
			"class", "segment_read_failed",
			"segment", segmentFileName(sequence),
		)

		return
	}
	defer func() { _ = file.Close() }()

	reader := bufio.NewReaderSize(file, maxRecordLineBytes+64)
	offset := int64(0)
	lastValid := int64(0)

	for {
		line, readErr := reader.ReadBytes('\n')
		if len(line) > 0 {
			if line[len(line)-1] != '\n' {
				s.corruptSegmentLocked(path, sequence, offset, lastValid, "incomplete_line")

				return
			}

			record, decodeErr := decodeRecord(line[:len(line)-1])
			if decodeErr != nil {
				s.corruptSegmentLocked(path, sequence, offset, lastValid, "invalid_line")

				return
			}
			if !s.index.add(record, sequence) {
				s.counters.record(failureDuplicateRecord)
			}

			offset += int64(len(line))
			lastValid = offset
		}
		if readErr != nil {
			if !errors.Is(readErr, io.EOF) {
				s.counters.record(failureRecovery)
				s.disableWritesLocked("segment_read_failed")
				s.logger.Warn("execution record recovery failed",
					"class", "segment_read_failed",
					"segment", segmentFileName(sequence),
				)
			}

			return
		}
	}
}

func (s *store) corruptSegmentLocked(path string, sequence, offset, lastValid int64, class string) {
	s.counters.record(failureRecovery)
	s.logger.Warn("execution record segment isolated",
		"class", class,
		"segment", filepath.Base(path),
		"offset", offset,
	)

	if err := s.truncateSegmentLocked(path, lastValid); err != nil {
		s.disableWritesLocked("truncate_failed")
	}
}

// truncateSegmentLocked removes the unvalidated suffix without copying it.
func (s *store) truncateSegmentLocked(path string, size int64) error {
	file, err := s.ops.openAppend(path)
	if err != nil {
		return err
	}

	if err := file.Truncate(size); err != nil {
		_ = file.Close()

		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		s.counters.record(failureSync)

		return err
	}
	if err := file.Close(); err != nil {
		s.counters.record(failureClose)

		return err
	}
	if err := s.ops.syncDir(s.cfg.dir); err != nil {
		s.counters.record(failureSync)
	}

	return nil
}
