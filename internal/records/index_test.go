package records

import (
	"testing"
	"time"
)

func testIndex() *index {
	return newIndex()
}

func TestIndexRejectsDuplicateExecutionID(t *testing.T) {
	idx := testIndex()
	record := completedRecord(executionID(1), at(1))

	if !idx.add(record, 1) {
		t.Fatal("first add must be accepted")
	}

	duplicate := completedRecord(executionID(1), at(2))
	if idx.add(duplicate, 1) {
		t.Fatal("duplicate execution id must be rejected")
	}

	stored, ok := idx.get(record.ExecutionID)
	if !ok || !stored.CompletedAt.Equal(record.CompletedAt) {
		t.Fatalf("stored record = %+v, want the first accepted value", stored)
	}
}

func TestIndexPageOrdersByCompletedAtDescendingThenExecutionID(t *testing.T) {
	idx := testIndex()
	idx.add(completedRecord(executionID(1), at(1)), 1)
	idx.add(completedRecord(executionID(3), at(3)), 1)
	idx.add(completedRecord(executionID(2), at(2)), 1)

	page := idx.page(Filter{Limit: 10})

	if len(page.Records) != 3 {
		t.Fatalf("records = %d, want 3", len(page.Records))
	}

	want := []string{executionID(3), executionID(2), executionID(1)}
	for position, id := range want {
		if page.Records[position].ExecutionID != id {
			t.Fatalf("record %d = %s, want %s", position, page.Records[position].ExecutionID, id)
		}
	}
}

func TestIndexPageAppliesFilters(t *testing.T) {
	idx := testIndex()
	completed := completedRecord(executionID(1), at(1))
	completed.ProviderID = "opencode-go"
	completed.ModelID = strPtr("vendor/model")
	idx.add(completed, 1)

	failed := failedRecord(executionID(2), at(2), CauseTransport)
	failed.ProviderID = "other"
	idx.add(failed, 1)

	model := "vendor/model"

	tests := []struct {
		name   string
		filter Filter
		want   int
	}{
		{name: "provider", filter: Filter{ProviderID: "opencode-go"}, want: 1},
		{name: "provider no match", filter: Filter{ProviderID: "missing"}, want: 0},
		{name: "model", filter: Filter{ModelID: model}, want: 1},
		{name: "outcome", filter: Filter{Outcome: OutcomeFailed}, want: 1},
		{name: "after excludes older start", filter: Filter{StartedAfter: timePtr(at(1).Add(-time.Second))}, want: 1},
		{name: "before excludes newer start", filter: Filter{StartedBefore: timePtr(at(1).Add(-time.Second))}, want: 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			page := idx.page(test.filter)
			if len(page.Records) != test.want {
				t.Fatalf("records = %d, want %d", len(page.Records), test.want)
			}
		})
	}
}

func TestIndexPageUsesCursorStrictComparison(t *testing.T) {
	idx := testIndex()
	for n := 1; n <= 4; n++ {
		idx.add(completedRecord(executionID(n), at(n)), 1)
	}

	first := idx.page(Filter{Limit: 2})
	if len(first.Records) != 2 || first.NextCursor == "" {
		t.Fatalf("first page = %+v, want two records and a cursor", first)
	}

	cursor, err := decodeCursor(first.NextCursor)
	if err != nil {
		t.Fatalf("decodeCursor() returned error: %v", err)
	}

	second := idx.page(Filter{Limit: 2, Cursor: &cursor})
	if len(second.Records) != 2 {
		t.Fatalf("second page records = %d, want 2", len(second.Records))
	}

	for _, record := range second.Records {
		if record.ExecutionID == first.Records[1].ExecutionID {
			t.Fatal("cursor row must not be returned twice")
		}
	}

	if second.NextCursor != "" {
		t.Fatalf("second page next cursor = %q, want empty", second.NextCursor)
	}
}

func TestIndexPageHasNoCrossPageSnapshot(t *testing.T) {
	idx := testIndex()
	for n := 1; n <= 4; n++ {
		idx.add(completedRecord(executionID(n), at(n)), 1)
	}

	first := idx.page(Filter{Limit: 2})
	cursor, err := decodeCursor(first.NextCursor)
	if err != nil {
		t.Fatalf("decodeCursor() returned error: %v", err)
	}

	idx.add(completedRecord(executionID(5), at(5)), 1)

	second := idx.page(Filter{Limit: 2, Cursor: &cursor})
	if len(second.Records) != 2 {
		t.Fatalf("second page records = %d, want 2", len(second.Records))
	}
	for _, record := range second.Records {
		if record.ExecutionID == executionID(5) {
			t.Fatal("a record written after the first page must not appear after the cursor")
		}
	}
}

func TestIndexPageBoundsLimit(t *testing.T) {
	idx := testIndex()
	idx.add(completedRecord(executionID(1), at(1)), 1)

	if page := idx.page(Filter{Limit: 0}); len(page.Records) != 1 {
		t.Fatalf("default limit page = %d, want 1", len(page.Records))
	}

	if page := idx.page(Filter{Limit: 500}); len(page.Records) != 1 {
		t.Fatalf("clamped limit page = %d, want 1", len(page.Records))
	}
}

func TestIndexSurvivesConcurrentReadsAndWrites(t *testing.T) {
	idx := testIndex()
	done := make(chan struct{})

	go func() {
		defer close(done)
		for n := 1; n <= 100; n++ {
			idx.add(completedRecord(executionID(n), at(n)), 1)
		}
	}()

	for n := 0; n < 100; n++ {
		_ = idx.page(Filter{Limit: 10})
	}

	<-done

	if idx.len() != 100 {
		t.Fatalf("index length = %d, want 100", idx.len())
	}
}

func timePtr(value time.Time) *time.Time {
	return &value
}
