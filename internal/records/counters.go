package records

import "sync"

// Counters is a process-local snapshot of record degradation diagnostics.
// Values are not persisted and never certify that older history is complete.
type Counters struct {
	AppendFailed         uint64
	RollbackFailed       uint64
	SyncUncertain        uint64
	CloseFailed          uint64
	RecoveryFailed       uint64
	RotationFailed       uint64
	RetentionFailed      uint64
	DuplicateRecord      uint64
	DuplicateObservation uint64
	InvalidRecord        uint64
	QueueOverflow        uint64
	StorageLimitReached  uint64
	DroppedClosed        uint64
	LostAtShutdown       uint64
	FlushTimeout         uint64
}

type failureKind int

const (
	failureAppend failureKind = iota
	failureRollback
	failureSync
	failureClose
	failureRecovery
	failureRotation
	failureRetention
	failureStorageLimit
	failureDuplicateRecord
	failureDuplicateObservation
	failureInvalidRecord
	failureQueueOverflow
	failureDroppedClosed
	failureFlushTimeout
)

type counters struct {
	mu       sync.Mutex
	values   Counters
	degraded bool
}

func newCounters() *counters {
	return &counters{}
}

func (c *counters) record(kind failureKind) {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch kind {
	case failureAppend:
		c.values.AppendFailed++
		c.degraded = true
	case failureRollback:
		c.values.RollbackFailed++
		c.degraded = true
	case failureSync:
		c.values.SyncUncertain++
		c.degraded = true
	case failureClose:
		c.values.CloseFailed++
		c.degraded = true
	case failureRecovery:
		c.values.RecoveryFailed++
		c.degraded = true
	case failureRotation:
		c.values.RotationFailed++
		c.degraded = true
	case failureRetention:
		c.values.RetentionFailed++
		c.degraded = true
	case failureStorageLimit:
		c.values.StorageLimitReached++
		c.degraded = true
	case failureDuplicateRecord:
		c.values.DuplicateRecord++
	case failureDuplicateObservation:
		c.values.DuplicateObservation++
	case failureInvalidRecord:
		c.values.InvalidRecord++
		c.degraded = true
	case failureQueueOverflow:
		c.values.QueueOverflow++
		c.degraded = true
	case failureDroppedClosed:
		c.values.DroppedClosed++
		c.degraded = true
	case failureFlushTimeout:
		c.values.FlushTimeout++
	}
}

func (c *counters) addLost(count int) {
	if count <= 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.values.LostAtShutdown += uint64(count)
	c.degraded = true
}

func (c *counters) snapshot() Counters {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.values
}

func (c *counters) historyDegraded() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.degraded
}
