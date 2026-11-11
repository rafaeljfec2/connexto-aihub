package records

import (
	"testing"
	"time"
)

func TestIndexListAllWalksPages(t *testing.T) {
	idx := newIndex()
	now := at(0)

	for i := range MaxPageLimit + 2 {
		id := executionID(i + 1)
		completedAt := now.Add(-time.Duration(i) * time.Second)
		if !idx.add(completedRecord(id, completedAt), 0) {
			t.Fatalf("failed to add record %s", id)
		}
	}

	got, err := idx.listAll(Filter{Limit: MaxPageLimit})
	if err != nil {
		t.Fatalf("listAll() error: %v", err)
	}

	want := MaxPageLimit + 2
	if len(got) != want {
		t.Fatalf("listAll() len = %d, want %d", len(got), want)
	}
}
