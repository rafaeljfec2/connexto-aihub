package records

import "time"

// dropLogInterval bounds how often rejected records are reported while the
// writer stays busy. Reports happen outside the observer callback.
const dropLogInterval = 30 * time.Second

// reportDrops emits rate-limited warnings for records lost since the previous
// report. Only the writer goroutine calls it.
func (s *Service) reportDrops() {
	snapshot := s.counters.snapshot()
	if snapshot.QueueOverflow <= s.reportedOverflow && snapshot.InvalidRecord <= s.reportedInvalid {
		return
	}
	if !s.lastDropReport.IsZero() && s.now().Sub(s.lastDropReport) < dropLogInterval {
		return
	}

	if snapshot.QueueOverflow > s.reportedOverflow {
		s.logger.Warn("execution records dropped",
			"class", "queue_overflow",
			"dropped", snapshot.QueueOverflow-s.reportedOverflow,
		)
		s.reportedOverflow = snapshot.QueueOverflow
	}
	if snapshot.InvalidRecord > s.reportedInvalid {
		s.logger.Warn("execution records rejected",
			"class", "invalid_record",
			"dropped", snapshot.InvalidRecord-s.reportedInvalid,
		)
		s.reportedInvalid = snapshot.InvalidRecord
	}
	s.lastDropReport = s.now()
}
