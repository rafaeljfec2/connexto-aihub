package records

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

func (s *store) openActiveLocked() error {
	for attempt := 0; attempt < 64; attempt++ {
		s.nextSequence++
		path := filepath.Join(s.cfg.dir, segmentFileName(s.nextSequence))

		file, err := s.ops.openExclusive(path)
		if err == nil {
			s.active = file
			s.activeSequence = s.nextSequence
			s.activeSize = 0
			if syncErr := s.ops.syncDir(s.cfg.dir); syncErr != nil {
				s.counters.record(failureSync)
			}

			return nil
		}
		if !errors.Is(err, os.ErrExist) {
			s.counters.record(failureRotation)
			s.logger.Warn("execution record segment open failed", "class", "segment_open_failed")

			return err
		}
	}

	s.counters.record(failureRotation)

	return errors.New("record segment sequence exhausted")
}

// crossesCapsLocked reports whether accepting one more line would exceed the
// count or the physical segment-size cap.
func (s *store) crossesCapsLocked(lineBytes int64) bool {
	if s.index.len()+1 > s.cfg.maxCount {
		return true
	}

	stored, err := s.storedBytesLocked()
	if err != nil {
		return true
	}

	return stored+lineBytes > s.cfg.maxStorageBytes
}

// confirmCapsLocked verifies that the pre-append trim freed enough room. It
// never deletes a newer record while an older one remains eligible.
func (s *store) confirmCapsLocked(lineBytes int64) error {
	if s.index.len()+1 > s.cfg.maxCount {
		s.counters.record(failureStorageLimit)
		s.logger.Warn("execution record dropped", "class", "count_cap_reached")

		return errCapExceeded
	}

	stored, err := s.storedBytesLocked()
	if err != nil {
		s.counters.record(failureStorageLimit)
		s.logger.Warn("execution record storage measurement failed", "class", "storage_measure_failed")

		return err
	}
	if stored+lineBytes > s.cfg.maxStorageBytes {
		s.counters.record(failureStorageLimit)
		s.logger.Warn("execution record dropped", "class", "storage_cap_reached")

		return errCapExceeded
	}

	return nil
}

// storedBytesLocked sums the physical size of the managed JSONL segments.
func (s *store) storedBytesLocked() (int64, error) {
	entries, err := s.ops.readDir(s.cfg.dir)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	total := int64(0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if _, ok := parseSegmentName(entry.Name()); !ok {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			return 0, err
		}
		total += info.Size()
	}

	return total, nil
}

func (s *store) rotateLocked() error {
	if err := s.closeActiveLocked(); err != nil {
		return err
	}

	return s.openActiveLocked()
}

func (s *store) rollbackLocked(offset int64) {
	if s.active == nil {
		return
	}

	if err := s.active.Truncate(offset); err != nil {
		s.counters.record(failureRollback)
		s.disableWritesLocked("rollback_failed")

		return
	}

	s.activeSize = offset
}

func (s *store) closeActiveLocked() error {
	if s.active == nil {
		return nil
	}

	file := s.active
	s.active = nil

	var errs []error
	if err := file.Sync(); err != nil {
		s.counters.record(failureSync)
		errs = append(errs, err)
	}
	if err := file.Close(); err != nil {
		s.counters.record(failureClose)
		s.disableWritesLocked("close_failed")
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func (s *store) disableWritesLocked(class string) {
	s.writesDisabled = true
	s.logger.Warn("execution record writes disabled", "class", class)
}

func writeAll(file appendFile, data []byte) (int, error) {
	total := 0

	for total < len(data) {
		written, err := file.Write(data[total:])
		total += written
		if err != nil {
			return total, err
		}
		if written == 0 {
			return total, io.ErrShortWrite
		}
	}

	return total, nil
}
