package records

import (
	"sort"
	"sync"
)

// index holds the retained records visible to queries.
type index struct {
	mu      sync.RWMutex
	records map[string]Record
}

func newIndex() *index {
	return &index{records: make(map[string]Record)}
}

// add retains the first value for an execution ID. The line number parameter
// keeps the recovery and append call sites explicit about their source.
func (i *index) add(record Record, _ int64) bool {
	i.mu.Lock()
	defer i.mu.Unlock()

	if _, exists := i.records[record.ExecutionID]; exists {
		return false
	}

	i.records[record.ExecutionID] = record

	return true
}

func (i *index) get(id string) (Record, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	record, ok := i.records[id]

	return record, ok
}

func (i *index) remove(id string) bool {
	i.mu.Lock()
	defer i.mu.Unlock()

	if _, exists := i.records[id]; !exists {
		return false
	}

	delete(i.records, id)

	return true
}

func (i *index) len() int {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return len(i.records)
}

func (i *index) all() []Record {
	i.mu.RLock()
	defer i.mu.RUnlock()

	records := make([]Record, 0, len(i.records))
	for _, record := range i.records {
		records = append(records, record)
	}

	return records
}

func (i *index) page(filter Filter) Page {
	matches := make([]Record, 0, i.len())

	for _, record := range i.all() {
		if filter.matches(record) {
			matches = append(matches, record)
		}
	}

	sort.Slice(matches, func(left, right int) bool {
		return recordBefore(matches[left], matches[right])
	})

	limit := pageLimit(filter.Limit)
	page := Page{Records: matches}
	if len(matches) <= limit {
		return page
	}

	page.Records = matches[:limit]
	page.NextCursor = encodeCursor(Cursor{
		CompletedAt: matches[limit-1].CompletedAt,
		ExecutionID: matches[limit-1].ExecutionID,
	})

	return page
}

func (i *index) listAll(filter Filter) ([]Record, error) {
	working := filter
	if working.Limit <= 0 {
		working.Limit = MaxPageLimit
	}

	collected := make([]Record, 0, pageLimit(working.Limit))
	for {
		page := i.page(working)
		collected = append(collected, page.Records...)
		if page.NextCursor == "" {
			return collected, nil
		}

		cursor, err := ParseCursor(page.NextCursor)
		if err != nil {
			return nil, err
		}

		working.Cursor = &cursor
	}
}

func (f Filter) matches(record Record) bool {
	if f.ProviderID != "" && record.ProviderID != f.ProviderID {
		return false
	}
	if f.ModelID != "" && (record.ModelID == nil || *record.ModelID != f.ModelID) {
		return false
	}
	if f.Outcome != "" && record.Outcome != f.Outcome {
		return false
	}
	if f.StartedAfter != nil && !record.StartedAt.After(*f.StartedAfter) {
		return false
	}
	if f.StartedBefore != nil && !record.StartedAt.Before(*f.StartedBefore) {
		return false
	}
	if f.Cursor != nil && !rowFollowsCursor(record, *f.Cursor) {
		return false
	}

	return true
}

// recordBefore orders completed_at descending, then execution_id descending.
func recordBefore(left, right Record) bool {
	if !left.CompletedAt.Equal(right.CompletedAt) {
		return left.CompletedAt.After(right.CompletedAt)
	}

	return left.ExecutionID > right.ExecutionID
}

// rowFollowsCursor keeps only tuples strictly below the cursor tuple.
func rowFollowsCursor(record Record, cursor Cursor) bool {
	if record.CompletedAt.Before(cursor.CompletedAt) {
		return true
	}
	if record.CompletedAt.Equal(cursor.CompletedAt) {
		return record.ExecutionID < cursor.ExecutionID
	}

	return false
}

func pageLimit(limit int) int {
	if limit <= 0 {
		return DefaultPageLimit
	}
	if limit > MaxPageLimit {
		return MaxPageLimit
	}

	return limit
}
