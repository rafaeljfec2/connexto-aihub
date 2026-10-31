package records

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	segmentPrefix = "executions-"
	segmentSuffix = ".jsonl"
)

var (
	errWritesDisabled = errors.New("record writes are disabled")
	errCapExceeded    = errors.New("record exceeds the retained storage cap")
)

// storageConfig carries the validated retention and segment bounds.
type storageConfig struct {
	dir               string
	maxAge            time.Duration
	maxCount          int
	maxStorageBytes   int64
	segmentBytes      int64
	retentionInterval time.Duration
}

// store owns every mutation of the JSONL segments and the query index.
type store struct {
	cfg      storageConfig
	logger   *slog.Logger
	ops      fileOps
	now      func() time.Time
	index    *index
	counters *counters

	mu             sync.Mutex
	active         appendFile
	activeSequence int64
	activeSize     int64
	nextSequence   int64
	tempSequence   int
	writesDisabled bool
	closed         bool
}

func newStore(cfg storageConfig, logger *slog.Logger, ops fileOps, now func() time.Time, idx *index, counters *counters) *store {
	if logger == nil {
		logger = slog.Default()
	}
	if now == nil {
		now = time.Now
	}

	return &store{cfg: cfg, logger: logger, ops: ops, now: now, index: idx, counters: counters}
}

func segmentFileName(sequence int64) string {
	return fmt.Sprintf("%s%020d%s", segmentPrefix, sequence, segmentSuffix)
}

func parseSegmentName(name string) (int64, bool) {
	if !strings.HasPrefix(name, segmentPrefix) || !strings.HasSuffix(name, segmentSuffix) {
		return 0, false
	}

	digits := strings.TrimSuffix(strings.TrimPrefix(name, segmentPrefix), segmentSuffix)
	if len(digits) != 20 {
		return 0, false
	}

	sequence, err := strconv.ParseInt(digits, 10, 64)
	if err != nil || sequence < 0 {
		return 0, false
	}

	return sequence, true
}

func (s *store) start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ops.mkdirAll(s.cfg.dir, directoryMode); err != nil {
		return fmt.Errorf("create records directory: %w", err)
	}
	if err := s.ops.chmod(s.cfg.dir, directoryMode); err != nil {
		return fmt.Errorf("restrict records directory: %w", err)
	}

	sequences, err := s.segmentSequencesLocked()
	if err != nil {
		s.counters.record(failureRecovery)
		s.logger.Warn("execution record recovery failed", "class", "read_directory")
		s.writesDisabled = true

		return nil
	}

	for _, sequence := range sequences {
		s.recoverSegmentLocked(sequence)
		if sequence > s.nextSequence {
			s.nextSequence = sequence
		}
	}

	if err := s.trimLocked(s.now(), 0, 0); err != nil {
		s.logger.Warn("execution record retention failed", "class", "startup_trim")
	}

	return nil
}

func (s *store) append(record Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed || s.writesDisabled {
		return errWritesDisabled
	}

	line, err := encodeRecord(record)
	if err != nil {
		s.counters.record(failureInvalidRecord)

		return err
	}

	if s.crossesCapsLocked(int64(len(line))) {
		if err := s.trimLocked(s.now(), int64(len(line)), 1); err != nil {
			return err
		}
		if err := s.confirmCapsLocked(int64(len(line))); err != nil {
			return err
		}
	}

	if s.active == nil {
		if err := s.openActiveLocked(); err != nil {
			return err
		}
	}
	if s.activeSize > 0 && s.activeSize+int64(len(line)) > s.cfg.segmentBytes {
		if err := s.rotateLocked(); err != nil {
			return err
		}
	}

	before := s.activeSize
	written, err := writeAll(s.active, line)
	if err != nil {
		if written > 0 {
			s.rollbackLocked(before)
		}
		s.counters.record(failureAppend)
		s.logger.Warn("execution record append failed", "class", "append_failed")

		return err
	}

	s.activeSize += int64(written)
	if !s.index.add(record, s.activeSequence) {
		s.counters.record(failureDuplicateRecord)
	}

	return nil
}

func (s *store) close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}
	s.closed = true

	return s.closeActiveLocked()
}
