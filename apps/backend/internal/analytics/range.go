package analytics

import (
	"fmt"
	"time"
)

const maxRangeDays = 366

// NormalizeRangeBounds returns inclusive UTC calendar day bounds for from/to instants.
func NormalizeRangeBounds(from, to time.Time) (start time.Time, end time.Time, err error) {
	if from.IsZero() || to.IsZero() {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: from and to are required", ErrInvalidInput)
	}
	start = utcDate(from)
	end = utcDate(to)
	if start.After(end) {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: from must not be after to", ErrInvalidInput)
	}
	days := int(end.Sub(start).Hours()/24) + 1
	if days > maxRangeDays {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: range must not exceed %d days", ErrInvalidInput, maxRangeDays)
	}
	return start, end, nil
}

func utcDate(instant time.Time) time.Time {
	utc := instant.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
}

// FillDailySeries inserts zero-count days between start and end inclusive.
func FillDailySeries(start, end time.Time, sparse []DailyAdmissionCount) []DailyAdmissionCount {
	index := make(map[string]int, len(sparse))
	for _, row := range sparse {
		key := utcDate(row.PeriodStart).Format(time.DateOnly)
		index[key] = row.AdmittedCount
	}
	out := make([]DailyAdmissionCount, 0, int(end.Sub(start).Hours()/24)+1)
	for day := start; !day.After(end); day = day.Add(24 * time.Hour) {
		key := day.Format(time.DateOnly)
		count := index[key]
		out = append(out, DailyAdmissionCount{PeriodStart: day, AdmittedCount: count})
	}
	return out
}
