package records

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"aihub/internal/execution"
)

const (
	queueCapacity       = 256
	defaultFlushTimeout = 2 * time.Second
	defaultRetention    = time.Hour
)

// ErrUnavailable means no usable query index could be constructed.
var ErrUnavailable = errors.New("execution records are unavailable")

// Options configures the records service.
type Options struct {
	Dir               string
	Logger            *slog.Logger
	MaxAge            time.Duration
	MaxCount          int
	MaxStorageBytes   int64
	SegmentBytes      int64
	RetentionInterval time.Duration
}

// Service owns the bounded observer queue and the single writer.
type Service struct {
	opts         Options
	logger       *slog.Logger
	counters     *counters
	index        *index
	ops          fileOps
	now          func() time.Time
	flushTimeout time.Duration

	mu          sync.Mutex
	store       *store
	queue       chan Record
	pending     map[string]struct{}
	started     bool
	closed      bool
	stopped     bool
	unavailable bool
	done        chan struct{}

	reportedOverflow uint64
	reportedInvalid  uint64
	lastDropReport   time.Time
}

// New builds the records service without starting storage or the writer.
func New(opts Options) *Service {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &Service{
		opts:         opts,
		logger:       logger,
		counters:     newCounters(),
		index:        newIndex(),
		ops:          osFileOps{},
		now:          time.Now,
		flushTimeout: defaultFlushTimeout,
		queue:        make(chan Record, queueCapacity),
		pending:      make(map[string]struct{}),
		done:         make(chan struct{}),
	}
}

// Name identifies the records module in the application lifecycle.
func (s *Service) Name() string {
	return "execution-records"
}

// Start runs bounded recovery and starts the single writer. Storage failures
// degrade history but never prevent application startup.
func (s *Service) Start(context.Context) error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()

		return nil
	}
	s.started = true
	store := newStore(storageConfig{
		dir:               s.opts.Dir,
		maxAge:            s.opts.MaxAge,
		maxCount:          s.opts.MaxCount,
		maxStorageBytes:   s.opts.MaxStorageBytes,
		segmentBytes:      s.opts.SegmentBytes,
		retentionInterval: s.retentionInterval(),
	}, s.logger, s.ops, s.now, s.index, s.counters)
	s.store = store
	s.mu.Unlock()

	if err := store.start(); err != nil {
		s.mu.Lock()
		s.unavailable = true
		s.closed = true
		s.mu.Unlock()
		close(s.done)
		s.logger.Error("execution record store unavailable", "class", "store_unavailable")

		return nil
	}

	go s.runWriter()

	return nil
}

// Stop closes observer admission and drains accepted records within the fixed
// flush window. Expiry is reported as degradation and never fails shutdown.
func (s *Service) Stop(_ context.Context) error {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()

		return nil
	}
	s.stopped = true
	s.closed = true

	if !s.started {
		s.mu.Unlock()

		return nil
	}
	close(s.queue)
	s.mu.Unlock()

	timer := time.NewTimer(s.flushTimeout)
	defer timer.Stop()

	select {
	case <-s.done:
		return nil
	case <-timer.C:
		s.recordFlushExpiry()
		return nil
	}
}

// OnExecutionFinished implements execution.ResultObserver.
func (s *Service) OnExecutionFinished(finished execution.FinishedExecution) {
	record, err := recordFromExecution(finished)
	if err != nil {
		s.counters.record(failureInvalidRecord)

		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started || s.closed {
		s.counters.record(failureDroppedClosed)

		return
	}
	if _, pending := s.pending[record.ExecutionID]; pending {
		s.counters.record(failureDuplicateObservation)

		return
	}
	if _, retained := s.index.get(record.ExecutionID); retained {
		s.counters.record(failureDuplicateObservation)

		return
	}

	select {
	case s.queue <- record:
		s.pending[record.ExecutionID] = struct{}{}
	default:
		s.counters.record(failureQueueOverflow)
	}
}

// Get returns one retained record, or false when it is absent from the current
// queryable history.
func (s *Service) Get(id string) (Record, bool, error) {
	if s.unavailableIndex() {
		return Record{}, false, ErrUnavailable
	}

	record, ok := s.index.get(id)

	return record, ok, nil
}

// List returns one ordered page from the retained index.
func (s *Service) List(filter Filter) (Page, error) {
	if s.unavailableIndex() {
		return Page{}, ErrUnavailable
	}

	return s.index.page(filter), nil
}

// Degraded reports process-local evidence of lost or incompletely recovered
// records.
func (s *Service) Degraded() bool {
	return s.unavailableIndex() || s.counters.historyDegraded()
}

// Counters returns the current degradation snapshot.
func (s *Service) Counters() Counters {
	return s.counters.snapshot()
}

func (s *Service) runWriter() {
	ticker := time.NewTicker(s.retentionInterval())
	defer ticker.Stop()
	defer close(s.done)

	for {
		select {
		case record, ok := <-s.queue:
			if !ok {
				if err := s.store.close(); err != nil {
					s.logger.Warn("execution record writer close failed", "class", "close_failed")
				}

				return
			}
			s.write(record)
			s.reportDrops()
		case <-ticker.C:
			if err := s.store.trim(s.now()); err != nil {
				s.logger.Warn("execution record retention failed", "class", "retention_failed")
			}
			s.reportDrops()
		}
	}
}

func (s *Service) write(record Record) {
	if err := s.store.append(record); err != nil {
		s.logger.Warn("execution record write failed",
			"class", "append_failed",
			"execution_id", record.ExecutionID,
		)
	}

	s.mu.Lock()
	delete(s.pending, record.ExecutionID)
	s.mu.Unlock()
}

func (s *Service) recordFlushExpiry() {
	s.mu.Lock()
	pending := len(s.pending)
	s.mu.Unlock()

	s.counters.record(failureFlushTimeout)
	s.counters.addLost(pending)
	s.logger.Warn("execution record flush expired",
		"class", "flush_timeout",
		"pending", pending,
	)
}

func (s *Service) unavailableIndex() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.unavailable || s.store == nil
}

func (s *Service) retentionInterval() time.Duration {
	if s.opts.RetentionInterval <= 0 {
		return defaultRetention
	}

	return s.opts.RetentionInterval
}
