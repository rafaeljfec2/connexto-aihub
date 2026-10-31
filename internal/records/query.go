package records

import "time"

// Page limits shared by the domain and the HTTP boundary.
const (
	DefaultPageLimit = 50
	MaxPageLimit     = 200
)

// Filter carries the validated query bounds for a page request.
type Filter struct {
	ProviderID    string
	ModelID       string
	Outcome       string
	StartedAfter  *time.Time
	StartedBefore *time.Time
	Cursor        *Cursor
	Limit         int
}

// Page is one ordered slice of retained records plus the next page boundary.
type Page struct {
	Records    []Record
	NextCursor string
}
